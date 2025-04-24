package veild

import "testing"

func newRequest() *Request {
	return &Request{
		data: []byte{
			0xa, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x6e, 0x6d,
			0x61, 0x69, 0x6c, 0x3, 0x63, 0x6f, 0x6d, 0x0},
	}
}

func TestResponseCache_Set(t *testing.T) {
	responseCache := NewResponseCache()

	v := newRequest()
	responseCache.Set(v)

	if !responseCache.Exists(v.cacheKey()) {
		t.Error("should exist")
	}
}

func TestResponseCache_Get(t *testing.T) {
	responseCache := NewResponseCache()

	v := newRequest()
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
