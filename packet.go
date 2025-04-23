package veild

import (
	"net"
	"time"
)

// Packet represents the structure of a client request.
type Packet struct {
	clientAddr *net.UDPAddr
	clientConn *net.UDPConn
	data       []byte
	start      time.Time
}

func (p Request) cacheKey() cacheKey {
	return createCacheKey(p.data[:2])
}
