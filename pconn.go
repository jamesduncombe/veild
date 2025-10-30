package veild

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
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
	log        *slog.Logger

	mu      sync.RWMutex
	start   time.Time
	lastReq time.Time
}

// NewPConn creates a new PConn which is an actual connection to an upstream DNS server.
func NewPConn(rc *ResponseCache, worker *Worker, logger *slog.Logger) (*PConn, error) {
	pc := &PConn{
		host:       worker.host,
		serverName: worker.serverName,
		writeCh:    make(chan *Request, 1),
		closeCh:    make(chan struct{}),
		cache:      rc,
		start:      time.Now(),
		lastReq:    time.Now(),
		log:        logger.With("module", "pconn"),
	}

	var t time.Duration = 1

retry:
	pc.log.Info("Dialing connection", "host", pc.host)

	// Reset duration back to 1 if we've exceeded a reasonable backoff.
	if t >= 1024 {
		t = 1
	}

	conn, err := pc.dialConn()
	pc.log.Debug("Dial complete", "host", pc.host)
	if err != nil {
		pc.log.Warn("Failed to connect", "host", pc.host, "reconnecting_in", t*time.Second)
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
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	return tls.DialWithDialer(dialer, "tcp", pc.host, &tls.Config{
		ServerName: pc.serverName,
		MinVersion: tls.VersionTLS13,
	})
}

// readLoop takes responses from the upstream DNS server.
// if the response matching a request that we've made, we write
// it back to the client.
func (pc *PConn) readLoop() {

	defer func() {
		// Lock needed due to accessing times.
		pc.mu.Lock()
		defer pc.mu.Unlock()

		pc.log.Info("Closing connection from read side", "host", pc.host, "last_request", time.Since(pc.lastReq), "lasted", time.Since(pc.start))
		pc.conn.Close()
		close(pc.closeCh)
	}()

	for {
		pc.log.Debug("Reading from upstream DNS server...", "host", pc.host)

		buff := make([]byte, ResponsePacketLength)
		n, err := pc.conn.Read(buff)

		if err != nil {
			pc.log.Info("Connection gone away", "host", pc.host, "err", err)
			return
		}

		// Discard first 2 bytes (packet length) and truncate to length of received bytes
		// as we're returning this over UDP to the client.
		// SEE: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
		buff = buff[2:n]

		trxID := buff[:2]
		key := createCacheKey(trxID)

		if request, ok := pc.cache.Get(key); ok {

			pc.cache.log.Info("Match request cache", "trx_id", fmt.Sprintf("0x%x", trxID))

			if caching {
				offsets, err := ttlOffsets(buff)
				if err != nil {
					pc.cache.log.Warn("Error parsing offsets", "err", err)
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
				pc.log.Warn("Error writing back to client", "err", err, "client_ip", request.clientAddr)
				break
			}
			pc.log.Debug("Wrote bytes back to client", "bytes", n)

			// Calculate ellapsed time since start of request.
			elapsed := time.Since(request.start)

			pc.log.Info("Processed request", "trx_id", fmt.Sprintf("0x%x", trxID), "elapsed", elapsed, "context", "pool")

		} else {
			pc.log.Warn("No matching request in cache", "trx_id", fmt.Sprintf("0x%x", trxID))
		}
	}

}

// writeLoop takes DNS requests and forwards them to the upstream DNS server.
func (pc *PConn) writeLoop() {

	defer func() {
		// Lock needed due to accessing times.
		pc.mu.Lock()
		defer pc.mu.Unlock()

		pc.log.Info("Closing connection from write side", "host", pc.host, "last_request", time.Since(pc.lastReq), "lasted", time.Since(pc.start))
		pc.conn.Close()
	}()

	for {
		pc.log.Debug("Waiting on incoming requests...", "host", pc.host)
		select {
		case request := <-pc.writeCh:

			// Update time of last request.
			pc.mu.Lock()
			pc.lastReq = time.Now()
			pc.mu.Unlock()

			// Calculate packet length and pack into uint16 (BigEndian).
			// Because we're writing this out over TCP we need to prepend the length.
			// SEE: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
			packetLength := make([]byte, 2)
			binary.BigEndian.PutUint16(packetLength, uint16(len(request.data)))

			pc.log.Debug("Writing request to upstream DNS server", "host", pc.host)

			// Prepend packet length as this is over TCP.
			n, err := pc.conn.Write(append(packetLength, request.data...))
			if err != nil {
				pc.log.Warn("Error passing request to upstream", "host", pc.host, "err", err)
				return
			}
			pc.log.Debug("Wrote bytes to server", "host", pc.host, "bytes", n)

			// Add to cache.
			pc.cache.Set(request)

		case <-pc.closeCh:
			pc.log.Debug("Connection closed", "host", pc.host)
			return
		}
	}

}
