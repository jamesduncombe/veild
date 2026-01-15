package veild

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// MockClientConn is a mock implementation of RequestConn for testing.

type MockClientConn struct {
	readBlockerCh chan struct{}
}

func NewMockClientConn() *MockClientConn {
	return &MockClientConn{
		readBlockerCh: make(chan struct{}),
	}
}

func (m MockClientConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	// Block until we're told to continue by read.
	<-m.readBlockerCh

	// Just return the length of what was written.
	return len(b), nil
}

func (m MockClientConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	defer func() {
		// Allow WriteToUDP to continue.
		m.readBlockerCh <- struct{}{}
	}()

	// Form a dummy response.
	rawPacket, _ := os.ReadFile("fixtures/response_protonmail.com_a.pkt")

	copy(b, rawPacket)
	return len(rawPacket), &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 5355}, nil
}

// MockUpstreamResolver is a mock implementation of io.ReadWriteCloser for testing.

type MockUpstreamResolver struct {
	coordCh chan struct{}
}

func NewMockUpstreamResolver() *MockUpstreamResolver {
	return &MockUpstreamResolver{
		coordCh: make(chan struct{}),
	}
}

func (r MockUpstreamResolver) Read(p []byte) (n int, err error) {
	// Block until we're told to continue by read.
	<-r.coordCh

	// Form a dummy response.
	rawPacket, _ := os.ReadFile("fixtures/response_protonmail.com_a.pkt")

	packetLength := make([]byte, 2)
	binary.BigEndian.PutUint16(packetLength, uint16(len(rawPacket)))

	// Prepend the length as this is "over TCP".
	copy(p, append(packetLength, rawPacket...))

	return 2 + len(rawPacket), nil
}

func (r MockUpstreamResolver) Write(p []byte) (n int, err error) {
	defer func() {
		// TODO: There's a timing issue here where the response cache doesn't
		// have the entry before the read happens.
		time.Sleep(20 * time.Millisecond)
		// Allow Read to continue.
		r.coordCh <- struct{}{}
	}()
	// Just return the length of what was written.
	return len(p), nil
}

// Close is a no-op for the Rwcr.
func (r MockUpstreamResolver) Close() error {
	return nil
}

// NullResolverDialer is a mock implementation of ResolverDialer for testing.
type NullResolverDialer struct{}

func (t NullResolverDialer) DialConn(re ResolverEntry) (io.ReadWriteCloser, error) {
	// Return our MockUpstreamResolver.
	b := NewMockUpstreamResolver()
	return b, nil
}

func TestResolver_NewResolver(t *testing.T) {

	t.Skip("skipping NewResolver test")

	// Setup.
	logger := newLogger()
	// TODO: We're using a global query cache which isn't right.
	queryCache = NewQueryCache(logger)
	rc := NewResponseCache(logger)
	re := ResolverEntry{Hostname: "dns.quad9.net", Address: "9.9.9.9:853"}

	rd := NullResolverDialer{}

	rs, err := NewResolver(rc, re, rd, logger)

	if err != nil {
		t.Errorf("got error when creating resolver %v", err)
	}

	// Turn caching on.
	caching = true

	// Read in a raw packet.
	rawPacket, _ := os.ReadFile("fixtures/request_protonmail.com_a.pkt")

	// Setup a listener for the return data.
	clientAddr := &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 5355}
	clientConn := NewMockClientConn()

	// Build a request and send it.
	rs.writeCh <- &Request{
		data:       rawPacket,
		clientConn: clientConn,
		clientAddr: clientAddr,
		start:      time.Now(),
	}

	// Read the response.
	b := make([]byte, ResponsePacketLength)
	n, _, _ := clientConn.ReadFrom(b)
	rawPacketResponse, _ := os.ReadFile("fixtures/response_protonmail.com_a.pkt")
	if !bytes.Equal(b[:n], rawPacketResponse) {
		t.Errorf("expected response packet to match fixture")
	}

	// Look at cache entries.
	var bf bytes.Buffer
	queryCache.Entries(&bf)
	want := "protonmail.com, A, [361]\n"
	got := bf.String()
	if got != want {
		t.Errorf("expected cache entries to include cached query, got %q, want %q", got, want)
	}

	// Teardown the connection etc.
	rs.conn.Close()
}

// TODO: Test the backoff mechanism when dialing fails.
