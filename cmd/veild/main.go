package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jamesduncombe/veild"
)

// veil version.
var veilVersion string

// Flags for setting up veil.
var (
	listenAddr    string
	caching       bool
	outboundPort  uint
	blacklistFile string
	resolversFile string
	version       bool
)

// usage handles the default usage instructions for the cmd.
func usage() {
	fmt.Println(veilVersion)
	fmt.Printf("\nUsage:\n\n")
	flag.PrintDefaults()
	fmt.Println()
}

func main() {

	// Setup usual usage intructions for the cmd.
	flag.Usage = usage

	// Flags init.
	flag.StringVar(&listenAddr, "l", "127.0.0.1:53", "Listen on `address:port` for serving requests")
	flag.UintVar(&outboundPort, "p", 853, "Default outbound `port` (443 (DNS-over-TLS) or 853 (DNS-over-TLS))")
	flag.BoolVar(&caching, "c", true, "Turn off caching (on by default)")
	flag.StringVar(&blacklistFile, "b", "", "Read `blacklist_file` and enable blacklisting Ad domains")
	flag.StringVar(&resolversFile, "r", "", "Read resolvers from `resolvers_file` and load them")
	flag.BoolVar(&version, "v", false, "Version info")
	flag.Parse()

	if version {
		fmt.Println(veilVersion)
		os.Exit(0)
	}

	// Sort out the config.
	config := &veild.Config{
		Version:       veilVersion,
		ListenAddr:    listenAddr,
		Caching:       caching,
		OutboundPort:  outboundPort,
		BlacklistFile: blacklistFile,
		ResolversFile: resolversFile,
	}

	// Start Veil.
	veild.Run(config)
}
