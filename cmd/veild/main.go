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
	blocklistFile string
	resolversFile string
	logLevel      string
	version       bool
)

func main() {

	// Setup usual usage intructions for the cmd.
	flag.Usage = usage

	// Flags init.
	flag.StringVar(&listenAddr, "l", "127.0.0.1:53", "Listen on `address:port` for serving requests")
	flag.BoolVar(&noCaching, "no-cache", false, "If specified, turn off caching")
	flag.StringVar(&blocklistFile, "b", "", "Read `blocklist_file` and enable blocklisting Ad domains")
	flag.StringVar(&resolversFile, "r", "", "Read resolvers from `resolvers_file` and load them")
	flag.StringVar(&logLevel, "log-level", "info", "Set the logging level (debug, info, warn)")
	flag.BoolVar(&version, "version", false, "Displays the version of Veild")
	flag.Parse()

	if version {
		fmt.Printf("veild: %s\n", veilVersion)
		os.Exit(0)
	}

	// Start Veil.
	veild.Run(&veild.Config{
		ListenAddr:     listenAddr,
		CachingEnabled: !noCaching,
		BlocklistFile:  blocklistFile,
		ResolversFile:  resolversFile,
		LogLevel:       veild.ParseLogLevel(logLevel),
		Version:        veilVersion,
	})
}

// usage handles the default usage instructions for the cmd.
func usage() {
	fmt.Println(veilVersion)
	fmt.Printf("\nUsage:\n\n")
	flag.PrintDefaults()
	fmt.Println()
}
