package veild

import (
	"crypto/tls"
	"encoding/binary"
	"log"
	"os"
	"sync"
	"time"
)

// ResponsePacketLength represents the buff size for the response packet.
const ResponsePacketLength = 2048

// PConn holds persistent connections.
type PConn struct {
	mu         sync.RWMutex
	host       string
	serverName string
	writech    chan Packet
	closech    chan struct{}
	conn       *tls.Conn
	cache      *ResponseCache
	start      time.Time
	lastReq    time.Time
	log        *log.Logger
}

// NewPConn creates a new PConn which is an actual connection to an upstream DNS server.
func NewPConn(rc *ResponseCache, host, serverName string) (*PConn, error) {
	pc := &PConn{
		host:       host,
		serverName: serverName,
		writech:    make(chan Packet, 1),
		closech:    make(chan struct{}),
		cache:      rc,
		start:      time.Now(),
		lastReq:    time.Now(),
		log:        log.New(os.Stdout, "[pconn] ", log.LstdFlags|log.Lmsgprefix),
	}

	var t time.Duration = 1

retry:
	pc.log.Printf("Dialing connection: %s\n", host)

	conn, err := dialConn(host, serverName)
	if err != nil {
		pc.log.Printf("Failed to connect to: %s, retrying in %d seconds\n", host, t)
		// Back off for t seconds (exponential backoff).
		time.Sleep(t * time.Second)
		t = t << 1
		goto retry
	}

	// Assign the underlying connection.
	pc.conn = conn

	go pc.readLoop()
	go pc.writeLoop()
	return pc, nil
}

// dialConn handles dialing the outbound connection to the underlying DNS server.
func dialConn(host, serverName string) (*tls.Conn, error) {
	return tls.Dial("tcp", host, &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS13,
	})
}

// readLoop continually tries to read responses from upstream DNS server.
func (pc *PConn) readLoop() {

	defer func() {
		// Lock needed due to accessing times.
		pc.mu.Lock()
		defer pc.mu.Unlock()

		pc.log.Printf("Closing connection: %s since last request: %v connection lasted: %v\n", pc.host, time.Since(pc.lastReq), time.Since(pc.start))
		pc.conn.Close()
		close(pc.closech)
	}()

	for {

		buff := make([]byte, ResponsePacketLength)
		n, err := pc.conn.Read(buff)

		pc.log.Printf("[debug] Buff used: %v - size: %v\n", n, len(buff))
		buff = buff[:n]

		// On any error exit.
		if err != nil {
			pc.log.Printf("Connection gone away: %s\n", pc.host)
			break
		}

		reqID := binary.BigEndian.Uint16(buff[2:4])

		if val, ok := pc.cache.Get(reqID); ok {

			pc.cache.log.Printf("Match for \x1b[31;1m0x%x\x1b[0m\n", reqID)

			if caching {
				// Write response to cache.
				nameType, err := sliceNameType(buff[2+12:])
				if err != nil {
					pc.cache.log.Printf("\x1b[35;1m%v\x1b[0m\n", err)
					continue
				}

				cacheKey := createCacheKey(nameType)
				offsets, err := ttlOffsets(buff[2:])
				if err != nil {
					pc.cache.log.Printf("\x1b[35;1m%v\x1b[0m\n", err)
					continue
				}
				queryCache.Put(cacheKey, Query{buff[2:], offsets, time.Now()})
			}

			// Shave off first 2 bytes for the length and write back to client over UDP.
			n, err := val.(Packet).clientConn.WriteToUDP(buff[2:], val.(Packet).clientAddr)
			if err != nil {
				pc.log.Printf("Error writting back to client\n")
				break
			}
			pc.log.Printf("Wrote %v back to client\n", n)

			// Calculate ellapsed time since start of request.
			elapsed := time.Since(val.(Packet).start)

			pc.log.Printf("[pool] Trans.ID: \x1b[31;1m0x%x\x1b[0m Query time: \x1b[31;1m%v\x1b[0m\n",
				reqID,
				elapsed,
			)
		}
	}

}

// writeLoop takes DNS requests and forwards them to the upstream DNS server.
func (pc *PConn) writeLoop() {
	for {
		select {
		case wr := <-pc.writech:

			// Overwrite time of last request.
			pc.mu.Lock()
			pc.lastReq = time.Now()
			pc.mu.Unlock()

			// Calculate packet length and pack into uint16 (BigEndian).
			rawLen := len(wr.packetData)
			packetLength := []byte{uint8(rawLen >> 8), uint8(rawLen)}

			// Prepend packet length.
			pc.conn.Write(append(packetLength, wr.packetData...))

			// Add to cache.
			reqID := binary.BigEndian.Uint16(wr.packetData[:2])
			pc.cache.Put(reqID, wr)

		case <-pc.closech:
			close(pc.writech)
			return
		}
	}

}
