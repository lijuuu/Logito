package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/api"
	"github.com/lijuuu/Logito/query-interface/internal/config"
	"github.com/lijuuu/Logito/query-interface/internal/indexer"
	"github.com/lijuuu/Logito/query-interface/internal/storage/postgres"
	"github.com/lijuuu/Logito/query-interface/internal/worker"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

func main() {
	log.Println("Starting Logito Query Interface Server...")

	// load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: failed to load .env file: %v", err)
	}

	// load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Printf("Configuration loaded successfully")

	// initialize database client
	log.Println("Initializing database client...")
	dbClient, err := postgres.NewClient(postgres.Config{
		Host:            cfg.Postgres.Host,
		Port:            cfg.Postgres.Port,
		User:            cfg.Postgres.User,
		Password:        cfg.Postgres.Password,
		DBName:          cfg.Postgres.DBName,
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
	})
	if err != nil {
		log.Fatalf("failed to create database client: %v", err)
	}
	defer dbClient.Close()
	log.Println("Database client initialized successfully")

	// initialize elasticsearch client
	log.Println("Initializing Elasticsearch client...")
	esClient, err := indexer.NewESClient(
		cfg.Elasticsearch.Host,
		cfg.Elasticsearch.Index,
		cfg.Elasticsearch.Timeout,
	)
	if err != nil {
		log.Fatalf("failed to create elasticsearch client: %v", err)
	}
	log.Println("Elasticsearch client initialized successfully")

	// create elasticsearch index
	log.Println("Creating Elasticsearch index...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := esClient.CreateIndex(ctx); err != nil {
		log.Fatalf("failed to create elasticsearch index: %v", err)
	}
	log.Println("Elasticsearch index created successfully")

	// initialize indexer components
	fetcher := indexer.NewFetcher(dbClient)
	marker := indexer.NewMarker(dbClient)

	// initialize index worker
	indexWorker := worker.NewIndexWorker(
		esClient,
		fetcher,
		marker,
		worker.Config{
			WorkerCount:       cfg.Indexer.WorkerCount,
			FetchInterval:     cfg.Indexer.FetchInterval,
			BatchSize:         cfg.Indexer.BatchSize,
			MaxRetries:        cfg.Indexer.MaxRetries,
			ProcessingTimeout: cfg.Indexer.ProcessingTimeout,
			RetryDelay:        1 * time.Second,
		},
	)

	// start index worker
	indexWorker.Start()

	// initialize http handler
	log.Println("Initializing HTTP handler...")
	handler := api.NewHandler(esClient, indexWorker, fetcher)

	// setup gin router
	log.Println("Setting up HTTP router...")
	router := gin.Default()

	// add middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// add cors middleware for frontend
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

	// setup routes
	log.Println("Setting up API routes...")
	router.GET("/search", handler.Search)
	router.GET("/metadata", handler.GetMetadata)
	router.GET("/counts", handler.GetCounts)
	router.GET("/logs/:id", handler.GetLogEntry)
	router.GET("/health", handler.HealthCheck)
	router.GET("/sync-status", handler.GetSyncStatus)

	// create http server
	srv := &http.Server{
		Addr:    ":4000",
		Handler: router,
	}

	// start server in goroutine
	go func() {
		log.Printf("Starting query interface server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down query interface server...")

	// graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// stop index worker
	log.Println("Stopping index worker...")
	indexWorker.Stop()
	log.Println("Index worker stopped")

	// shutdown http server
	log.Println("Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("HTTP server shutdown gracefully")
	}

	log.Println("Query interface server exited")
}

// loadConfig loads configuration from yaml file
func loadConfig() (*config.Config, error) {
	// determine config file based on environment
	configFile := "configs/local.yml"
	if env := os.Getenv("ENV"); env == "prod" {
		configFile = "configs/prod.yml"
	}

	// read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	// parse yaml
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
