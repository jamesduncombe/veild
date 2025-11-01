package veild

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func newQuery() *Query {
	return &Query{
		data: []byte{
			0x53, 0x1, 0x1, 0x20, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x1, 0xa, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x6e,
			0x6d, 0x61, 0x69, 0x6c, 0x3, 0x63, 0x6f, 0x6d, 0x0,
			0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x29, 0x0, 0x32, 0x0,
			0x0, 0x80, 0x0, 0x0, 0x0},
	}
}

func TestQueryCache_NewQueryCache(t *testing.T) {
	t.Skip()
}

func TestQueryCache_Set(t *testing.T) {
	logger := newLogger()

	queryCache := NewQueryCache(logger)
	v := newQuery()
	queryCache.Set(v)

	queryCache.Get(v.cacheKey())

	if _, ok := queryCache.Get(v.cacheKey()); !ok {
		t.Errorf("expected set and fetch to return query")
	}
}

func TestQueryCache_Get(t *testing.T) {
	logger := newLogger()

	queryCache := NewQueryCache(logger)
	v := newQuery()
	queryCache.Set(v)

	queryCache.Get(v.cacheKey())

	if _, ok := queryCache.Get(v.cacheKey()); !ok {
		t.Errorf("expected set and fetch to return query")
	}
}

func TestQueryCache_Entries(t *testing.T) {
	file, _ := os.ReadFile("fixtures/phishing-detection.api.cx.metamask.io_a.pkt")
	logger := newLogger()

	queryCache := NewQueryCache(logger)
	n := len(file)

	offsets, _ := ttlOffsets(file[:n])
	queryCache.Set(&Query{file[:n], offsets, time.Now()})

	var b bytes.Buffer
	queryCache.Entries(&b)

	want := "phishing-detection.api.cx.metamask.io, A, [56 582 15 14 14 40 32]\n"
	got := b.String()
	if got != want {
		t.Errorf("expected entries to output cache contents, got %q, want %q", got, want)
	}
}

func TestQueryCache_Reaper(t *testing.T) {
	file, _ := os.ReadFile("fixtures/phishing-detection.api.cx.metamask.io_a.pkt")
	logger := newLogger()

	queryCache := NewQueryCache(logger)
	n := len(file)

	offsets, _ := ttlOffsets(file[:n])
	queryCache.Set(&Query{file[:n], offsets, time.Now()})
	queryCache.reaper()
}
