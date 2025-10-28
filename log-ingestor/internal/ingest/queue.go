package ingest

import (
	"sync"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/config"
	"github.com/lijuuu/Logito/log-ingestor/internal/dlq"
	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
	"github.com/lijuuu/Logito/log-ingestor/internal/streaming"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

type Batcher struct {
	maxBatchSize  int
	flushInterval time.Duration
	entries       []*logentry.LogEntry
	mu            sync.RWMutex
	flushTicker   *time.Ticker
	stopChan      chan struct{}
	workerChan    chan *logentry.Batch
	pool          *ObjectPool
	dlq           dlq.DLQ
	config        *config.Config
	stats         *BatcherStats
	streamServer  *streaming.StreamServer
}

type BatcherStats struct {
	TotalReceived    int64
	TotalProcessed   int64
	TotalFailed      int64
	CurrentBatchSize int
	LastFlushTime    time.Time
}

func NewBatcher(maxBatchSize, maxBatchCount int, flushInterval time.Duration, pool *ObjectPool, dlq dlq.DLQ, cfg *config.Config, streamServer *streaming.StreamServer) *Batcher {
	b := &Batcher{
		maxBatchSize:  maxBatchSize,
		flushInterval: flushInterval,
		entries:       make([]*logentry.LogEntry, 0, maxBatchSize),
		stopChan:      make(chan struct{}),
		workerChan:    make(chan *logentry.Batch, maxBatchCount),
		pool:          pool,
		dlq:           dlq,
		config:        cfg,
		stats:         &BatcherStats{},
		streamServer:  streamServer,
	}

	b.flushTicker = time.NewTicker(flushInterval)
	go b.flushLoop()

	return b
}

func (b *Batcher) AddLogs(entries []*logentry.LogEntry) {
	if len(entries) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries)+len(entries) > b.maxBatchSize {
		logger.Info("Batch size limit reached, flushing current batch (size: %d)", len(b.entries))
		b.flushBatch()
	}

	b.entries = append(b.entries, entries...)

	if len(b.entries) >= b.maxBatchSize {
		logger.Info("Batch full, flushing (size: %d)", len(b.entries))
		b.flushBatch()
	}
}

func (b *Batcher) flushBatch() {
	if len(b.entries) == 0 {
		return
	}

	entries := b.pool.GetLogEntrySlice()
	entries = append(entries, b.entries...)

	batch := &logentry.Batch{
		Entries: entries,
		Size:    len(entries),
	}

	if b.streamServer != nil {
		b.streamServer.BroadcastBatch(batch)
	}

	select {
	case b.workerChan <- batch:
		logger.Info("Batch sent to worker channel, size: %d", len(batch.Entries))
	case <-time.After(500 * time.Millisecond):
		logger.Error("Worker channel overloaded, sending batch to DLQ, size: %d", len(batch.Entries))
		b.sendToDLQ(batch)
	}

	b.entries = b.entries[:0]
}

// Background goroutine that periodically flushes batches based on time interval
func (b *Batcher) flushLoop() {
	for {
		select {
		case <-b.flushTicker.C:
			b.mu.Lock()
			if len(b.entries) > 0 {
				logger.Info("Periodic flush triggered, batch size: %d", len(b.entries))
				b.flushBatch()
			}
			b.mu.Unlock()
		case <-b.stopChan:
			b.mu.Lock()
			if len(b.entries) > 0 {
				b.flushBatch()
			}
			b.mu.Unlock()
			return
		}
	}
}

func (b *Batcher) sendToDLQ(batch *logentry.Batch) {
	if batch.IsEmpty() {
		// Return slice to pool before discarding batch
		b.pool.PutLogEntrySlice(batch.Entries)
		return
	}

	if b.dlq == nil {
		logger.Error("DLQ not available, batch dropped - size: %d", len(batch.Entries))
		// Return slice to pool before discarding batch
		b.pool.PutLogEntrySlice(batch.Entries)
		return
	}

	dlqCount := 0
	for _, entry := range batch.Entries {
		err := dlq.SendToDLQIfEnabled(b.dlq, b.config, entry, "Worker channel overloaded", dlq.FailureTypeTimeoutError)
		if err != nil {
			logger.Error("Failed to send entry to DLQ: %v", err)
		} else {
			dlqCount++
		}
	}

	logger.Info("Batch sent to DLQ - Total: %d, Success: %d", len(batch.Entries), dlqCount)
	// Return slice to pool after processing
	b.pool.PutLogEntrySlice(batch.Entries)
}

// GetWorkerChan returns the worker channel for batch processing
func (b *Batcher) GetWorkerChan() <-chan *logentry.Batch {
	return b.workerChan
}

func (b *Batcher) Stop() {
	close(b.stopChan)
	b.flushTicker.Stop()
}
