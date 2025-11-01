package veild

import (
	"crypto/tls"
	"io"
	"net"
	"time"
)

type TLSResolverDialer struct{}

// dialConn handles dialing the outbound connection to the underlying DNS server.
func (t TLSResolverDialer) DialConn(re ResolverEntry) (io.ReadWriteCloser, error) {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	return tls.DialWithDialer(dialer, "tcp", re.Address, &tls.Config{
		ServerName: re.Hostname,
		MinVersion: tls.VersionTLS13,
	})
}
