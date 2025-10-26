package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/config"
	"github.com/lijuuu/Logito/log-ingestor/internal/dlq"
	"github.com/lijuuu/Logito/log-ingestor/internal/ingest"
	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
	"github.com/lijuuu/Logito/log-ingestor/internal/migration"
	"github.com/lijuuu/Logito/log-ingestor/internal/storage/postgres"
	"github.com/lijuuu/Logito/log-ingestor/internal/worker"

	"github.com/gin-gonic/gin"
	//"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logger.Init("Configuration loaded successfully")

	dbClient, err := postgres.NewClient(postgres.Config{
		Host:            cfg.Database.Postgres.Host,
		Port:            cfg.Database.Postgres.Port,
		User:            cfg.Database.Postgres.User,
		Password:        cfg.Database.Postgres.Password,
		DBName:          cfg.Database.Postgres.DBName,
		MaxOpenConns:    cfg.Database.Postgres.ConnectionPool.MaxOpenConns,
		MaxIdleConns:    cfg.Database.Postgres.ConnectionPool.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.Postgres.ConnectionPool.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.Postgres.ConnectionPool.ConnMaxIdleTime,
	})
	if err != nil {
		log.Fatalf("failed to create database client: %v", err)
	}
	defer dbClient.Close()
	logger.Database("Connected to PostgreSQL at %s:%d", cfg.Database.Postgres.Host, cfg.Database.Postgres.Port)

	if err := migration.Migrate(dbClient.GetPool()); err != nil {
		log.Fatalf("failed to run database migrations: %v", err)
	}
	logger.Database("Migrations completed successfully")

	dlqClient := dlq.NewDLQ(cfg)
	if dlqClient != nil {
		logger.Init("DLQ initialized successfully")
	} else {
		logger.Init("DLQ disabled")
	}

	pool := ingest.NewObjectPool(cfg.LogIngestor.Processing.Batcher.MaxBatchCount)
	logger.Init("Object pool initialized with capacity: %d", cfg.LogIngestor.Processing.Batcher.MaxBatchCount)

	batcher := ingest.NewBatcher(
		cfg.LogIngestor.Processing.Batcher.MaxBatchSize,
		cfg.LogIngestor.Processing.Batcher.MaxBatchCount,
		cfg.LogIngestor.Processing.Batcher.FlushInterval,
		pool,
		dlqClient,
		cfg,
	)
	logger.Init("Batcher initialized - MaxBatchSize: %d, MaxBatchCount: %d, FlushInterval: %v",
		cfg.LogIngestor.Processing.Batcher.MaxBatchSize, cfg.LogIngestor.Processing.Batcher.MaxBatchCount, cfg.LogIngestor.Processing.Batcher.FlushInterval)

	worker := worker.NewWorker(
		dbClient,
		cfg.LogIngestor.Processing.Workers.Concurrency,
		cfg.LogIngestor.Processing.Workers.RetryCount,
		cfg.LogIngestor.Processing.Workers.RetryInterval,
		dlqClient,
		cfg,
	)
	logger.Init("Worker initialized with concurrency: %d", cfg.LogIngestor.Processing.Workers.Concurrency)

	worker.Start(batcher.GetWorkerChan())

	handler := ingest.NewHandler(batcher, pool, dlqClient, cfg)

	router := gin.Default()

	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	router.POST("/logs", handler.IngestLogs)
	router.GET("/health", handler.HealthCheck)
	router.GET("/quick-stats", handler.GetQuickStats)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.LogIngestor.Server.Port),
		Handler: router,
	}

	go func() {
		logger.Init("Starting HTTP server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server: %v", err)
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	go func() {
		logger.Init("Profiling server started on :6060")
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Init("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batcher.Stop()
	worker.Stop()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Init("Server exited")
}

func loadConfig() (*config.Config, error) {
	configFile := "./config.yaml"
	if env := os.Getenv("ENV"); env == "prod" {
		configFile = "./config.prod.yaml"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
