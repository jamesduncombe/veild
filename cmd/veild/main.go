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
	version       bool
	blacklistFile string
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
	flag.BoolVar(&caching, "c", false, "Turn on caching (off by default)")
	flag.StringVar(&blacklistFile, "b", "", "Read `blacklist` file and enable blacklisting Ad domains")
	flag.BoolVar(&version, "v", false, "Version info")
	flag.Parse()

	if version {
		fmt.Println(veilVersion)
		os.Exit(0)
	}

	// Sort out the config.
	config := &veild.Config{
		ListenAddr:    listenAddr,
		BlacklistFile: blacklistFile,
		Caching:       caching,
	}

	// Start Veil.
	veild.Run(config)
}
