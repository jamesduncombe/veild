package veild

import (
	"crypto/sha1"
	"log"
	"net"
	"time"
)

// Config represents the command line options for Veil.
type Config struct {
	ListenAddr    string
	Caching       bool
	BlacklistFile string
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

// Run starts up Veild.
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

	// Setup caching.
	if config.Caching {
		caching = true
		log.Println("[cache] \x1b[31;1mCaching on\x1b[0m")
		queryCache = &QueryCache{
			queries: make(map[[sha1.Size]byte]Query),
		}

		go queryCache.Reaper()
	}

	// Parse the listener address.
	udpAddr, err := net.ResolveUDPAddr("udp", config.ListenAddr)
	if err != nil {
		log.Fatalln(err)
	}

	// Setup listening for UDP server.
	log.Printf("[main] Listening on %s (UDP)\n", udpAddr)
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
	pool.NewWorker("9.9.9.9:853", "dns.quad9.net")

	// Enter the listening loop.
	for {
		buff := make([]byte, 512)
		n, clientAddr, _ := conn.ReadFromUDP(buff)
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
			first := append(packet.packetData[:2], []byte{0x81, 0x83}...)
			j := append(first, packet.packetData[4:]...)
			packet.clientConn.WriteToUDP(j, packet.clientAddr)
			return
		}
	}

	// Handle caching if enabled.
	if caching {
		// Create cache key.
		nameType := sliceNameType(packet.packetData[12:])
		s := createCacheKey(nameType)

		// Get the cached entry if we have one.
		if resp, ok := queryCache.Get(s); ok {
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
	var parts []byte
	i := 0
	for {
		if data[i] == 0x00 {
			break
		}
		if i != 0 {
			parts = append(parts, []byte{0x2e}...)
		}
		l := int(data[i])
		parts = append(parts, data[i+1:i+l+1]...)
		i += l + 1 // increment to next label offset
	}
	return string(parts)
}
