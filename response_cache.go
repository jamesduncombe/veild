package veild

import (
	"sync"
)

// ResponseCache represents a response cache.
type ResponseCache struct {
	mu        sync.Mutex
	responses map[uint16]Packet
}

// NewResponseCache handles ResponseCache initialization.
func NewResponseCache() *ResponseCache {
	return &ResponseCache{
		responses: make(map[uint16]Packet),
	}
}

// Put puts an entry into the response cache.
func (r *ResponseCache) Put(key uint16, value Packet) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses[key] = value
}

// Get gets an entry from the response cache.
func (r *ResponseCache) Get(key uint16) (interface{}, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if val, ok := r.responses[key]; ok {
		delete(r.responses, key)
		return val, true
	}
	return nil, false
}
