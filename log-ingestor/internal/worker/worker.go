package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/config"
	"github.com/lijuuu/Logito/log-ingestor/internal/dlq"
	"github.com/lijuuu/Logito/log-ingestor/internal/ingest"
	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
	"github.com/lijuuu/Logito/log-ingestor/internal/storage/postgres"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

type Worker struct {
	dbClient    *postgres.Client
	concurrency int
	retryCount  int
	retryDelay  time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
	dlq         dlq.DLQ
	config      *config.Config
	pool        *ingest.ObjectPool
}

func NewWorker(dbClient *postgres.Client, concurrency int, retryCount int, retryDelay time.Duration, dlq dlq.DLQ, cfg *config.Config, pool *ingest.ObjectPool) *Worker {
	return &Worker{
		dbClient:    dbClient,
		concurrency: concurrency,
		retryCount:  retryCount,
		retryDelay:  retryDelay,
		stopChan:    make(chan struct{}),
		dlq:         dlq,
		config:      cfg,
		pool:        pool,
	}
}

func (w *Worker) Start(batchChan <-chan *logentry.Batch) {
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.workerLoop(batchChan, i)
	}
}

func (w *Worker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

func (w *Worker) workerLoop(batchChan <-chan *logentry.Batch, workerID int) {
	defer w.wg.Done()

	for {
		select {
		case batch := <-batchChan:
			if batch == nil {
				return
			}
			w.processBatch(batch, workerID)
		case <-w.stopChan:
			return
		}
	}
}

func (w *Worker) processBatch(batch *logentry.Batch, workerID int) {
	if batch.IsEmpty() {
		// Return slice to pool before returning
		w.pool.PutLogEntrySlice(batch.Entries)
		return
	}

	batchSize := len(batch.Entries)
	logger.Worker(workerID, batchSize, "Processing batch")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for attempt := 0; attempt <= w.retryCount; attempt++ {
		err := w.dbClient.InsertBatch(ctx, batch.Entries)
		if err == nil {
			logger.Worker(workerID, batchSize, "Batch processed successfully")
			// Return slice to pool after successful processing
			w.pool.PutLogEntrySlice(batch.Entries)
			return
		}

		if attempt == w.retryCount {
			logger.Error("Worker-%d failed to process batch after %d attempts: %v", workerID, w.retryCount+1, err)

			// Send failed batch to DLQ
			if w.dlq != nil {
				dlqCount := 0
				for _, entry := range batch.Entries {
					reason := fmt.Sprintf("Database insert failed after %d retries: %v", w.retryCount+1, err)
					if dlqErr := dlq.SendToDLQIfEnabled(w.dlq, w.config, entry, reason, dlq.FailureTypeDBFailure); dlqErr != nil {
						logger.Error("Failed to send entry to DLQ: %v", dlqErr)
					} else {
						dlqCount++
					}
				}
				logger.Worker(workerID, batchSize, "Sent %d/%d failed entries to DLQ", dlqCount, len(batch.Entries))
			} else {
				logger.Error("DLQ not available, batch of %d entries lost", len(batch.Entries))
			}
			// Return slice to pool after DLQ processing
			w.pool.PutLogEntrySlice(batch.Entries)
			return
		}

		logger.Worker(workerID, batchSize, "Retry attempt %d/%d", attempt+1, w.retryCount+1)
		// Exponential backoff to avoid overwhelming the database
		time.Sleep(w.retryDelay * time.Duration(attempt+1))
	}
}

func (w *Worker) HealthCheck(ctx context.Context) error {
	return w.dbClient.HealthCheck(ctx)
}
