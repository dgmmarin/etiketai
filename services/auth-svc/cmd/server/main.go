package main

import (
	"context"
	"fmt"
	"log"
	"net"
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

	"github.com/dgmmarin/etiketai/services/auth-svc/internal/grpchandler"
	"github.com/dgmmarin/etiketai/services/auth-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/auth-svc/internal/service"
)

func main() {
	// Load .env.local for dev
	_ = godotenv.Load(".env.local")

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	// ─── Database ─────────────────────────────────────────────────────────────
	dsn := mustEnv("AUTH_DB_DSN")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("database ping failed", zap.Error(err))
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

	// ─── Repositories ─────────────────────────────────────────────────────────
	userRepo := repo.NewUserRepo(pool)
	tokenRepo := repo.NewTokenRepo(pool, redisClient)

	// ─── Service ──────────────────────────────────────────────────────────────
	cfg := service.Config{
		JWTSecret:          mustEnv("JWT_SECRET"),
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTLDays: 30,
		ResendAPIKey:       os.Getenv("RESEND_API_KEY"),
		EmailFrom:          getEnvOr("EMAIL_FROM", "noreply@etiketai.ro"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	}
	authService := service.NewAuthService(cfg, userRepo, tokenRepo, logger)

	// ─── gRPC server ──────────────────────────────────────────────────────────
	port := getEnvOr("PORT", "8081")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)

	// Register gRPC service (generated code stub — wired after proto gen)
	grpchandler.Register(grpcServer, authService)

	// Enable reflection for grpcurl in dev
	if os.Getenv("ENV") != "production" {
		reflection.Register(grpcServer)
	}

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("auth-svc starting", zap.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down auth-svc...")
	grpcServer.GracefulStop()
	logger.Info("auth-svc stopped")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
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
		logger.Info("gRPC request",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return resp, err
	}
}
