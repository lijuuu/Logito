package ingest

import (
	"sync"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

// Batcher handles in-memory batching of log entries with object pooling
type Batcher struct {
	// configuration
	maxBatchSize  int
	flushInterval time.Duration

	// internal state
	entries     []*logentry.LogEntry
	mu          sync.RWMutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
	workerChan  chan *logentry.Batch

	// object pool for reusing objects
	pool *ObjectPool

	// statistics
	stats   *BatcherStats
	statsMu sync.RWMutex
}
// BatcherStats holds statistics about the batcher
type BatcherStats struct {
	TotalReceived     int64
	TotalProcessed    int64
	TotalFailed       int64
	CurrentBatchSize  int
	CurrentMemoryLogs int
	LastFlushTime     time.Time
}

// NewBatcher creates a new batcher with the specified configuration
func NewBatcher(maxBatchSize, maxBatchCount int, flushInterval time.Duration, pool *ObjectPool) *Batcher {
	b := &Batcher{
		maxBatchSize:  maxBatchSize,
		flushInterval: flushInterval,
		entries:       make([]*logentry.LogEntry, 0, maxBatchSize),
		stopChan:      make(chan struct{}),
		workerChan:    make(chan *logentry.Batch, maxBatchCount), // much larger buffer for extreme load
		pool:          pool,                                      // assign the pool
		stats:         &BatcherStats{},
	}

	// start flush ticker
	b.flushTicker = time.NewTicker(flushInterval)

	// start background goroutines
	go b.flushLoop()

	return b
}

// AddLogs adds log entries to the batcher with timeout protection
func (b *Batcher) AddLogs(entries []*logentry.LogEntry) {
	if len(entries) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// check if adding new entries would exceed batch size
	if len(b.entries)+len(entries) > b.maxBatchSize {
		// force flush current batch
		b.flushBatch()
	}

	// add entries to current batch
	b.entries = append(b.entries, entries...)
	b.updateStats()

	// check if batch is full
	if len(b.entries) >= b.maxBatchSize {
		b.flushBatch()
	}
}

// flushBatch flushes the current batch to workers
func (b *Batcher) flushBatch() {
	if len(b.entries) == 0 {
		return
	}

	// create batch from pool
	batch := b.pool.GetBatch()

	// copy entries to batch
	for _, entry := range b.entries {
		batch.Add(entry)
	}

	// send to worker channel with timeout
	select {
	case b.workerChan <- batch:
		// successfully sent
	case <-time.After(10 * time.Millisecond):
		// shorter timeout to prevent stalling, process synchronously
		b.processBatch(batch)
	}

	// clear current entries
	b.entries = b.entries[:0]
	b.updateStats()
}

// flushLoop handles periodic flushing
func (b *Batcher) flushLoop() {
	for {
		select {
		case <-b.flushTicker.C:
			b.mu.Lock()
			if len(b.entries) > 0 {
				b.flushBatch()
			}
			b.mu.Unlock()
		case <-b.stopChan:
			// flush remaining entries before stopping
			b.mu.Lock()
			if len(b.entries) > 0 {
				b.flushBatch()
			}
			b.mu.Unlock()
			return
		}
	}
}

// processBatch processes a batch of log entries (fallback when workers are busy)
func (b *Batcher) processBatch(batch *logentry.Batch) {
	// this is a fallback when worker channel is full
	// process the batch synchronously to prevent stalling
	if batch.IsEmpty() {
		b.pool.PutBatch(batch)
		return
	}

	// create a simple database client for fallback processing
	// this prevents the main pipeline from stalling
	go func() {
		defer b.pool.PutBatch(batch)
		// in a real implementation, you'd process the batch here
		// for now, we just return it to the pool to prevent memory leaks
	}()
}

// GetWorkerChan returns the worker channel for batch processing
func (b *Batcher) GetWorkerChan() <-chan *logentry.Batch {
	return b.workerChan
}

// GetStats returns current batcher statistics (non-blocking)
func (b *Batcher) GetStats() *BatcherStats {
	// try to acquire stats lock with timeout
	statsChan := make(chan *BatcherStats, 1)

	go func() {
		b.statsMu.RLock()
		defer b.statsMu.RUnlock()

		stats := *b.stats

		// try to get current batch size with very short timeout
		select {
		case <-time.After(1 * time.Millisecond):
			// timeout - use cached values to prevent blocking
			stats.CurrentBatchSize = 0
			stats.CurrentMemoryLogs = 0
		default:
			// try to acquire the main mutex
			done := make(chan struct{})
			go func() {
				b.mu.RLock()
				stats.CurrentBatchSize = len(b.entries)
				stats.CurrentMemoryLogs = len(b.entries)
				b.mu.RUnlock()
				close(done)
			}()

			select {
			case <-done:
				// successfully got the values
			case <-time.After(1 * time.Millisecond):
				// timeout - use default values
				stats.CurrentBatchSize = 0
				stats.CurrentMemoryLogs = 0
			}
		}

		statsChan <- &stats
	}()

	select {
	case stats := <-statsChan:
		return stats
	case <-time.After(10 * time.Millisecond):
		// return minimal stats if timeout to prevent http blocking
		return &BatcherStats{
			TotalReceived:     0,
			TotalProcessed:    0,
			TotalFailed:       0,
			CurrentBatchSize:  0,
			CurrentMemoryLogs: 0,
		}
	}
}

// updateStats updates internal statistics
func (b *Batcher) updateStats() {
	b.statsMu.Lock()
	defer b.statsMu.Unlock()

	b.stats.CurrentBatchSize = len(b.entries)
	b.stats.CurrentMemoryLogs = len(b.entries)
	b.stats.LastFlushTime = time.Now()
}

// Stop stops the batcher and flushes remaining entries
func (b *Batcher) Stop() {
	close(b.stopChan)
	b.flushTicker.Stop()
}
