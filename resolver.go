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

// Resolver represents an upstream DNS resolver.
type Resolver struct {
	resolver ResolverEntry
	writeCh  chan *Request
	closeCh  chan struct{}
	doneCh   chan struct{}
	conn     *tls.Conn
	cache    *ResponseCache
	log      *slog.Logger

	mu      sync.RWMutex
	start   time.Time
	lastReq time.Time
}

// NewResolver creates a new Resolver which is an actual connection to an upstream DNS server.
func NewResolver(rc *ResponseCache, re ResolverEntry, logger *slog.Logger) (*Resolver, error) {
	rs := &Resolver{
		resolver: re,
		writeCh:  make(chan *Request, 1),
		closeCh:  make(chan struct{}),
		doneCh:   make(chan struct{}),
		cache:    rc,
		start:    time.Now(),
		lastReq:  time.Now(),
		log:      logger.With("module", "resolver"),
	}

	var t time.Duration = 1

retry:
	rs.log.Info("Dialing connection", "host", rs.resolver.Address)

	// Reset duration back to 1 if we've exceeded a reasonable backoff.
	if t >= 1024 {
		t = 1
	}

	conn, err := rs.dialConn()
	rs.log.Debug("Dial complete", "host", rs.resolver.Address)
	if err != nil {
		rs.log.Warn("Failed to connect", "host", rs.resolver.Address, "reconnecting_in", t*time.Second)
		// Back off for t seconds (exponential backoff).
		time.Sleep(t * time.Second)
		t = t << 1
		goto retry
	}

	// Assign the underlying connection.
	rs.conn = conn

	go rs.readLoop()
	go rs.writeLoop()
	return rs, nil
}

// dialConn handles dialing the outbound connection to the underlying DNS server.
func (rs *Resolver) dialConn() (*tls.Conn, error) {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	return tls.DialWithDialer(dialer, "tcp", rs.resolver.Address, &tls.Config{
		ServerName: rs.resolver.Hostname,
		MinVersion: tls.VersionTLS13,
	})
}

// readLoop takes responses from the upstream DNS server.
// if the response matching a request that we've made, we write
// it back to the client.
func (rs *Resolver) readLoop() {

	defer func() {
		// Lock needed due to accessing times.
		rs.mu.Lock()
		defer rs.mu.Unlock()

		rs.log.Info("Closing connection from read side", "host", rs.resolver.Address, "last_request", time.Since(rs.lastReq), "lasted", time.Since(rs.start))
		rs.conn.Close()
		close(rs.closeCh)
	}()

	for {
		rs.log.Debug("Reading from upstream DNS server...", "host", rs.resolver.Address)

		buff := make([]byte, ResponsePacketLength)
		n, err := rs.conn.Read(buff)

		if err != nil {
			rs.log.Info("Connection gone away", "host", rs.resolver.Address, "err", err)
			return
		}

		// Discard first 2 bytes (packet length) and truncate to length of received bytes
		// as we're returning this over UDP to the client.
		// SEE: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
		buff = buff[2:n]

		trxID := buff[:2]
		key := createCacheKey(trxID)

		if request, ok := rs.cache.Get(key); ok {

			rs.cache.log.Info("Match request cache", "trx_id", fmt.Sprintf("0x%x", trxID))

			if caching {
				offsets, err := ttlOffsets(buff)
				if err != nil {
					rs.cache.log.Warn("Error parsing offsets", "err", err)
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
				rs.log.Warn("Error writing back to client", "err", err, "client_ip", request.clientAddr)
				break
			}
			rs.log.Debug("Wrote bytes back to client", "bytes", n)

			// Calculate ellapsed time since start of request.
			elapsed := time.Since(request.start)

			rs.log.Info("Processed request", "trx_id", fmt.Sprintf("0x%x", trxID), "elapsed", elapsed, "context", "pool")

		} else {
			rs.log.Warn("No matching request in cache", "trx_id", fmt.Sprintf("0x%x", trxID))
		}
	}

}

// writeLoop takes DNS requests and forwards them to the upstream DNS server.
func (rs *Resolver) writeLoop() {

	defer func() {
		// Lock needed due to accessing times.
		rs.mu.Lock()
		defer rs.mu.Unlock()

		rs.log.Info("Closing connection from write side", "host", rs.resolver.Address, "last_request", time.Since(rs.lastReq), "lasted", time.Since(rs.start))
		rs.conn.Close()
	}()

	for {
		rs.log.Debug("Waiting on incoming requests...", "host", rs.resolver.Address)
		select {
		case request := <-rs.writeCh:

			// Update time of last request.
			rs.mu.Lock()
			rs.lastReq = time.Now()
			rs.mu.Unlock()

			// Calculate packet length and pack into uint16 (BigEndian).
			// Because we're writing this out over TCP we need to prepend the length.
			// SEE: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2
			packetLength := make([]byte, 2)
			binary.BigEndian.PutUint16(packetLength, uint16(len(request.data)))

			rs.log.Debug("Writing request to upstream DNS server", "host", rs.resolver.Address)

			// Prepend packet length as this is over TCP.
			n, err := rs.conn.Write(append(packetLength, request.data...))
			if err != nil {
				rs.log.Warn("Error passing request to upstream", "host", rs.resolver.Address, "err", err)
				return
			}
			rs.log.Debug("Wrote bytes to server", "host", rs.resolver.Address, "bytes", n)

			// Add to cache.
			rs.cache.Set(request)

		case <-rs.closeCh:
			rs.log.Debug("Connection closed", "host", rs.resolver.Address)
			return
		}
	}

}
