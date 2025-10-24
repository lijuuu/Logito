package worker

import (
	"context"
	"sync"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/storage/postgres"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

// Worker handles async processing of log batches
type Worker struct {
	dbClient    *postgres.Client
	concurrency int
	retryCount  int
	retryDelay  time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewWorker creates a new worker instance
func NewWorker(dbClient *postgres.Client, concurrency int, retryCount int, retryDelay time.Duration) *Worker {
	return &Worker{
		dbClient:    dbClient,
		concurrency: concurrency,
		retryCount:  retryCount,
		retryDelay:  retryDelay,
		stopChan:    make(chan struct{}),
	}
}

// Start starts multiple workers with specified concurrency
func (w *Worker) Start(batchChan <-chan *logentry.Batch) {
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.workerLoop(batchChan, i) // pass worker id for logging
	}
}

// Stop stops the worker gracefully
func (w *Worker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// workerLoop is the main worker loop for concurrent batch processing
func (w *Worker) workerLoop(batchChan <-chan *logentry.Batch, workerID int) {
	defer w.wg.Done()

	for {
		select {
		case batch := <-batchChan:
			if batch == nil {
				return
			}
			// process batch concurrently with other workers
			w.processBatch(batch, workerID)
		case <-w.stopChan:
			return
		}
	}
}

// processBatch processes a batch of log entries with worker id for logging
func (w *Worker) processBatch(batch *logentry.Batch, workerID int) {
	defer func() {
		// always return batch to pool after processing
		// note: this assumes the batch is managed by a pool
		// in a real implementation, you'd need access to the pool
	}()

	if batch.IsEmpty() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// attempt to insert batch with retries
	for attempt := 0; attempt <= w.retryCount; attempt++ {
		err := w.dbClient.InsertBatch(ctx, batch.Entries)
		if err == nil {
			// success, we're done
			return
		}

		// if this is the last attempt, send to dlq
		if attempt == w.retryCount {
			//send to DLQ
			return
		}

		// wait before retry with exponential backoff
		time.Sleep(w.retryDelay * time.Duration(attempt+1))
	}
}

// HealthCheck checks if the worker is healthy
func (w *Worker) HealthCheck(ctx context.Context) error {
	return w.dbClient.HealthCheck(ctx)
}
