package veild

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// QueryCache holds the main structure of the query cache.
type QueryCache struct {
	mu      sync.RWMutex
	queries map[cacheKey]Query
	log     *log.Logger
}

// NewQueryCache handles QueryCache initialization.
func NewQueryCache() *QueryCache {
	return &QueryCache{
		queries: make(map[cacheKey]Query),
		log:     log.New(os.Stdout, "[query_cache] ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (qc *QueryCache) Set(value Query) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.queries[value.cacheKey()] = value
}

// Get gets an entry from the query cache.
func (qc *QueryCache) Get(key cacheKey) (Query, bool) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if query, ok := qc.queries[key]; ok {
		// Try decrementing the TTL by n seconds.
		decBy := uint32(time.Since(query.creation).Seconds())

		if newQuery, ok := query.decTTL(decBy); ok {
			return newQuery, true
		}

		// Remove it, must be too old.
		qc.log.Printf("\x1b[31;1m[get] Removing: 0x%x\x1b[0m\n", key)
		delete(qc.queries, key)
	}

	return Query{}, false
}

// Entries outputs all the current entries in the cache along with their TTLs.
func (qc *QueryCache) Entries(f io.Writer) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	for _, query := range qc.queries {
		rr, _ := NewRR(query.data[DnsHeaderLength:])
		ttls := query.getTTLs()
		fmt.Fprintf(f, "%s, %s, %+v\n", rr.hostname, rr.rType, ttls)
	}
}

// Reaper ticks over and runs through the TTL decrements.
func (qc *QueryCache) Reaper() {
	for {
		qc.reaper()

		// Re-run after...
		time.Sleep(time.Minute)
	}
}

func (qc *QueryCache) reaper() {
	t := time.Now()

	qc.mu.Lock()
	defer qc.mu.Unlock()

	for k, query := range qc.queries {
		now := time.Now()

		decBy := uint32(now.Sub(query.creation).Seconds())

		if query, ok := query.decTTL(decBy); ok {
			qc.queries[k] = Query{query.data, query.offsets, now}
			continue
		}
		qc.log.Printf("\x1b[31;1mRemoving: 0x%x\x1b[0m\n", k)
		delete(qc.queries, k)
	}

	elapsed := time.Since(t)
	numEntries := len(qc.queries)

	qc.log.Printf("Spent in loop: %v - entries: %d\n", elapsed, numEntries)
}
