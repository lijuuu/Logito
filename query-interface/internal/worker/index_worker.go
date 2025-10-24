package worker

import (
	"context"
	"sync"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/indexer"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"
)

// IndexWorker handles syncing logs from postgres to elasticsearch
type IndexWorker struct {
	esClient   *indexer.ESClient
	fetcher    *indexer.Fetcher
	marker     *indexer.Marker
	config     Config
	stopChan   chan struct{}
	wg         sync.WaitGroup
	fetchMutex sync.Mutex // prevents concurrent fetching and marking of the same rows
}

// Config represents worker configuration
type Config struct {
	WorkerCount       int
	FetchInterval     time.Duration
	BatchSize         int
	MaxRetries        int
	ProcessingTimeout time.Duration
	RetryDelay        time.Duration
}

// NewIndexWorker creates a new index worker
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

// Start starts multiple index workers for concurrent processing
func (w *IndexWorker) Start() {
	// start multiple workers for concurrent es indexing
	for i := 0; i < w.config.WorkerCount; i++ {
		w.wg.Add(1)
		go w.workerLoop(i)
	}
}

// Stop stops the index worker gracefully
func (w *IndexWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// workerLoop is the main worker loop for concurrent processing
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

// processBatch processes a batch of unindexed logs with worker id
func (w *IndexWorker) processBatch(workerID int) {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.ProcessingTimeout)
	defer cancel()

	// use mutex to prevent concurrent fetching and marking of the same rows
	w.fetchMutex.Lock()

	// fetch unindexed logs
	entries, err := w.fetcher.FetchUnindexedLogs(ctx, w.config.BatchSize)
	if err != nil {
		w.fetchMutex.Unlock()
		return
	}

	if len(entries) == 0 {
		w.fetchMutex.Unlock()
		return // no logs to process
	}

	// extract ids for tracking
	ids := w.extractIDs(entries)

	// mark as processing to prevent duplicate processing
	if err := w.marker.MarkAsProcessing(ctx, ids); err != nil {
		w.fetchMutex.Unlock()
		return
	}

	// release the mutex after fetching and marking as processing
	w.fetchMutex.Unlock()

	// attempt to index with retries
	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		err := w.esClient.BulkIndex(ctx, entries)
		if err == nil {
			// success, mark as indexed
			if err := w.marker.MarkAsIndexed(ctx, ids); err != nil {
			}
			return
		}

		// if this is the last attempt, mark as failed
		if attempt == w.config.MaxRetries {
			if err := w.marker.MarkAsFailed(ctx, ids, err.Error()); err != nil {
			}
			return
		}

		// wait before retry with exponential backoff
		retryDelay := w.config.RetryDelay * time.Duration(attempt+1)
		time.Sleep(retryDelay)
	}
}

// extractIDs extracts database ids from log entries
func (w *IndexWorker) extractIDs(entries []*logentry.LogEntry) []int64 {
	// this is a simplified version
	// in a real implementation, you'd need to track the database ids
	ids := make([]int64, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}
	return ids
}

// HealthCheck checks if the worker is healthy
func (w *IndexWorker) HealthCheck(ctx context.Context) error {
	// check elasticsearch connection
	if err := w.esClient.HealthCheck(ctx); err != nil {
		return err
	}

	// check postgres connection
	// this would be implemented in the fetcher
	return nil
}

// GetStats returns worker statistics
func (w *IndexWorker) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"workerStatus": "running",
		"lastSync":     time.Now(),
	}

	// add es stats
	if err := w.esClient.HealthCheck(ctx); err != nil {
		stats["esStatus"] = "error"
	} else {
		stats["esStatus"] = "healthy"
	}

	// add fetcher stats
	fetcherStats, err := w.fetcher.GetStats(ctx)
	if err == nil {
		for k, v := range fetcherStats {
			stats[k] = v
		}
	}

	return stats, nil
}
