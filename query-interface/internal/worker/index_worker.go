package worker

import (
	"context"
	"sync"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/indexer"
	"github.com/lijuuu/Logito/query-interface/internal/logger"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"
)

type IndexWorker struct {
	esClient   *indexer.ESClient
	fetcher    *indexer.Fetcher
	marker     *indexer.Marker
	config     Config
	stopChan   chan struct{}
	wg         sync.WaitGroup
	fetchMutex sync.Mutex
}

type Config struct {
	WorkerCount       int
	FetchInterval     time.Duration
	BatchSize         int
	MaxRetries        int
	ProcessingTimeout time.Duration
	RetryDelay        time.Duration
}

func NewIndexWorker(
	esClient *indexer.ESClient,
	fetcher *indexer.Fetcher,
	marker *indexer.Marker,
	config Config,
) *IndexWorker {
	return &IndexWorker{
		esClient: esClient,
		fetcher:  fetcher,
		marker:   marker,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

func (w *IndexWorker) Start() {
	for i := 0; i < w.config.WorkerCount; i++ {
		w.wg.Add(1)
		go w.workerLoop(i)
	}
}

func (w *IndexWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

func (w *IndexWorker) workerLoop(workerID int) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.FetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processBatch(workerID)
		case <-w.stopChan:
			return
		}
	}
}

func (w *IndexWorker) processBatch(workerID int) {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.ProcessingTimeout)
	defer cancel()

	// Check ES health before processing
	healthy, err := w.esClient.IsHealthyForIndexing(ctx)
	if err != nil {
		logger.Info("Worker-%d: Failed to check ES health: %v", workerID, err)
		return
	}

	if !healthy {
		logger.Info("Worker-%d: ES health check failed - skipping batch processing", workerID)
		return
	}

	w.fetchMutex.Lock()

	entries, err := w.fetcher.FetchUnindexedLogs(ctx, w.config.BatchSize)
	if err != nil {
		w.fetchMutex.Unlock()
		return
	}

	if len(entries) == 0 {
		w.fetchMutex.Unlock()
		return
	}

	ids := w.extractIDs(entries)

	if err := w.marker.MarkAsProcessing(ctx, ids); err != nil {
		w.fetchMutex.Unlock()
		return
	}

	w.fetchMutex.Unlock()

	logger.Info("Worker-%d processing batch - Size: %d", workerID, len(entries))

	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Worker-%d retry attempt %d/%d - Size: %d", workerID, attempt, w.config.MaxRetries, len(entries))
		}

		err := w.esClient.BulkIndex(ctx, entries)
		if err == nil {
			// Successfully indexed in ES, now mark as indexed in PostgreSQL
			if markErr := w.marker.MarkAsIndexed(ctx, ids); markErr != nil {
				logger.Error("Worker-%d CRITICAL: ES indexed but failed to mark as indexed in PostgreSQL: %v", workerID, markErr)
				// If we can't mark as indexed, we should mark as failed to prevent inconsistency
				if failErr := w.marker.MarkAsFailed(ctx, ids, "Failed to mark as indexed: "+markErr.Error()); failErr != nil {
					logger.Error("Worker-%d CRITICAL: Failed to mark entries as failed: %v", workerID, failErr)
				}
				return
			}
			logger.Info("Worker-%d batch indexed successfully - Size: %d", workerID, len(entries))
			return
		}

		// Log the specific error for debugging
		logger.Error("Worker-%d attempt %d failed: %v", workerID, attempt+1, err)

		if attempt == w.config.MaxRetries {
			// Final attempt failed, mark as failed in PostgreSQL
			if failErr := w.marker.MarkAsFailed(ctx, ids, err.Error()); failErr != nil {
				logger.Error("Worker-%d CRITICAL: Failed to mark entries as failed: %v", workerID, failErr)
			}
			logger.Error("Worker-%d batch failed after %d attempts - Size: %d, Final Error: %v", workerID, w.config.MaxRetries+1, len(entries), err)
			return
		}

		// Exponential backoff for retries
		retryDelay := w.config.RetryDelay * time.Duration(1<<attempt) // 2s, 4s, 8s
		logger.Info("Worker-%d waiting %v before retry", workerID, retryDelay)
		time.Sleep(retryDelay)
	}
}

func (w *IndexWorker) extractIDs(entries []*logentry.LogEntry) []int64 {
	ids := make([]int64, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}
	return ids
}

func (w *IndexWorker) HealthCheck(ctx context.Context) error {
	if err := w.esClient.HealthCheck(ctx); err != nil {
		return err
	}

	return nil
}

func (w *IndexWorker) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"workerStatus": "running",
		"lastSync":     time.Now(),
	}

	if err := w.esClient.HealthCheck(ctx); err != nil {
		stats["esStatus"] = "error"
	} else {
		stats["esStatus"] = "healthy"
	}

	fetcherStats, err := w.fetcher.GetStats(ctx)
	if err == nil {
		for k, v := range fetcherStats {
			stats[k] = v
		}
	}

	return stats, nil
}
