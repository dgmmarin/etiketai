package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dgmmarin/etiketai/services/api-gateway/internal/handlers"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/middleware"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/proxy"
	"github.com/dgmmarin/etiketai/services/api-gateway/internal/router"
)

func main() {
	_ = godotenv.Load(".env.local")

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// ─── gRPC clients ─────────────────────────────────────────────────────────
	authConn := mustDial(mustEnv("AUTH_SVC_ADDR"), logger)
	defer authConn.Close()

	workspaceConn := mustDial(mustEnv("WORKSPACE_SVC_ADDR"), logger)
	defer workspaceConn.Close()

	labelConn := mustDial(mustEnv("LABEL_SVC_ADDR"), logger)
	defer labelConn.Close()

	agentConn := mustDial(mustEnv("AGENT_SVC_ADDR"), logger)
	defer agentConn.Close()

	printSvcURL    := getEnvOr("PRINT_SVC_URL", "http://localhost:8085")
	agentAdminURL  := getEnvOr("AGENT_ADMIN_URL", "http://localhost:9084")
	productSvcURL  := getEnvOr("PRODUCT_SVC_URL", "http://localhost:8088")
	billingSvcURL  := getEnvOr("BILLING_SVC_URL", "http://localhost:8092")

	// ─── Proxy clients ────────────────────────────────────────────────────────
	proxies := proxy.NewClients(authConn, workspaceConn, labelConn, agentConn, printSvcURL, agentAdminURL, productSvcURL, billingSvcURL)

	// ─── S3 client (for upload pre-processing) ────────────────────────────────
	s3Cfg := handlers.S3Config{
		Endpoint:  getEnvOr("S3_ENDPOINT", "http://localhost:9000"),
		AccessKey: mustEnv("S3_ACCESS_KEY"),
		SecretKey: mustEnv("S3_SECRET_KEY"),
		Bucket:    getEnvOr("S3_BUCKET", "etiketai"),
		Region:    getEnvOr("S3_REGION", "us-east-1"),
	}

	// ─── Middleware config ────────────────────────────────────────────────────
	mwCfg := middleware.Config{
		JWTSecret: mustEnv("JWT_SECRET"),
	}

	// ─── HTTP router ──────────────────────────────────────────────────────────
	r := router.New(proxies, s3Cfg, mwCfg, logger)

	port := getEnvOr("PORT", "8080")
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("api-gateway starting", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down api-gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}
	logger.Info("api-gateway stopped")
}

func mustDial(addr string, logger *zap.Logger) *grpc.ClientConn {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("failed to dial gRPC service", zap.String("addr", addr), zap.Error(err))
	}
	return conn
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %q not set", key)
	}
	return v
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
