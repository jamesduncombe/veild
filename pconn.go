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
	host       string
	serverName string
	writeCh    chan *Request
	closeCh    chan struct{}
	conn       *tls.Conn
	cache      *ResponseCache
	log        *log.Logger

	mu      sync.RWMutex
	start   time.Time
	lastReq time.Time
}

// NewPConn creates a new PConn which is an actual connection to an upstream DNS server.
func NewPConn(rc *ResponseCache, worker *Worker) (*PConn, error) {
	pc := &PConn{
		host:       worker.host,
		serverName: worker.serverName,
		writeCh:    make(chan *Request, 1),
		closeCh:    make(chan struct{}),
		cache:      rc,
		start:      time.Now(),
		lastReq:    time.Now(),
		log:        log.New(os.Stdout, "[pconn] ", log.LstdFlags|log.Lmsgprefix),
	}

	var t time.Duration = 1

retry:
	pc.log.Printf("Dialing connection: %s\n", pc.host)

	// Reset duration back to 1 if we've exceeded a reasonale backoff.
	if t >= 1024 {
		t = 1
	}

	conn, err := pc.dialConn()
	if err != nil {
		pc.log.Printf("Failed to connect to: %s, retrying in %d seconds\n", pc.host, t)
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
func (pc *PConn) dialConn() (*tls.Conn, error) {
	return tls.Dial("tcp", pc.host, &tls.Config{
		ServerName: pc.serverName,
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
		close(pc.closeCh)
	}()

	for {

		buff := make([]byte, ResponsePacketLength)
		n, err := pc.conn.Read(buff)

		// On any error exit.
		// TODO: Probably should at least wrap this error.
		if err != nil {
			pc.log.Printf("Connection gone away: %s\n", pc.host)
			return
		}

		// Discard first 2 bytes (packet length) and truncate to length of received bytes.
		// SEE: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
		buff = buff[2:n]

		rawRequestId := buff[:2]
		key := createCacheKey(rawRequestId)
		reqID := binary.BigEndian.Uint16(rawRequestId)

		if request, ok := pc.cache.Get(key); ok {

			pc.cache.log.Printf("Match for \x1b[31;1m0x%x\x1b[0m\n", reqID)

			if caching {
				offsets, err := ttlOffsets(buff)
				if err != nil {
					pc.cache.log.Printf("\x1b[35;1m%v\x1b[0m\n", err)
					continue
				}

				// Only cache if there are TTLs to decrement otherwise the cache will
				// get filled with entries that can't be evicted.
				if len(offsets) > 0 {
					queryCache.Set(&Query{buff, offsets, time.Now()})
				}
			}

			// Write back to client over UDP.
			_, err = request.clientConn.WriteToUDP(buff, request.clientAddr)
			if err != nil {
				pc.log.Printf("Error writting back to client\n")
				break
			}
			pc.log.Printf("Wrote %v back to client\n", n)

			// Calculate ellapsed time since start of request.
			elapsed := time.Since(request.start)

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
		case request := <-pc.writeCh:

			// Overwrite time of last request.
			pc.mu.Lock()
			pc.lastReq = time.Now()
			pc.mu.Unlock()

			// Calculate packet length and pack into uint16 (BigEndian).
			// Because we're writing this out over TCP we need to prepend the length.
			// SEE: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
			packetLength := make([]byte, 2)
			binary.BigEndian.PutUint16(packetLength, uint16(len(request.data)))

			// Prepend packet length.
			pc.conn.Write(append(packetLength, request.data...))

			// Add to cache.
			pc.cache.Set(request)

		case <-pc.closeCh:
			close(pc.writeCh)
			return
		}
	}

}
