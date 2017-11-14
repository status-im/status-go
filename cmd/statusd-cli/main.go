package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	statusdConn = flag.String("statusd", "localhost:12345", "set host and port of statusd")
)

// main is the entrypoint for the statusd command line interface.
func main() {
	flag.Usage = printUsage
	flag.Parse()
}

// printUsage prints a little help for statusd-cli.
func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: statusd-cli [options]")
	fmt.Fprintf(os.Stderr, `
Examples:
  statusd-cli -statusd=<host>:<port> # contact statusd on host and port
  
Options:
`)
	flag.PrintDefaults()
}
