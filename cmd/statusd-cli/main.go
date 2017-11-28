package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

const (
	// StatusDHost is the default host to connect to.
	StatusDHost = "localhost"

	// StatusDPort is the default port to connect to.
	StatusDPort = "51515"
)

var (
	statusdHost = flag.String("statusd", StatusDHost, "set host of statusd connection")
	statusdPort = flag.String("statusdport", StatusDPort, "set port of statusd connection")
)

// main is the entrypoint for the statusd command line interface.
func main() {
	flag.Usage = printUsage
	flag.Parse()

	host := net.JoinHostPort(*statusdHost, *statusdPort)

	fmt.Printf("statusd-cli connecting statusd on '%s'\n", host)

	// Running REPL.
	repl := NewREPL(host)
	err := repl.Run()
	if err != nil {
		fmt.Printf("stopped with error: %v\n", err)
		os.Exit(-1)
	}
}

// printUsage prints a little help for statusd-cli.
func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: statusd-cli [options]")
	fmt.Fprintf(os.Stderr, `
Examples:
  statusd-cli -statusd=<host> # contact statusd on host
  
Options:
`)
	flag.PrintDefaults()
}
