package veild

import (
	"encoding/binary"
	"os"
	"reflect"
	"testing"
)

func TestQuery_cacheKey(t *testing.T) {
	t.Skip()
}

func TestQuery_decTTL(t *testing.T) {
	file, _ := os.ReadFile("fixtures/client.dropbox.com_aaaa.pkt")
	offsets, _ := ttlOffsets(file)

	originalTtls := []uint32{}

	for _, offset := range offsets {
		originalTtls = append(originalTtls, binary.BigEndian.Uint32(file[offset:offset+4]))
	}

	query := &Query{offsets: offsets, data: file}

	query.decTTL(1)

	for i, offset := range offsets {
		got := binary.BigEndian.Uint32(query.data[offset : offset+4])
		want := originalTtls[i] - 1

		// Check TTL is decremented by 1.
		if want != got {
			t.Errorf("wanted %v got %v", want, got)
		}
	}
}

func TestQuery_getTTLs(t *testing.T) {
	file, _ := os.ReadFile("fixtures/client.dropbox.com_aaaa.pkt")
	offsets, _ := ttlOffsets(file)

	query := Query{data: file, offsets: offsets}
	got := query.getTTLs()
	want := []uint32{31, 60}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v got %v", want, got)
	}
}
