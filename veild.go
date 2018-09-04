// Package veild is the main veil package for handling DNS to DNS-over-TLS connections.
package veild

import (
	"io/ioutil"
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
		log.Println("[cache] \x1b[31;1mCaching on\x1b[0m")
		queryCache = NewQueryCache()
		go queryCache.Reaper()
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

	// Add workers to the pooler.
	resolversList := []byte{}
	if config.ResolversFile == "" {
		log.Println("[pool] No resolvers file given, using default (1.1.1.1 and 1.0.0.1)")
		resolversList = []byte(defaultResolver)
	} else {
		if resolversList, err = ioutil.ReadFile(config.ResolversFile); err != nil {
			log.Fatalln(err)
		}
	}

	resolvers, err := LoadResolvers(resolversList)
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
		buff := make([]byte, 512)
		n, clientAddr, _ := conn.ReadFromUDP(buff)

		// Potential to catch small packets here.
		if n < 12 {
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

	// Handle blacklisted domains if enabled.
	// See: https://en.wikipedia.org/wiki/DNS_sinkhole
	if blacklisting {
		name := parseDomainName(packet.packetData[12:])
		if blacklist.Exists(name) {
			log.Printf("\x1b[31;1m[blacklist] Match: %s\x1b[0m\n", name)
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
		nameType := sliceNameType(packet.packetData[12:])
		cacheKey := createCacheKey(nameType)

		// Get the cached entry if we have one.
		if resp, ok := queryCache.Get(cacheKey); ok {
			log.Printf("[cache] \x1b[32;1mCache hit for 0x%x\x1b[0m\n", cacheKey)
			// Prepend the transaction id to the payload.
			r := append(packet.packetData[:2], resp.(Query).data[2:]...)
			packet.clientConn.WriteToUDP(r, packet.clientAddr)
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

// parseDomainName takes a slice of bytes and returns a parsed domain name.
func parseDomainName(data []byte) string {
	parts := make([]byte, 0)
	i := 0
	for {
		if data[i] == 0x00 {
			break
		}
		if i != 0x00 {
			parts = append(parts, 0x2e)
		}
		l := int(data[i])
		parts = append(parts, data[i+1:i+l+1]...)
		// Increment to next label offset.
		i += l + 1
	}
	return string(parts)
}

// addHostForPort matches if a port that is parsed form resolverAddr matches outboundPort.
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
