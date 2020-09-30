// Package veild is the main veil package for handling DNS to DNS-over-TLS connections.
package veild

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
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
	numRequests  int
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

	log.Println("Starting Veil")

	// Setup blacklist.
	if config.BlacklistFile != "" {
		log.Println("[blacklist] Loading blacklist")
		var err error
		blacklist, err = NewBlacklist(config.BlacklistFile)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("[blacklist] Loaded %d entries into the blacklist\n", len(blacklist.list))
		blacklisting = true
	}

	log.Printf("[main] \x1b[31;1mOutbound port set to %d\x1b[0m\n", config.OutboundPort)

	// Setup caching.
	if config.Caching {
		caching = true
		queryCache = NewQueryCache()
		go queryCache.Reaper()
	} else {
		log.Println("[cache] \x1b[31;1mCaching off\x1b[0m")
	}

	// Setup goroutine for handling the exit signals.
	go cleanup()

	// Parse the listener address.
	udpAddr, err := net.ResolveUDPAddr("udp", config.ListenAddr)
	if err != nil {
		log.Fatalln(err)
	}

	// Setup listening for UDP server.
	log.Printf("[main] \x1b[34;1mListening on %s (UDP)\x1b[0m\n", udpAddr)
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Println(err)
		log.Fatalln("Did you specify one of your IP addresses?")
	}
	defer conn.Close()

	// Create the pooler.
	pool := NewPool()
	go pool.ConnectionManagement()
	go pool.Dispatch()

	// Load the list of resolvers.
	resolvers, err := NewResolvers(config.ResolversFile)
	if err != nil {
		log.Fatalln(err)
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
			log.Println("Packet length too small")
			continue
		}

		packet := Packet{
			clientAddr: clientAddr,
			clientConn: conn,
			packetData: buff[:n],
			start:      time.Now()}

		numRequests++

		log.Printf("[stats] Requests: %d\n", numRequests)

		// Spin up new goroutine per request.
		go resolve(pool, packet)
	}
}

// resolve handles individual requests.
func resolve(p *Pool, packet Packet) {

	rr, err := NewRR(packet.packetData[HeaderLength:])
	if err != nil {
		log.Println("[main] Problem handling RR")
		return
	}
	log.Printf("[main] Request for host: \x1b[31;1m%s\x1b[0m rtype: \x1b[31;1m%s\x1b[0m\n", rr.hostname, rr.rType)

	// Handle blacklisted domains if enabled.
	// See: https://en.wikipedia.org/wiki/DNS_sinkhole
	if blacklisting {
		if blacklist.Exists(rr.hostname) {
			log.Printf("\x1b[31;1m[blacklist] Match: %s\x1b[0m\n", rr.hostname)
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
			log.Printf("[cache] \x1b[32;1mCache hit for host: %s rtype: %s\x1b[0m\n", rr.hostname, rr.rType)
			// Prepend the transaction id to the payload.
			responsePacket := append(packet.packetData[:2], data[2:]...)
			packet.clientConn.WriteToUDP(responsePacket, packet.clientAddr)
			return
		}
	}

	// Otherwise, send it on.
	select {
	case p.packets <- packet:
	default:
		log.Println("[main] Dropping oldest request")
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
	if outboundPort == uint(resolverPort) {
		return true
	}
	return false
}

// cleanup handles the exiting of veil.
func cleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Printf("[main] Exiting...\n")
		log.Printf("[stats] Total requests served: %d\n", numRequests)
		os.Exit(0)
	}()
}
