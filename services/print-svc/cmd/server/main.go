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

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/dgmmarin/etiketai/services/print-svc/internal/handler"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/repo"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/service"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/storage"
	"github.com/dgmmarin/etiketai/services/print-svc/internal/worker"
)

func main() {
	_ = godotenv.Load(".env.local")

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ─── Database ─────────────────────────────────────────────────────────────
	dsn := getEnvOr("PRINT_DB_DSN", "postgres://postgres:postgres@localhost:5432/print_db?sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Fatal("db connect", zap.Error(err))
	}
	defer pool.Close()

	jobs := repo.NewPrintJobRepo(pool)

	// ─── S3 ───────────────────────────────────────────────────────────────────
	s3cfg := storage.Config{
		Endpoint:  getEnvOr("S3_ENDPOINT", "http://localhost:9000"),
		AccessKey: getEnvOr("S3_ACCESS_KEY", "minioadmin"),
		SecretKey: getEnvOr("S3_SECRET_KEY", "minioadmin"),
		Bucket:    getEnvOr("S3_BUCKET", "etiketai"),
		Region:    getEnvOr("S3_REGION", "us-east-1"),
	}
	s3Client, err := storage.NewS3Client(ctx, s3cfg)
	if err != nil {
		logger.Warn("s3 client init failed — PDFs will not be uploaded", zap.Error(err))
	}

	// ─── Asynq queue client + server ──────────────────────────────────────────
	redisURL := getEnvOr("REDIS_URL", "redis://localhost:6379")
	asynqOpt, _ := asynq.ParseRedisURI(redisURL)
	queue := asynq.NewClient(asynqOpt)
	defer queue.Close()

	workerHandler := worker.NewHandler(jobs, s3Client, logger)
	mux := asynq.NewServeMux()
	worker.RegisterMux(mux, workerHandler)

	asynqServer := worker.NewAsynqServer(redisURL)

	// ─── Service + HTTP handler ────────────────────────────────────────────────
	productSvcURL := getEnvOr("PRODUCT_SVC_URL", "http://localhost:8088")
	svc := service.NewPrintService(jobs, queue, s3Client, productSvcURL, logger)
	printH := handler.NewPrintHandler(svc, logger)

	muxHTTP := http.NewServeMux()
	muxHTTP.HandleFunc("GET /health", handler.Health)
	muxHTTP.HandleFunc("POST /jobs", printH.CreateJob)
	muxHTTP.HandleFunc("GET /jobs/{id}", printH.GetJob)
	muxHTTP.HandleFunc("GET /jobs/{id}/pdf-url", printH.GetReprintURL)

	port := getEnvOr("PORT", "8085")
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      muxHTTP,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start Asynq worker
	go func() {
		logger.Info("print-svc asynq worker starting")
		if err := asynqServer.Run(mux); err != nil {
			logger.Fatal("asynq worker error", zap.Error(err))
		}
	}()

	// Start HTTP server
	go func() {
		logger.Info("print-svc HTTP starting", zap.String("port", port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down print-svc...")
	asynqServer.Shutdown()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = httpServer.Shutdown(shutCtx)
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
