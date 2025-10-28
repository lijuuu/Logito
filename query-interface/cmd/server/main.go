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

	"github.com/lijuuu/Logito/query-interface/internal/api"
	"github.com/lijuuu/Logito/query-interface/internal/auth"
	"github.com/lijuuu/Logito/query-interface/internal/config"
	"github.com/lijuuu/Logito/query-interface/internal/dlq"
	"github.com/lijuuu/Logito/query-interface/internal/indexer"
	"github.com/lijuuu/Logito/query-interface/internal/logger"
	"github.com/lijuuu/Logito/query-interface/internal/storage/postgres"
	"github.com/lijuuu/Logito/query-interface/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

func main() {
	logger.Init("Starting Logito Query Interface Server...")

	if err := godotenv.Load(); err != nil {
		logger.Info("Warning: failed to load .env file: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
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
		logger.Error("Failed to create database client: %v", err)
		log.Fatalf("failed to create database client: %v", err)
	}
	defer dbClient.Close()
	logger.Database("Connected to PostgreSQL at %s:%d", cfg.Database.Postgres.Host, cfg.Database.Postgres.Port)

	esClient, err := indexer.NewESClient(
		cfg.QueryInterface.Elasticsearch.Connection.Host,
		cfg.QueryInterface.Elasticsearch.Connection.Index,
		cfg.QueryInterface.Elasticsearch.Connection.Timeout,
		indexer.ElasticsearchPoolConfig{
			MaxIdleConns:    cfg.QueryInterface.Elasticsearch.Connection.ConnectionPool.MaxIdleConns,
			MaxConnsPerHost: cfg.QueryInterface.Elasticsearch.Connection.ConnectionPool.MaxConnsPerHost,
			IdleConnTimeout: cfg.QueryInterface.Elasticsearch.Connection.ConnectionPool.IdleConnTimeout,
		},
		indexer.ElasticsearchHealthConfig{
			CheckMinHealth:     cfg.QueryInterface.Elasticsearch.Connection.HealthCheck.CheckMinHealth,
			MinHealthThreshold: cfg.QueryInterface.Elasticsearch.Connection.HealthCheck.MinHealthThreshold,
			HealthCheckTimeout: cfg.QueryInterface.Elasticsearch.Connection.HealthCheck.HealthCheckTimeout,
		},
	)
	if err != nil {
		logger.Error("Failed to create elasticsearch client: %v", err)
		log.Fatalf("failed to create elasticsearch client: %v", err)
	}
	logger.Elasticsearch("Connected to Elasticsearch at %s", cfg.QueryInterface.Elasticsearch.Connection.Host)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := esClient.CreateIndex(ctx); err != nil {
		logger.Error("Failed to create elasticsearch index: %v", err)
		log.Fatalf("failed to create elasticsearch index: %v", err)
	}
	logger.Elasticsearch("Index '%s' created successfully", cfg.QueryInterface.Elasticsearch.Connection.Index)

	fetcher := indexer.NewFetcher(dbClient)
	marker := indexer.NewMarker(dbClient)
	logger.Init("Indexer components initialized")

	indexWorker := worker.NewIndexWorker(
		esClient,
		fetcher,
		marker,
		worker.Config{
			WorkerCount:       cfg.QueryInterface.Elasticsearch.Indexer.Workers.Count,
			FetchInterval:     cfg.QueryInterface.Elasticsearch.Indexer.Workers.FetchInterval,
			BatchSize:         cfg.QueryInterface.Elasticsearch.Indexer.Workers.BatchSize,
			MaxRetries:        cfg.QueryInterface.Elasticsearch.Indexer.Retry.MaxRetries,
			ProcessingTimeout: cfg.QueryInterface.Elasticsearch.Indexer.Retry.ProcessingTimeout,
			RetryDelay:        cfg.QueryInterface.Elasticsearch.Indexer.Retry.Delay,
		},
	)
	logger.Init("Index worker initialized - Workers: %d, BatchSize: %d, FetchInterval: %v",
		cfg.QueryInterface.Elasticsearch.Indexer.Workers.Count, cfg.QueryInterface.Elasticsearch.Indexer.Workers.BatchSize, cfg.QueryInterface.Elasticsearch.Indexer.Workers.FetchInterval)

	//Wait for logs table to exist before starting indexer
	logger.Init("Waiting for logs table to be created...")
	waitForLogsTable(dbClient)
	logger.Init("Logs table found, starting indexer...")

	indexWorker.Start()

	// Initialize DLQ client
	dlqClient, err := dlq.NewDLQClient(cfg, dbClient)
	if err != nil {
		logger.Error("Failed to initialize DLQ client: %v", err)
		log.Fatalf("failed to initialize DLQ client: %v", err)
	}
	if dlqClient != nil {
		logger.Init("DLQ client initialized successfully")
	} else {
		logger.Init("DLQ client disabled")
	}

	// Initialize auth service
	authService := auth.NewAuthService(cfg)
	logger.Init("Auth service initialized")

	handler := api.NewHandler(esClient, indexWorker, fetcher, dlqClient, authService, cfg)

	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	logger.Init("Setting up API routes...")

	// Public routes (no auth required)
	router.POST("/auth/login", handler.Login)
	router.GET("/health", handler.ESHealthCheck)

	// All routes require admin authentication
	allRoutes := router.Group("/")
	allRoutes.Use(auth.AuthMiddleware(authService))
	allRoutes.Use(auth.RequireRole(auth.RoleAdmin))

	// All endpoints now require admin authentication
	allRoutes.GET("/search", handler.Search)
	allRoutes.GET("/metadata", handler.GetMetadata)
	allRoutes.GET("/counts", handler.GetCounts)
	allRoutes.GET("/logs/:id", handler.GetLogEntry)
	allRoutes.GET("/profile", handler.GetProfile)

	//indexing
	allRoutes.GET("/sync-status", handler.GetSyncStatus)
	allRoutes.POST("/reset-indexing", handler.ResetIndexingStatus)

	//dlq
	allRoutes.GET("/dlq/count", handler.GetDLQCount)
	allRoutes.GET("/dlq/messages", handler.GetDLQMessages)
	allRoutes.POST("/dlq/force-add-all", handler.ForceAddAllDLQMessages)
	allRoutes.DELETE("/dlq/clear", handler.ClearDLQ)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.QueryInterface.Server.Port),
		Handler: router,
	}

	go func() {
		logger.Init("Starting query interface server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server: %v", err)
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Init("Shutting down query interface server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Init("Stopping index worker...")
	indexWorker.Stop()
	logger.Init("Index worker stopped")

	logger.Init("Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	} else {
		logger.Init("HTTP server shutdown gracefully")
	}

	logger.Init("Query interface server exited")
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

// waitForLogsTable waits for the logs table to be created by log-ingestor,
// only start indexing after that
func waitForLogsTable(dbClient *postgres.Client) {
	ctx := context.Background()
	maxRetries := 60
	retryInterval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		var tableExists bool
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'logs'
			)
		`

		err := dbClient.GetPool().QueryRow(ctx, query).Scan(&tableExists)
		if err == nil && tableExists {
			logger.Init("Logs table found after %d attempts", i+1)
			return
		}

		if i < maxRetries-1 {
			logger.Init("Logs table not found, waiting... (attempt %d/%d)", i+1, maxRetries)
			time.Sleep(retryInterval)
		}
	}

	logger.Error("Timeout waiting for logs table to be created after %d attempts", maxRetries)
	log.Fatalf("logs table was not created within the expected time")
}
