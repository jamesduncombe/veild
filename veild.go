// Package veild is the main veil package for handling DNS to DNS-over-TLS connections.
package veild

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

// Config represents the command line options.
type Config struct {
	ListenAddr    string
	Caching       bool
	OutboundPort  uint
	BlacklistFile string
	ResolversFile string
}

// Packet represents the structure of a client request.
type Packet struct {
	clientAddr *net.UDPAddr
	clientConn *net.UDPConn
	packetData []byte
	start      time.Time
}

var (
	queryCache   *QueryCache
	blacklist    *Blacklist
	numRequests  atomic.Uint64
	caching      bool
	blacklisting = false
)

const (
	// PacketLength is the maximum allowed packet length for a DNS packet.
	PacketLength int = 512

	// HeaderLength is the length of a normal DNS request/response header (in bytes).
	HeaderLength int = 12
)

// Run starts up the app.
func Run(config *Config) {

	mainLog := log.New(os.Stdout, "[main] ", log.LstdFlags|log.Lmsgprefix)

	mainLog.Println("Starting Veil")

	mainLog.Printf("\x1b[31;1mOutbound port set to %d\x1b[0m\n", config.OutboundPort)

	// Setup blacklist.
	if config.BlacklistFile != "" {
		var err error
		blacklist, err = NewBlacklist(config.BlacklistFile)
		blacklist.log.Println("Loading blacklist")
		if err != nil {
			mainLog.Fatal(err)
		}
		blacklist.log.Printf("Loaded %d entries into the blacklist\n", len(blacklist.list))
		blacklisting = true
	}

	// Setup caching.
	if config.Caching {
		caching = true
		queryCache = NewQueryCache()
		go queryCache.Reaper()
	} else {
		queryCache.log.Println("\x1b[31;1mCaching off\x1b[0m")
	}

	// Setup goroutine for handling the exit signals.
	go cleanup(mainLog)

	// Parse the listener address.
	udpAddr, err := net.ResolveUDPAddr("udp", config.ListenAddr)
	if err != nil {
		mainLog.Fatalln(err)
	}

	// Setup listening for UDP server.
	mainLog.Printf("\x1b[34;1mListening on %s (UDP)\x1b[0m\n", udpAddr)
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		mainLog.Println(err)
		mainLog.Fatalln("Did you specify one of your IP addresses?")
	}
	defer conn.Close()

	// Create the pooler.
	pool := NewPool()
	go pool.ConnectionManagement()
	go pool.Dispatch()

	// Load the list of resolvers.
	resolvers, err := NewResolvers(config.ResolversFile)
	if err != nil {
		mainLog.Fatalln(err)
	}

	// Load each resolver into the pool.
	for _, resolver := range resolvers.Resolvers {
		if ok := addHostForPort(resolver.Address, config.OutboundPort); ok {
			pool.NewWorker(resolver.Address, resolver.Hostname)
		}
	}

	// Enter the listening loop.
	for {
		buff := make([]byte, PacketLength)
		n, clientAddr, _ := conn.ReadFromUDP(buff)

		// Potential to catch small packets here.
		if n < HeaderLength {
			mainLog.Println("Packet length too small")
			continue
		}

		packet := Packet{
			clientAddr: clientAddr,
			clientConn: conn,
			packetData: buff[:n],
			start:      time.Now()}

		numRequests.Add(1)

		mainLog.Printf("[stats] Requests: %d\n", numRequests.Load())

		// Spin up new goroutine per request.
		go resolve(pool, packet, mainLog)
	}
}

// resolve handles individual requests.
func resolve(p *Pool, packet Packet, mainLog *log.Logger) {

	rr, err := NewRR(packet.packetData[HeaderLength:])
	if err != nil {
		mainLog.Println("Problem handling RR")
		mainLog.Println(err)
		return
	}
	mainLog.Printf("Request for host: \x1b[31;1m%s\x1b[0m rtype: \x1b[31;1m%s\x1b[0m\n", rr.hostname, rr.rType)

	// Handle blacklisted domains if enabled.
	// See: https://en.wikipedia.org/wiki/DNS_sinkhole
	if blacklisting {
		if blacklist.Exists(rr.hostname) {
			blacklist.log.Printf("\x1b[31;1mMatch: %s\x1b[0m\n", rr.hostname)
			// Reform the query as a response with 0 answers.
			transIDFlags := append(packet.packetData[:2], []byte{0x81, 0x83}...)
			newPacket := append(transIDFlags, packet.packetData[4:]...)
			packet.clientConn.WriteToUDP(newPacket, packet.clientAddr)
			return
		}
	}

	// Handle caching if enabled.
	if caching {
		// Create cache key.
		cacheKey := createCacheKey(rr.cacheKey)

		// Get the cached entry if we have one.
		if data, ok := queryCache.Get(cacheKey); ok {
			queryCache.log.Printf("\x1b[32;1mCache hit for host: %s rtype: %s\x1b[0m\n", rr.hostname, rr.rType)
			// TODO: Check that this lock actually works as expected.
			// Then maybe move the logic into query cache?
			queryCache.mu.Lock()
			// Prepend the transaction id to the payload.
			responsePacket := append(packet.packetData[:2], data[2:]...)
			queryCache.mu.Unlock()
			packet.clientConn.WriteToUDP(responsePacket, packet.clientAddr)
			return
		}
	}

	// Otherwise, send it on.
	select {
	case p.packets <- packet:
	default:
		p.log.Println("[main] Dropping oldest request")
		<-p.packets
		p.packets <- packet
	}

}

// addHostForPort matches if a port that is parsed from resolverAddr matches outboundPort.
func addHostForPort(resolverAddr string, outboundPort uint) bool {
	_, rport, err := net.SplitHostPort(resolverAddr)
	if err != nil {
		log.Fatalln(err)
	}
	resolverPort, _ := strconv.Atoi(rport)
	return outboundPort == uint(resolverPort)
}

// cleanup handles the exiting of veil.
func cleanup(mainLog *log.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		mainLog.Printf("Exiting...\n")
		mainLog.Printf("[stats] Total requests served: %d\n", numRequests.Load())

		os.Exit(0)
	}()
}
