package veild

import (
	"bytes"
	"net"
	"os"
	"testing"
	"time"
)

func TestPConn_NewPConn(t *testing.T) {

	// Setup pconn.
	logger := newLogger()
	// TODO: We're using a global query cache which isn't right.
	queryCache = NewQueryCache(logger)
	rc := NewResponseCache(logger)
	worker := NewWorker("9.9.9.9:853", "dns.quad9.net")
	pc, err := NewPConn(rc, worker, logger)

	if err != nil {
		t.Errorf("got error when creating pconn %v", err)
	}

	// Turn caching on.
	caching = true

	// Read in a raw packet.
	rawPacket, _ := os.ReadFile("fixtures/request_protonmail.com_a.pkt")

	// Setup a listener for the return data.
	clientAddr := &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 5355}
	clientConn, err := net.ListenUDP("udp", clientAddr)
	if err != nil {
		t.Errorf("got error when creating udp listener %v", err)
	}

	// Build a request and send it.
	pc.writeCh <- &Request{
		data:       rawPacket,
		clientConn: clientConn,
		clientAddr: clientAddr,
		start:      time.Now(),
	}

	// Look at cache entries.
	var bf bytes.Buffer
	rc.Entries(&bf)
	t.Log(bf.String())

	// Read the response.
	b := make([]byte, ResponsePacketLength)
	n, _, _ := clientConn.ReadFrom(b)
	t.Log(b[:n])

	// Teardown the connection etc.
	pc.conn.Close()
	clientConn.Close()
}

// TODO: Test the backoff mechanism when dialing fails.
