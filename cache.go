package veild

import (
	"hash/fnv"
)

// Overall cache key type.
type cacheKey = uint64

// createCacheKey generates a cache key from a slice of bytes.
func createCacheKey(key []byte) cacheKey {
	f := fnv.New64()
	f.Write(key)
	return cacheKey(f.Sum64())
}
