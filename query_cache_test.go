package veild

import (
	"crypto/sha1"
	"encoding/binary"
	"os"
	"testing"
	"time"
)

func TestQueryCache_NewQueryCache(t *testing.T) {
	queryCache := NewQueryCache()
	v := Query{}
	k := sha1.Sum([]byte("test"))
	queryCache.Put(k, v)
}

func TestQueryCache_Get(t *testing.T) {
	queryCache := NewQueryCache()
	v := Query{}
	k := sha1.Sum([]byte("test"))
	queryCache.Put(k, v)

	kv := sha1.Sum([]byte("test"))
	queryCache.Get(kv)

	de := sha1.Sum([]byte("doesn't exist"))
	if _, ok := queryCache.Get(de); ok {
		t.Fail()
	}
}

func TestQueryCache_Reaper(t *testing.T) {
	file, _ := os.ReadFile("fixtures/phishing-detection.api.cx.metamask.io_a.pkt")
	queryCache := NewQueryCache()
	n := len(file)
	nameType, _ := sliceNameType(file[12:n])
	s := createCacheKey(nameType)
	offsets, _ := ttlOffsets(file[:n])
	queryCache.Put(s, Query{file[:n], offsets, time.Now()})
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

func TestQueryCache_createCacheKey(t *testing.T) {
	k := []byte{0x00}
	if len(createCacheKey(k)) != sha1.Size {
		t.Error("key not the right length")
	}
}
