package veild

import (
	"net"
	"time"
)

// Request represents the structure of a client request.
type Request struct {
	clientAddr *net.UDPAddr
	clientConn RequestConn
	data       []byte
	start      time.Time
}

func (r *Request) cacheKey() cacheKey {
	return createCacheKey(r.data[:2])
}

// RequestConn is an interface for writing to UDP connections.
type RequestConn interface {
	WriteToUDP([]byte, *net.UDPAddr) (int, error)
	ReadFrom(b []byte) (int, net.Addr, error)
}
