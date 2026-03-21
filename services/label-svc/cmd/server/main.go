package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/dgmmarin/etiketai/services/label-svc/internal/grpchandler"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/producthttp"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/service"
	"github.com/dgmmarin/etiketai/services/label-svc/internal/worker"
)

func main() {
	_ = godotenv.Load(".env.local")

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	pool, err := pgxpool.New(context.Background(), mustEnv("LABEL_DB_DSN"))
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("db ping failed", zap.Error(err))
	}
	logger.Info("connected to database")

	redisOpt, err := asynq.ParseRedisURI(mustEnv("REDIS_URL"))
	if err != nil {
		logger.Fatal("invalid REDIS_URL", zap.Error(err))
	}
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	labelRepo := repo.NewLabelRepo(pool)
	productRepo := repo.NewProductRepo(pool)
	labelSvc := service.NewLabelService(labelRepo, asynqClient, logger)

	// ─── Product HTTP server ───────────────────────────────────────────────────
	productPort := getEnvOr("PRODUCT_PORT", "8088")
	productH := producthttp.NewHandler(productRepo, logger)
	productSrv := &http.Server{
		Addr:         fmt.Sprintf(":%s", productPort),
		Handler:      producthttp.NewMux(productH),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	port := getEnvOr("PORT", "8083")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Fatal("listen failed", zap.Error(err))
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(loggingInterceptor(logger)))
	grpchandler.Register(grpcServer, labelSvc)

	if os.Getenv("ENV") != "production" {
		reflection.Register(grpcServer)
	}

	taskHandler, err := worker.NewHandler(labelRepo, getEnvOr("AGENT_SVC_ADDR", "localhost:8084"), logger)
	if err != nil {
		logger.Fatal("failed to init task handler", zap.Error(err))
	}

	asynqServer := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 5,
		Queues:      map[string]int{"default": 1},
	})
	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TaskProcessLabel, taskHandler.ProcessLabel)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("label-svc gRPC starting", zap.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("label-svc asynq worker starting")
		if err := asynqServer.Run(mux); err != nil {
			logger.Fatal("asynq worker error", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("label-svc product HTTP starting", zap.String("port", productPort))
		if err := productSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("product HTTP error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down label-svc...")
	grpcServer.GracefulStop()
	asynqServer.Shutdown()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = productSrv.Shutdown(shutCtx)
	logger.Info("label-svc stopped")
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

func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		logger.Info("gRPC",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return resp, err
	}
}
