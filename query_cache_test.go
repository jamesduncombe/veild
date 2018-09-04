package veild

import (
	"crypto/sha1"
	"encoding/binary"
	"io/ioutil"
	"testing"
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
	file, _ := ioutil.ReadFile("fixtures/long_response.bin")
	queryCache := NewQueryCache()
	n := len(file)
	nameType := sliceNameType(file[12:n])
	s := createCacheKey(nameType)
	offsets, _ := ttlOffsets(file[:n])
	queryCache.Put(s, Query{file[:n], offsets})
	queryCache.reaper()
}

func TestQueryCache_ttlOffsets(t *testing.T) {
	file, _ := ioutil.ReadFile("fixtures/long_response.bin")
	offsets, _ := ttlOffsets(file)
	shouldBe := []int{45, 67, 91, 147, 221, 237, 253, 269, 285, 301, 317, 333}
	for i := range offsets {
		if offsets[i] != shouldBe[i] {
			t.Fail()
		}
	}
}

func TestQueryCache_ttlOffsets_ExtraData(t *testing.T) {
	file, _ := ioutil.ReadFile("fixtures/packet_with_extra_data.bin")
	offsets, _ := ttlOffsets(file)
	shouldBe := []int{42, 75}
	for i := range offsets {
		if offsets[i] != shouldBe[i] {
			t.Fail()
		}
	}
}

func TestQueryCache_decTTL(t *testing.T) {
	file, _ := ioutil.ReadFile("fixtures/response.bin")
	// TTL at 60 seconds
	offsets, _ := ttlOffsets(file)
	b, _ := decTTL(file, offsets, 1)
	for _, offset := range offsets {
		// Check TTL goes to 59
		if binary.BigEndian.Uint32(b[offset:offset+4]) != 59 {
			t.Fail()
		}
	}
}

func TestQueryCache_createCacheKey(t *testing.T) {
	k := []byte{0x00}
	key := createCacheKey(k)
	if len(key) != sha1.Size {
		t.Error("key not the right length")
	}
}
