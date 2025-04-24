package veild

import (
	"net"
	"time"
)

// Request represents the structure of a client request.
type Request struct {
	clientAddr *net.UDPAddr
	clientConn *net.UDPConn
	data       []byte
	start      time.Time
}

func (r *Request) cacheKey() cacheKey {
	return createCacheKey(r.data[:2])
}
