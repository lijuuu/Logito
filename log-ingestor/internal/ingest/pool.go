package ingest

import (
	"sync"

	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

type ObjectPool struct {
	logEntryPool      sync.Pool
	logEntrySlicePool sync.Pool
}

func NewObjectPool() *ObjectPool {
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
