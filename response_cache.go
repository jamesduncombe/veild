package veild

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// ResponseCache represents a response cache.
type ResponseCache struct {
	mu        sync.Mutex
	responses map[uint64]Packet
	log       *log.Logger
}

// NewResponseCache handles ResponseCache initialization.
func NewResponseCache() *ResponseCache {
	return &ResponseCache{
		responses: make(map[uint64]Packet),
		log:       log.New(os.Stdout, "[response_cache] ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (r *ResponseCache) Set(value Packet) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.responses[value.cacheKey()] = value
}

// Get gets an entry from the response cache.
func (r *ResponseCache) Get(key uint64) (Packet, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if packet, ok := r.responses[key]; ok {
		delete(r.responses, key)
		return packet, true
	}
	return Packet{}, false
}

// Entries outputs all the current entries in the cache along with their TTLs.
func (rc *ResponseCache) Entries(f io.Writer) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for _, response := range rc.responses {
		rr, _ := NewRR(response.data[HeaderLength:])

		fmt.Fprintf(f, "%s, %s\n", rr.hostname, rr.rType)
	}
}
