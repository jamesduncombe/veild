package veild

import (
	"encoding/binary"
	"time"
)

// Query holds the structure for the raw response data and offsets of TTLs.
type Query struct {
	data     []byte
	offsets  []int
	creation time.Time
}

func (q Query) cacheKey() cacheKey {
	nameType, _ := sliceNameType(q.data[DnsHeaderLength:])
	return createCacheKey(nameType)
}

// decTTL decrements a responses TTL by n seconds.
func (q Query) decTTL(decrementBy uint32) (Query, bool) {
	for _, offset := range q.offsets {
		currentTTL := binary.BigEndian.Uint32(q.data[offset : offset+4])

		// If we're decrementing to 0 or past 0 then the record should expire.
		if decrementBy >= currentTTL {
			return q, false
		}

		// Update TTL.
		binary.BigEndian.PutUint32(q.data[offset:offset+4], currentTTL-decrementBy)
	}
	return q, true
}

// getTTLs gets the TTLs from the offsets using the data.
func (q Query) getTTLs() []uint32 {
	ttls := []uint32{}
	for _, offset := range q.offsets {
		currentTTL := binary.BigEndian.Uint32(q.data[offset : offset+4])
		ttls = append(ttls, currentTTL)
	}
	return ttls
}
