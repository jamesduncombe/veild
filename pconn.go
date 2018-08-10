package veild

import (
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
	"time"
)

// PConn holds persistent connections.
type PConn struct {
	t          *Pool
	host       string
	serverName string
	writech    chan Packet
	closech    chan struct{}
	conn       *tls.Conn
	cache      *ResponseCache
	start      time.Time
	lastReq    time.Time
}

// NewPConn creates a new PConn which is an actual connection to an upstream DNS server.
func NewPConn(p *Pool, cache *ResponseCache, host, serverName string) (*PConn, error) {
	pc := &PConn{
		t:          p,
		host:       host,
		serverName: serverName,
		writech:    make(chan Packet, 1),
		closech:    make(chan struct{}),
		cache:      cache,
		start:      time.Now(),
		lastReq:    time.Now(),
	}

	var t time.Duration = 1

retry:
	conn, err := dialConn(host, serverName)
	if err != nil {
		log.Printf("[pconn] Failed to connect to: %s, retrying in %d seconds\n", host, t)
		// Back off for t servers (exponential backoff).
		time.Sleep(t * time.Second)
		t = t << 1
		goto retry
	}

	pc.conn = conn

	go pc.readLoop()
	go pc.writeLoop()
	return pc, nil
}

// dialConn handles dialing the outbound connection to the underlying DNS server.
func dialConn(host, serverName string) (*tls.Conn, error) {
	log.Printf("[pconn] Dialing connection: %s\n", host)

	return tls.Dial("tcp", host, &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	})
}

// readLoop continually tries to read responses from upstream DNS server.
func (p *PConn) readLoop() {

	defer func() {
		log.Printf("[pconn] Closing connection: %s since last request: %v connection lasted: %v\n", p.host, time.Since(p.lastReq), time.Since(p.start))
		p.conn.Close()
		close(p.closech)
	}()

	alive := true
	for alive {

		buff := make([]byte, 512)
		n, err := p.conn.Read(buff)
		if err == io.EOF {
			log.Printf("[pconn] Connection gone away: %s\n", p.host)
			alive = false
			continue
		}

		reqID := binary.BigEndian.Uint16(buff[2:4])

		if val, ok := p.cache.Get(reqID); ok {

			// Shave off first 2 bytes for the length and write back to client over UDP.
			val.(Packet).clientConn.WriteToUDP(buff[2:n], val.(Packet).clientAddr)

			// Calculate ellapsed time since start of request.
			elapsed := time.Since(val.(Packet).start)

			log.Printf("[pool] Trans.ID: 0x%x Query time: %v\n", reqID, elapsed)
		}
	}

}

// writeLoop takes DNS requests and forwards them to the upstream DNS server.
func (p *PConn) writeLoop() {
	for {
		select {
		case wr := <-p.writech:

			// Overwrite time of last request.
			p.lastReq = time.Now()

			// Calculate packet length and pack into uint16 (BigEndian).
			rawLen := len(wr.packetData)
			packetLength := []byte{uint8(rawLen >> 8), uint8(rawLen)}

			// Prepend packet length.
			p.conn.Write(append(packetLength, wr.packetData...))

			// Add to cache.
			reqID := binary.BigEndian.Uint16(wr.packetData[:2])
			p.cache.Put(reqID, wr)

		case <-p.closech:
			close(p.writech)
			return
		}
	}

}
