package ingest

import (
	"sync"

	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

type ObjectPool struct {
	logEntryPool      sync.Pool
	logEntrySlicePool sync.Pool
	batchPool         sync.Pool
	byteSlicePool     sync.Pool
}

func NewObjectPool(maxBatchCount int) *ObjectPool {
	return &ObjectPool{
		logEntryPool: sync.Pool{
			New: func() interface{} {
				return &logentry.LogEntry{}
			},
		},
		logEntrySlicePool: sync.Pool{
			New: func() interface{} {
				// Return pointer to slice to avoid allocations when reusing
				slice := make([]*logentry.LogEntry, 0, 1000)
				return &slice
			},
		},
		batchPool: sync.Pool{
			New: func() interface{} {
				return logentry.NewBatch(maxBatchCount)
			},
		},
		byteSlicePool: sync.Pool{
			New: func() interface{} {
				// Return pointer to slice to avoid allocations when reusing
				slice := make([]byte, 0, 4096)
				return &slice
			},
		},
	}
}

func (p *ObjectPool) GetLogEntry() *logentry.LogEntry {
	entry := p.logEntryPool.Get().(*logentry.LogEntry)
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

func (p *ObjectPool) PutLogEntry(entry *logentry.LogEntry) {
	p.logEntryPool.Put(entry)
}

func (p *ObjectPool) GetLogEntrySlice() []*logentry.LogEntry {
	slicePtr := p.logEntrySlicePool.Get().(*[]*logentry.LogEntry)
	*slicePtr = (*slicePtr)[:0]
	return *slicePtr
}

func (p *ObjectPool) PutLogEntrySlice(slice []*logentry.LogEntry) {
	if cap(slice) <= 10000 {
		slicePtr := &slice
		p.logEntrySlicePool.Put(slicePtr)
	}
}

func (p *ObjectPool) GetBatch() *logentry.Batch {
	batch := p.batchPool.Get().(*logentry.Batch)
	batch.Reset()
	return batch
}

func (p *ObjectPool) PutBatch(batch *logentry.Batch) {
	p.batchPool.Put(batch)
}

func (p *ObjectPool) GetByteSlice() []byte {
	slicePtr := p.byteSlicePool.Get().(*[]byte)
	// Reset length but keep capacity to reuse allocated memory
	*slicePtr = (*slicePtr)[:0]
	return *slicePtr
}

func (p *ObjectPool) PutByteSlice(slice []byte) {
	// Only return reasonable-sized slices to prevent memory bloat
	if cap(slice) <= 65536 {
		slicePtr := &slice
		p.byteSlicePool.Put(slicePtr)
	}
}
