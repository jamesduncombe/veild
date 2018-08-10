package veild

import (
	"log"
	"net"
	"time"
)

// Config represents the command line options for Veil.
type Config struct {
	ListenAddr string
}

// Packet represents the structure of a client request.
type Packet struct {
	clientAddr *net.UDPAddr
	clientConn *net.UDPConn
	packetData []byte
	start      time.Time
}

var (
	numRequests int
)

// Run starts up Veild.
func Run(config *Config) {

	log.Println("Starting Veil")

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

	select {
	case p.packets <- packet:
	default:
		log.Println("[main] Dropping oldest request")
		<-p.packets
		p.packets <- packet
	}

}
