package veild

import (
	"strings"
	"testing"
)

func TestCache_createCacheKey(t *testing.T) {
	got := uint64(createCacheKey([]byte("some key seed")))
	want := uint64(2892094225965879911)
	if got != want {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestCache_cacheKey_String(t *testing.T) {
	ck := createCacheKey([]byte("some other key seed"))
	got := ck.String()
	want := "0xa059c23b24ac935"
	if strings.Compare(got, want) != 0 {
		t.Errorf("wanted %v, got %v", want, got)
	}
}
