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
	responses map[cacheKey]*Request
	log       *log.Logger
}

// NewResponseCache handles ResponseCache initialization.
func NewResponseCache() *ResponseCache {
	return &ResponseCache{
		responses: make(map[cacheKey]*Request),
		log:       log.New(os.Stdout, "[response_cache] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Set adds a [Request] to the cache.
func (rc *ResponseCache) Set(value *Request) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.responses[value.cacheKey()] = value
}

// Get gets a [Request] from the cache.
func (rc *ResponseCache) Get(key cacheKey) (*Request, bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if request, ok := rc.responses[key]; ok {
		delete(rc.responses, key)
		return request, true
	}
	return &Request{}, false
}

// Check if an entry exists in the cache.
func (rc *ResponseCache) Exists(key cacheKey) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	_, ok := rc.responses[key]
	return ok
}

// Entries outputs all the current entries in the cache along with their TTLs.
func (rc *ResponseCache) Entries(f io.Writer) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for _, response := range rc.responses {
		rr, _ := NewRR(response.data[DnsHeaderLength:])

		fmt.Fprintf(f, "%s, %s\n", rr.hostname, rr.rType)
	}
}
