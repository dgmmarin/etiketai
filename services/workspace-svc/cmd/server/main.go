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
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/billing"
	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/grpchandler"
	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/workspace-svc/internal/service"
	stripe "github.com/dgmmarin/etiketai/services/workspace-svc/internal/stripe"
)

func main() {
	_ = godotenv.Load(".env.local")

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	// ─── Database ─────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), mustEnv("WORKSPACE_DB_DSN"))
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("db ping failed", zap.Error(err))
	}
	logger.Info("connected to database")

	// ─── Repos & service ──────────────────────────────────────────────────────
	workspaceRepo := repo.NewWorkspaceRepo(pool)
	workspaceSvc := service.NewWorkspaceService(workspaceRepo, logger)

	// ─── Stripe ───────────────────────────────────────────────────────────────
	stripeCl := stripe.NewClient(stripe.Config{
		SecretKey:       os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret:   os.Getenv("STRIPE_WEBHOOK_SECRET"),
		PriceStarter:    os.Getenv("STRIPE_PRICE_STARTER"),
		PriceBusiness:   os.Getenv("STRIPE_PRICE_BUSINESS"),
		PriceEnterprise: os.Getenv("STRIPE_PRICE_ENTERPRISE"),
	})

	// ─── Billing HTTP server ───────────────────────────────────────────────────
	billingH := billing.New(stripeCl, workspaceRepo, logger)
	billingMux := http.NewServeMux()
	billing.RegisterMux(billingMux, billingH)

	billingPort := getEnvOr("BILLING_PORT", "8092")
	billingSrv := &http.Server{
		Addr:         fmt.Sprintf(":%s", billingPort),
		Handler:      billingMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	go func() {
		logger.Info("billing HTTP starting", zap.String("port", billingPort))
		if err := billingSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("billing server error", zap.Error(err))
		}
	}()

	// ─── Subscription expiry cron ──────────────────────────────────────────────
	go runExpiryCron(workspaceRepo, logger)

	// ─── gRPC server ──────────────────────────────────────────────────────────
	port := getEnvOr("PORT", "8082")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Fatal("listen failed", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)
	grpchandler.Register(grpcServer, workspaceSvc)

	if os.Getenv("ENV") != "production" {
		reflection.Register(grpcServer)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("workspace-svc starting", zap.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down workspace-svc...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = billingSrv.Shutdown(ctx)
	grpcServer.GracefulStop()
	logger.Info("workspace-svc stopped")
}

// runExpiryCron checks every 12 hours for subscriptions expiring within 7 days
// and logs a warning (notifications are enqueued via notification-svc in production).
func runExpiryCron(wsRepo *repo.WorkspaceRepo, logger *zap.Logger) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(7 * 24 * time.Hour)
		expiring, err := wsRepo.ListExpiringSubscriptions(context.Background(), cutoff)
		if err != nil {
			logger.Error("expiry cron: list expiring", zap.Error(err))
			continue
		}
		for _, ws := range expiring {
			daysLeft := int(time.Until(cutoff).Hours() / 24)
			logger.Warn("subscription expiring soon",
				zap.String("workspace_id", ws.ID),
				zap.String("plan", ws.Plan),
				zap.Int("days_left", daysLeft),
			)
		}
	}
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
