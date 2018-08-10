package veild

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"
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

// Put puts an entry into the response cache.
func (r *QueryCache) Put(key [sha1.Size]byte, value Query) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries[key] = value
}

// Get gets an entry from the response cache.
func (r *QueryCache) Get(key [sha1.Size]byte) (interface{}, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if val, ok := r.queries[key]; ok {
		log.Printf("[cache] \x1b[32;1mCache hit for 0x%x\x1b[0m\n", key)
		return val, true
	}
	return nil, false
}

// Reaper ticks over every second and runs through the TTL decrements.
func (r *QueryCache) Reaper() {
	for {
		mm := 0
		t := time.Now()

		r.mu.Lock() // Lock first
		for k, v := range r.queries {
			if newData, l := decTTL(v.data, v.offsets, 1); l {
				r.queries[k] = Query{newData, v.offsets}
				mm += len(v.offsets)
				continue
			}
			delete(r.queries, k)
		}
		r.mu.Unlock() // Remove lock

		elapsed := time.Since(t)
		numEntries := len(r.queries)

		log.Printf("[cache] Spent in loop: %v - entries: %d\n", elapsed, numEntries)

		time.Sleep(time.Second)
	}
}

// ttlRipper scans a DNS records and returns offsets of all the TTLs within it.
func ttlRipper(data []byte) []int {

	byteOffsets := []int{}

	// Skip first 12 bytes (always the header, no TTLs).
	startOffset := 12
	// Quickly run through the query (single one).
	i := bytes.IndexByte(data[startOffset:], 0x00)
	i += 5 // jump 1 + 4 more bytes (Type and Class).
	startOfAnswers := i + startOffset

	length := len(data) // size of payload

	for length > startOfAnswers {
		for {
			marker := data[startOfAnswers : startOfAnswers+1]
			if bytes.Equal(marker, []byte{0xc0}) {
				// pointer ref, only 2 bytes
				startOfAnswers += 2
				break
			} else if bytes.Equal(marker, []byte{0x00}) {
				// end of record
				startOfAnswers++
				break
			} else {
				startOfAnswers += int(marker[0]) + 1
			}
		}

		startOfAnswers += 4 // skip over type and class

		// TTL
		byteOffsets = append(byteOffsets, startOfAnswers)
		startOfAnswers += 4

		// data length
		le := binary.BigEndian.Uint16(data[startOfAnswers : startOfAnswers+2])
		startOfAnswers += 2
		startOfAnswers += int(le)
	}

	return byteOffsets
}

// createCacheKey generates a cache key from a given name and rtype (in bytes).
func createCacheKey(key []byte) [sha1.Size]byte {
	return sha1.Sum(key)
}

// decTTL decrements a responses TTL by n seconds.
func decTTL(data []byte, offsets []int, n int) ([]byte, bool) {
	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		if err := recover(); err != nil {
			fmt.Println(err)
			fmt.Printf("Data at point of problem: %x - offsets: %v\n", data, offsets)
		}
	}()
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
