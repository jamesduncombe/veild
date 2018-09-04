package veild

import (
	"testing"
)

func TestResponseCache_Put(t *testing.T) {
	responseCache := NewResponseCache()
	k := uint16(10)
	v := Packet{}
	responseCache.Put(k, v)
}

func TestResponseCache_Get(t *testing.T) {
	responseCache := NewResponseCache()
	k := uint16(10)
	v := Packet{}
	responseCache.Put(k, v)
	_, ok := responseCache.Get(k)
	if !ok {
		t.Error("should exist")
	}

	nk := uint16(10)
	_, ok = responseCache.Get(nk)
	if ok {
		t.Error("shouldn't exist")
	}
}
