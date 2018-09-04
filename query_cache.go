package veild

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"log"
	"sync"
	"time"
)

var (
	errProblemParsingOffsets = errors.New("problem parsing TTL offsets")
)

// Query holds the structure for the raw response data and offsets of TTLs.
type Query struct {
	data    []byte
	offsets []int
}

// QueryCache holds the main structure of the query cache.
type QueryCache struct {
	mu      sync.Mutex
	queries map[[sha1.Size]byte]Query
}

// NewQueryCache handles QueryCache initialization.
func NewQueryCache() *QueryCache {
	return &QueryCache{
		queries: make(map[[sha1.Size]byte]Query),
	}
}

// Put puts an entry into the query cache.
func (r *QueryCache) Put(key [sha1.Size]byte, value Query) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries[key] = value
}

// Get gets an entry from the query cache.
func (r *QueryCache) Get(key [sha1.Size]byte) (interface{}, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if val, ok := r.queries[key]; ok {
		return val, true
	}
	return nil, false
}

// Reaper ticks over every second and runs through the TTL decrements.
func (r *QueryCache) Reaper() {
	for {
		r.reaper()
	}
}

func (r *QueryCache) reaper() {
	mm := 0
	t := time.Now()

	r.mu.Lock()
	for k, v := range r.queries {
		if newRecord, ok := decTTL(v.data, v.offsets, 1); ok {
			r.queries[k] = Query{newRecord, v.offsets}
			mm += len(v.offsets)
			continue
		}
		delete(r.queries, k)
	}
	r.mu.Unlock()

	elapsed := time.Since(t)
	numEntries := len(r.queries)

	log.Printf("[cache] Spent in loop: %v - entries: %d\n", elapsed, numEntries)

	time.Sleep(time.Second)
}

// ttlOffsets scans a DNS records and returns offsets of all the TTLs within it.
func ttlOffsets(data []byte) ([]int, error) {

	byteOffsets := []int{}

	// Get total answers etc.
	answers := binary.BigEndian.Uint16(data[6:8])
	authority := binary.BigEndian.Uint16(data[8:10])
	additional := binary.BigEndian.Uint16(data[10:12])
	total := int(answers + authority + additional)

	// Skip first 12 bytes (always the header, no TTLs).
	startOffset := 12
	// Quickly run through the query (single one).
	i := bytes.IndexByte(data[startOffset:], 0x00)
	i += 5 // jump 1 + 4 more bytes (Type and Class).
	startOfAnswers := i + startOffset

	for n := 0; n < total; n++ {
		for {
			if len(data) < startOfAnswers+1 {
				return nil, errProblemParsingOffsets
			}
			marker := data[startOfAnswers : startOfAnswers+1]
			if bytes.Equal(marker, []byte{0xc0}) {
				// Pointer ref, only 2 bytes.
				startOfAnswers += 2
				break
			} else if bytes.Equal(marker, []byte{0x00}) {
				// End of record.
				startOfAnswers++
				break
			} else {
				startOfAnswers += int(marker[0]) + 1
			}
		}

		// Skip over type and class.
		startOfAnswers += 4

		// Before appending make sure this is a sane offset.
		if startOfAnswers > len(data) {
			return nil, errProblemParsingOffsets
		}

		// TTL.
		byteOffsets = append(byteOffsets, startOfAnswers)
		startOfAnswers += 4

		// Data length.
		le := binary.BigEndian.Uint16(data[startOfAnswers : startOfAnswers+2])
		startOfAnswers += 2
		startOfAnswers += int(le)
	}

	return byteOffsets, nil
}

// createCacheKey generates a cache key from a given name and rtype (in bytes).
func createCacheKey(key []byte) [sha1.Size]byte {
	return sha1.Sum(key)
}

// decTTL decrements a responses TTL by n seconds.
func decTTL(data []byte, offsets []int, n int) ([]byte, bool) {
	for _, offset := range offsets {
		m := binary.BigEndian.Uint32(data[offset : offset+4])
		if m-uint32(n) == uint32(0) {
			return nil, false
		}
		k := make([]byte, 4)
		binary.BigEndian.PutUint32(k, m-uint32(n))
		data[offset] = k[0]
		data[offset+1] = k[1]
		data[offset+2] = k[2]
		data[offset+3] = k[3]
	}
	return data, true
}
