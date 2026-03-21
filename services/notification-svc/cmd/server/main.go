package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/notification-svc/internal/email"
	"github.com/dgmmarin/etiketai/services/notification-svc/internal/worker"
)

func main() {
	_ = godotenv.Load(".env.local")

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	// ─── Email client ──────────────────────────────────────────────────────────
	emailClient := email.NewClient(
		getEnvOr("RESEND_API_KEY", ""),
		getEnvOr("EMAIL_FROM", "EtiketAI <noreply@etiketai.ro>"),
	)
	if !emailClient.IsConfigured() {
		logger.Warn("RESEND_API_KEY not set — emails will be dropped (dry-run mode)")
	}

	// ─── Asynq server ─────────────────────────────────────────────────────────
	redisURL := getEnvOr("REDIS_URL", "redis://localhost:6379")
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		logger.Fatal("parse redis url", zap.Error(err))
	}

	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: 5,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
	})

	mux := asynq.NewServeMux()
	h := worker.NewHandler(emailClient, logger)
	worker.RegisterMux(mux, h)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("notification-svc starting", zap.String("redis", redisURL))
		if err := srv.Run(mux); err != nil {
			logger.Fatal("asynq server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down notification-svc...")
	srv.Shutdown()
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
