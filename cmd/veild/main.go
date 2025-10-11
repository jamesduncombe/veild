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
	noCaching     bool
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
	flag.UintVar(&outboundPort, "p", 853, "Default outbound `port` 853 (standard DNS-over-TLS port)")
	flag.BoolVar(&noCaching, "no-cache", false, "If specified, turn off caching")
	flag.StringVar(&blacklistFile, "b", "", "Read `blacklist_file` and enable blacklisting Ad domains")
	flag.StringVar(&resolversFile, "r", "", "Read resolvers from `resolvers_file` and load them")
	flag.BoolVar(&version, "version", false, "Displays the version of Veild")
	flag.Parse()

	if version {
		fmt.Printf("veild: %s\n", veilVersion)
		os.Exit(0)
	}

	// Sort out the config.
	config := &veild.Config{
		ListenAddr:    listenAddr,
		OutboundPort:  outboundPort,
		Caching:       !noCaching,
		BlacklistFile: blacklistFile,
		ResolversFile: resolversFile,
		Version:       veilVersion,
	}

	// Start Veil.
	veild.Run(config)
}
