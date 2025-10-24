package ingest

import (
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

// ObjectPool manages object pools for efficient memory reuse
type ObjectPool struct {
	// pool for log entries
	logEntryPool sync.Pool
	// pool for log entry slices
	logEntrySlicePool sync.Pool
	// pool for batch objects
	batchPool sync.Pool
	// pool for json decoders
	jsonDecoderPool sync.Pool
	// pool for byte slices
	byteSlicePool sync.Pool
}

// NewObjectPool creates a new object pool with proper initialization
func NewObjectPool(maxBatchCount int) *ObjectPool {
	return &ObjectPool{
		logEntryPool: sync.Pool{
			New: func() interface{} {
				return &logentry.LogEntry{}
			},
		},
		logEntrySlicePool: sync.Pool{
			New: func() interface{} {
				// use pointer to slice to avoid allocations
				slice := make([]*logentry.LogEntry, 0, 1000) // pre-allocate with capacity
				return &slice
			},
		},
		batchPool: sync.Pool{
			New: func() interface{} {
				return logentry.NewBatch(maxBatchCount)
			},
		},
		jsonDecoderPool: sync.Pool{
			New: func() interface{} {
				// create new json-iterator decoder for reuse
				return jsoniter.NewDecoder(nil)
			},
		},
		byteSlicePool: sync.Pool{
			New: func() interface{} {
				// use pointer to slice to avoid allocations
				slice := make([]byte, 0, 4096) // pre-allocate 4kb buffer
				return &slice
			},
		},
	}
}

// GetLogEntry gets a log entry from the pool
func (p *ObjectPool) GetLogEntry() *logentry.LogEntry {
	entry := p.logEntryPool.Get().(*logentry.LogEntry)
	// reset the entry
	entry.Level = ""
	entry.Message = ""
	entry.ResourceID = ""
	entry.Timestamp = logentry.LogEntry{}.Timestamp
	entry.TraceID = nil
	entry.SpanID = nil
	entry.Commit = nil
	entry.Metadata = nil
	entry.Indexed = false
	entry.ProcessingAt = nil
	return entry
}

// PutLogEntry returns a log entry to the pool
func (p *ObjectPool) PutLogEntry(entry *logentry.LogEntry) {
	p.logEntryPool.Put(entry)
}

// GetLogEntrySlice gets a log entry slice from the pool
func (p *ObjectPool) GetLogEntrySlice() []*logentry.LogEntry {
	slicePtr := p.logEntrySlicePool.Get().(*[]*logentry.LogEntry)
	// reset the slice but keep capacity
	*slicePtr = (*slicePtr)[:0]
	return *slicePtr
}

// PutLogEntrySlice returns a log entry slice to the pool
func (p *ObjectPool) PutLogEntrySlice(slice []*logentry.LogEntry) {
	// only return to pool if capacity is reasonable
	if cap(slice) <= 10000 {
		// create pointer to slice for pool
		slicePtr := &slice
		p.logEntrySlicePool.Put(slicePtr)
	}
}

// GetBatch gets a batch from the pool
func (p *ObjectPool) GetBatch() *logentry.Batch {
	batch := p.batchPool.Get().(*logentry.Batch)
	batch.Reset()
	return batch
}

// PutBatch returns a batch to the pool
func (p *ObjectPool) PutBatch(batch *logentry.Batch) {
	p.batchPool.Put(batch)
}

// GetByteSlice gets a byte slice from the pool
func (p *ObjectPool) GetByteSlice() []byte {
	slicePtr := p.byteSlicePool.Get().(*[]byte)
	// reset length but keep capacity
	*slicePtr = (*slicePtr)[:0]
	return *slicePtr
}

// PutByteSlice returns a byte slice to the pool
func (p *ObjectPool) PutByteSlice(slice []byte) {
	// only return to pool if capacity is reasonable
	if cap(slice) <= 65536 { //64KB max
		// create pointer to slice for pool
		slicePtr := &slice
		p.byteSlicePool.Put(slicePtr)
	}
}

// GetJSONDecoder gets a json-iterator decoder from the pool
func (p *ObjectPool) GetJSONDecoder() *jsoniter.Decoder {
	return p.jsonDecoderPool.Get().(*jsoniter.Decoder)
}

// PutJSONDecoder returns a json-iterator decoder to the pool
func (p *ObjectPool) PutJSONDecoder(decoder *jsoniter.Decoder) {
	p.jsonDecoderPool.Put(decoder)
}
