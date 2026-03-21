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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/dgmmarin/etiketai/services/agent-svc/internal/adminhttp"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/agentfactory"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/crypto"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/grpchandler"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/service"
	"github.com/dgmmarin/etiketai/services/agent-svc/internal/storage"
)

func main() {
	_ = godotenv.Load(".env.local")

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	// ─── Database ─────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), mustEnv("AGENT_DB_DSN"))
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("db ping failed", zap.Error(err))
	}
	logger.Info("connected to database")

	// ─── Redis ────────────────────────────────────────────────────────────────
	redisOpts, err := redis.ParseURL(mustEnv("REDIS_URL"))
	if err != nil {
		logger.Fatal("invalid REDIS_URL", zap.Error(err))
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("redis ping failed", zap.Error(err))
	}
	logger.Info("connected to redis")

	// ─── S3 ───────────────────────────────────────────────────────────────────
	s3Client, err := storage.NewS3Client(context.Background(), storage.S3Config{
		Endpoint:  getEnvOr("S3_ENDPOINT", "http://localhost:9000"),
		AccessKey: mustEnv("S3_ACCESS_KEY"),
		SecretKey: mustEnv("S3_SECRET_KEY"),
		Bucket:    getEnvOr("S3_BUCKET", "etiketai"),
		Region:    getEnvOr("S3_REGION", "us-east-1"),
	})
	if err != nil {
		logger.Fatal("failed to init S3 client", zap.Error(err))
	}

	// ─── KMS ──────────────────────────────────────────────────────────────────
	var kms *crypto.KMS
	if hexKey := os.Getenv("ENCRYPTION_KEY"); hexKey != "" {
		kms, err = crypto.NewKMS(hexKey)
		if err != nil {
			logger.Fatal("invalid ENCRYPTION_KEY", zap.Error(err))
		}
	} else {
		logger.Warn("ENCRYPTION_KEY not set — API keys will be stored unencrypted")
	}

	// ─── Repositories ─────────────────────────────────────────────────────────
	configRepo := repo.NewAgentConfigRepo(pool, kms)
	logsRepo := repo.NewCallLogRepo(pool)

	// ─── Factory ──────────────────────────────────────────────────────────────
	cacheTTL := time.Duration(getEnvInt("AGENT_CACHE_TTL_SECONDS", 300)) * time.Second
	factory := agentfactory.NewFactory(
		configRepo,
		redisClient,
		s3Client,
		os.Getenv("ANTHROPIC_API_KEY"),
		cacheTTL,
		logger,
	)

	// ─── Service ──────────────────────────────────────────────────────────────
	agentSvc := service.NewAgentService(factory, logsRepo, logger)

	// ─── gRPC server ──────────────────────────────────────────────────────────
	port := getEnvOr("PORT", "8084")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Fatal("listen failed", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)
	grpchandler.Register(grpcServer, agentSvc, configRepo)

	if os.Getenv("ENV") != "production" {
		reflection.Register(grpcServer)
	}

	// ─── Internal admin HTTP server ───────────────────────────────────────────
	adminPort := getEnvOr("ADMIN_PORT", "9084")
	adminH := adminhttp.NewHandler(logsRepo, logger)
	adminSrv := &http.Server{
		Addr:         fmt.Sprintf(":%s", adminPort),
		Handler:      adminhttp.NewMux(adminH),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("agent-svc starting", zap.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("agent-svc admin HTTP starting", zap.String("admin_port", adminPort))
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("admin HTTP error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down agent-svc...")
	grpcServer.GracefulStop()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = adminSrv.Shutdown(shutCtx)
	logger.Info("agent-svc stopped")
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

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var i int
	if _, err := fmt.Sscanf(v, "%d", &i); err != nil {
		return fallback
	}
	return i
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
