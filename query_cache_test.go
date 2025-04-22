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

func TestQueryCache_ttlOffsets(t *testing.T) {

	tests := []struct {
		filename string
		shouldBe []int
	}{
		{
			filename: "fixtures/phishing-detection.api.cx.metamask.io_a.pkt",
			shouldBe: []int{61, 128, 186, 213, 251, 307, 321},
		},
	}

	for i, test := range tests {
		data, _ := os.ReadFile(test.filename)
		offsets, _ := ttlOffsets(data)
		if len(offsets) != len(test.shouldBe) {
			t.Errorf("wanted %d length got %d", len(offsets), len(test.shouldBe))
			break
		}
		if offsets[i] != test.shouldBe[i] {
			t.Errorf("wanted %d offset got %d", offsets[i], test.shouldBe[i])
		}
	}
}

func TestQueryCache_decTTL(t *testing.T) {
	file, _ := os.ReadFile("fixtures/client.dropbox.com_aaaa.pkt")

	offsets, _ := ttlOffsets(file)

	originalTtls := []uint32{}

	for _, offset := range offsets {
		originalTtls = append(originalTtls, binary.BigEndian.Uint32(file[offset:offset+4]))
	}

	b, _ := decTTL(file, offsets, 1)

	for i, offset := range offsets {
		newTtl := binary.BigEndian.Uint32(b[offset : offset+4])

		// Check TTL is decremented by 1.
		if newTtl != originalTtls[i]-1 {
			t.Errorf("wanted %d ttl got %d", originalTtls[i]-1, newTtl)
		}
	}
}
