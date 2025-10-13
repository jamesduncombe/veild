// Package veild is the main veil package for handling DNS to DNS-over-TLS connections.
package veild

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
)

// Config represents the command line options.
type Config struct {
	Version       string
	ListenAddr    string
	Caching       bool
	OutboundPort  uint
	BlacklistFile string
	ResolversFile string
}

var (
	queryCache   *QueryCache
	blacklist    *Blacklist
	numRequests  atomic.Uint64
	caching      bool
	blacklisting = false
)

// Run starts up the app.
func Run(config *Config) {

	mainLog := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelDebug,
		}),
	)

	mainLog.Info("Starting Veil", "version", config.Version)

	mainLog.Debug("Setting outbound port", "outbound", config.OutboundPort)

	// Setup blacklist.
	if config.BlacklistFile != "" {
		var err error
		blacklist, err = NewBlacklist(config.BlacklistFile, mainLog)
		blacklist.log.Info("Loading blacklist")
		if err != nil {
			mainLog.Error(fmt.Sprintf("Error %w", err))
			os.Exit(1)
		}
		blacklist.log.Info("Loading entries into the blacklist", "entries", len(blacklist.list))
		blacklisting = true
	}

	// Setup caching.
	if config.Caching {
		caching = true
		queryCache = NewQueryCache(mainLog)
		go queryCache.Reaper()
	} else {
		queryCache = NewQueryCache(mainLog)
		queryCache.log.Debug("Caching off")
	}

	// Setup goroutine for handling the exit signals.
	go cleanup(mainLog)

	// Parse the listener address.
	udpAddr, err := net.ResolveUDPAddr("udp", config.ListenAddr)
	if err != nil {
		mainLog.Error(fmt.Sprintf("Error listening: %w", err))
		os.Exit(1)
	}

	// Setup listening for UDP server.
	mainLog.Info("Adding listener", "host", udpAddr)
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		mainLog.Error(fmt.Sprintf("Error: %w", err))
		mainLog.Error("Did you specify one of your IP addresses?")
		os.Exit(1)
	}
	defer conn.Close()

	// Create the pooler.
	pool := NewPool(mainLog)
	go pool.ConnectionManagement()
	go pool.Dispatch()

	// Load the list of resolvers.
	resolvers, err := NewResolvers(config.ResolversFile)
	if err != nil {
		mainLog.Error(fmt.Sprintf("Error loading resolvers: %w", err))
		os.Exit(1)
	}

	// Load each resolver into the pool.
	for _, resolver := range resolvers.Resolvers {
		if ok, err := addHostForPort(resolver.Address, config.OutboundPort); ok {

			w := pool.NewWorker(resolver.Address, resolver.Hostname)
			pool.AddWorker(w)
		} else if err != nil {
			mainLog.Warn("Problem parsing resolver address", "address", resolver.Address, "error", err)
			continue
		}
	}

	// Enter the listening loop.
	for {
		buff := make([]byte, DnsPacketLength)
		n, clientAddr, _ := conn.ReadFromUDP(buff)

		// Potential to catch small packets here.
		if n < DnsHeaderLength {
			mainLog.Warn("Packet length too small", "length", n)
			continue
		}

		request := &Request{
			clientAddr: clientAddr,
			clientConn: conn,
			data:       buff[:n],
			start:      time.Now()}

		numRequests.Add(1)

		mainLog.Info("Requests", "requests", numRequests.Load(), "context", "stats")

		// Spin up new goroutine per request.
		go resolve(pool, request, mainLog)
	}
}

// resolve handles individual requests.
func resolve(p *Pool, request *Request, mainLog *slog.Logger) {

	rr, err := NewRR(request.data[DnsHeaderLength:])
	if err != nil {
		mainLog.Warn("Problem handling RR", "error", err)
		return
	}
	mainLog.Info("New request", "host", rr.hostname, "rtype", rr.rType)

	// Handle blacklisted domains if enabled.
	// SEE: https://en.wikipedia.org/wiki/DNS_sinkhole
	if blacklisting && blacklist.Exists(rr.hostname) {
		blacklist.log.Info("Blocklist match", "host", rr.hostname)
		// Reform the query as a response with 0 answers.
		transIDFlags := append(request.data[:2], []byte{0x81, 0x83}...)
		newPacket := append(transIDFlags, request.data[len(transIDFlags):]...)
		request.clientConn.WriteToUDP(newPacket, request.clientAddr)
		return
	}

	// Handle caching if enabled.
	if caching {
		// Create cache key.
		cacheKey := createCacheKey(rr.cacheKey)

		// Get the cached entry if we have one.
		if query, ok := queryCache.Get(cacheKey); ok {
			queryCache.log.Debug("Cache hit", "host", rr.hostname, "rtype", rr.rType)
			// TODO: Check that this lock actually works as expected.
			// Then maybe move the logic into query cache?
			queryCache.mu.Lock()
			// Prepend the transaction id to the payload.
			responsePacket := append(request.data[:2], query.data[2:]...)
			queryCache.mu.Unlock()
			request.clientConn.WriteToUDP(responsePacket, request.clientAddr)
			return
		}
	}

	// Otherwise, send it on.
	select {
	case p.requests <- request:
		p.log.Debug("Request added to pool", "context", "pool")
	default:
		p.log.Debug("Dropping oldest request", "context", "pool")
		<-p.requests
		p.requests <- request
	}
}

// addHostForPort matches if a port that is parsed from resolverAddr matches outboundPort.
func addHostForPort(resolverAddr string, outboundPort uint) (bool, error) {
	_, rport, err := net.SplitHostPort(resolverAddr)
	if err != nil {
		return false, err
	}

	resolverPort, err := strconv.Atoi(rport)
	if err != nil {
		return false, err
	}

	return outboundPort == uint(resolverPort), nil
}

// cleanup handles the exiting of veil.
func cleanup(mainLog *slog.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	mainLog.Info("Exiting...")
	mainLog.Info("Total requests served", "total", numRequests.Load(), "context", "stats")

	os.Exit(0)
}
