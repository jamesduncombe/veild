package veild

import "testing"

func TestResponseCache_Put(t *testing.T) {
	responseCache := NewResponseCache()

	v := Packet{}
	responseCache.Set(v)
}

func TestResponseCache_Get(t *testing.T) {
	responseCache := NewResponseCache()

	v := Packet{}
	responseCache.Set(v)
	_, ok := responseCache.Get(v.cacheKey())
	if !ok {
		t.Error("should exist")
	}

	_, ok = responseCache.Get(v.cacheKey())
	if ok {
		t.Error("shouldn't exist")
	}
}
