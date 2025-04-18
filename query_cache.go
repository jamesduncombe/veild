package veild

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Errors in the query cache.
var (
	// ErrProblemParsingOffsets is returned when a TTL offset cannot be parsed.
	ErrProblemParsingOffsets = errors.New("problem parsing TTL offsets")
)

// Query holds the structure for the raw response data and offsets of TTLs.
type Query struct {
	data     []byte
	offsets  []int
	creation time.Time
}

// QueryCache holds the main structure of the query cache.
type QueryCache struct {
	mu      sync.RWMutex
	queries map[[sha1.Size]byte]Query
	log     *log.Logger
}

// NewQueryCache handles QueryCache initialization.
func NewQueryCache() *QueryCache {
	return &QueryCache{
		queries: make(map[[sha1.Size]byte]Query),
		log:     log.New(os.Stdout, "[query_cache] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Put puts an entry into the query cache.
func (qc *QueryCache) Put(key [sha1.Size]byte, value Query) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.queries[key] = value
}

// Get gets an entry from the query cache.
func (qc *QueryCache) Get(key [sha1.Size]byte) ([]byte, bool) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	if query, ok := qc.queries[key]; ok {
		// Try decrementing the TTL by n seconds.
		decBy := uint32(time.Since(query.creation).Seconds())

		if newRecord, ok := decTTL(query.data, query.offsets, decBy); ok {
			return newRecord, true
		}

		// Remove it, must be too old.
		qc.log.Printf("\x1b[31;1m[get] Removing: 0x%x\x1b[0m\n", key)
		delete(qc.queries, key)
	}

	return []byte{}, false
}

// Entries outputs all the current entries in the cache along with their TTLs.
func (qc *QueryCache) Entries(f io.Writer) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	for _, query := range qc.queries {
		rr, _ := NewRR(query.data[12:])
		ttls := getTTLs(query.data, query.offsets)
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

		if newRecord, ok := decTTL(query.data, query.offsets, decBy); ok {
			qc.queries[k] = Query{newRecord, query.offsets, now}
			continue
		}
		qc.log.Printf("\x1b[31;1mRemoving: 0x%x\x1b[0m\n", k)
		delete(qc.queries, k)
	}

	elapsed := time.Since(t)
	numEntries := len(qc.queries)

	qc.log.Printf("Spent in loop: %v - entries: %d\n", elapsed, numEntries)
}

// ttlOffsets scans a DNS record and returns offsets of all the TTLs within it.
// SEE: https://www.rfc-editor.org/rfc/rfc1035#section-3.2
// SEE: https://cs.opensource.google/go/x/net/+/master:dns/dnsmessage/message.go;l=2105;drc=ea0c1d94f5e0c4b4c18b927e26e188ad8fadb38e
func ttlOffsets(data []byte) ([]int, error) {

	ttlOffsets := []int{}

	// Get total answers etc.
	answers := binary.BigEndian.Uint16(data[6:8])
	authority := binary.BigEndian.Uint16(data[8:10])

	total := int(answers + authority)

	// Skip first 12 bytes (always the header, no TTLs).
	offset := HeaderLength

	// Attempting to jump over Questions section.

	// Quickly run through the query (single one).
	i := bytes.IndexByte(data[offset:], 0x00)
	i += 5 // jump 1 + 4 more bytes (End of Name, Type and Class).
	offset += i

	// Parsing Answers and Authority RRs.

	for n := 0; n < total; n++ {

		// Handle NAME field.
		// This could be a pointer or a label.

		// Check we're not overrunning the length of the message.
		if len(data) < offset+1 {
			return nil, ErrProblemParsingOffsets
		}

		marker := data[offset : offset+1]
		c := marker[0]

		switch c & 0xc0 {

		case 0xc0: // Pointer ref, only 2 bytes.
			offset += 2

		case 0x00: // End of record.
			offset++

		default:
			return nil, fmt.Errorf("error on marker: 0x%x", marker)

		}

		// Advance past the TYPE and CLASS fields.
		offset += 4

		// TTL field.
		ttlOffsets = append(ttlOffsets, offset)
		offset += 4

		// RDLENGTH field.
		rdLength := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2

		// Advance past the RDATA field using RDLENGTH.
		offset += int(rdLength)
	}

	return ttlOffsets, nil
}

// createCacheKey generates a cache key from a given name and rtype (in bytes).
func createCacheKey(key []byte) [sha1.Size]byte {
	return sha1.Sum(key)
}

// decTTL decrements a responses TTL by n seconds.
func decTTL(data []byte, offsets []int, decrementBy uint32) ([]byte, bool) {
	for _, offset := range offsets {
		currentTTL := binary.BigEndian.Uint32(data[offset : offset+4])

		// If we're decrementing to 0 or past 0 then the record should expire.
		if decrementBy >= currentTTL {
			return nil, false
		}

		// Update TTL.
		binary.BigEndian.PutUint32(data[offset:offset+4], currentTTL-decrementBy)
	}
	return data, true
}

// getTTLs gets the TTLs from the offsets using the data.
func getTTLs(data []byte, offsets []int) []uint32 {
	ttls := []uint32{}
	for _, offset := range offsets {
		currentTTL := binary.BigEndian.Uint32(data[offset : offset+4])
		ttls = append(ttls, currentTTL)
	}
	return ttls
}
