package veild

import (
	"encoding/binary"
	"os"
	"testing"
	"time"
)

func TestQueryCache_NewQueryCache(t *testing.T) {
	queryCache := NewQueryCache()
	v := Query{}
	queryCache.Set(v)
}

func TestQueryCache_Get(t *testing.T) {
	queryCache := NewQueryCache()
	v := Query{}
	queryCache.Set(v)

	queryCache.Get(v.cacheKey())

	if _, ok := queryCache.Get(v.cacheKey()); ok {
		t.Fail()
	}
}

func TestQueryCache_Reaper(t *testing.T) {
	file, _ := os.ReadFile("fixtures/phishing-detection.api.cx.metamask.io_a.pkt")
	queryCache := NewQueryCache()
	n := len(file)

	offsets, _ := ttlOffsets(file[:n])
	queryCache.Set(Query{file[:n], offsets, time.Now()})
	queryCache.reaper()
}
