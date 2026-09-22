package mock

import "sync"

// Bucket supplies seed data to constructors; clients never retain its map.
type Bucket struct {
	Name    string
	Objects map[string]*MemoryFile
}

// SharedBucket owns objects and the mutex used by every attached client.
// Use a pointer; do not copy a SharedBucket after first use.
type SharedBucket struct {
	mu      sync.RWMutex
	objects map[string]*MemoryFile
}

// NewSharedBucket snapshots seed; subsequent changes to seed do not affect it.
// Nil seed entries are ignored. Do not mutate seed during this call.
func NewSharedBucket(seed Bucket) *SharedBucket {
	bucket := &SharedBucket{objects: make(map[string]*MemoryFile, len(seed.Objects))}
	for key, file := range seed.Objects {
		if file != nil {
			bucket.objects[key] = file.snapshot()
		}
	}
	return bucket
}

func (b *SharedBucket) initialize() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.objects == nil {
		b.objects = make(map[string]*MemoryFile)
	}
}
