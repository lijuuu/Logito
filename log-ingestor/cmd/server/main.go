package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/config"
	"github.com/lijuuu/Logito/log-ingestor/internal/ingest"
	"github.com/lijuuu/Logito/log-ingestor/internal/migration"
	"github.com/lijuuu/Logito/log-ingestor/internal/storage/postgres"
	"github.com/lijuuu/Logito/log-ingestor/internal/worker"

	"github.com/gin-gonic/gin"
	//"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

func main() {
	// load environment variables (commented out)
	// if err := godotenv.Load(); err != nil {
	// 	log.Printf("warning: failed to load .env file: %v", err)
	// }

	// load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// initialize database client
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

	// run database migrations
	if err := migration.Migrate(dbClient.GetPool()); err != nil {
		log.Fatalf("failed to run database migrations: %v", err)
	}
	log.Println("database migrations completed successfully")


	// initialize object pool for memory reuse
	pool := ingest.NewObjectPool(cfg.Batcher.MaxBatchCount)

	// initialize batcher
	batcher := ingest.NewBatcher(
		cfg.Batcher.MaxBatchSize,
		cfg.Batcher.MaxBatchCount,
		cfg.Batcher.FlushInterval,
		pool,
	)

	// initialize worker
	worker := worker.NewWorker(
		dbClient,
		cfg.Worker.Concurrency,
		3, // retry count
		cfg.Worker.RetryInterval,
	)

	// start worker
	worker.Start(batcher.GetWorkerChan())

	// initialize http handler
	handler := ingest.NewHandler(batcher, pool)

	// setup gin router
	router := gin.Default()

	// add middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// setup routes
	router.POST("/logs", handler.IngestLogs)
	router.GET("/health", handler.HealthCheck)
	router.GET("/quick-stats", handler.GetQuickStats)
	router.GET("/stats", handler.GetStats)

	// create http server
	srv := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	// start server in goroutine
	go func() {
		log.Printf("starting server on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// start profiling server
	go func() {
		log.Println("profiling server started on :6060")
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	// wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	// graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// stop batcher
	batcher.Stop()

	// stop worker
	worker.Stop()

	// shutdown http server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
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
