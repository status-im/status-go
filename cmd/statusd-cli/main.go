package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	// Addr is the default statusd address to connect to.
	Addr = "localhost:51515"
)

var (
	statusdAddr = flag.String("addr", Addr, "set statusd address (default localhost:51515)")
)

// main is the entrypoint for the statusd command line interface.
func main() {
	flag.Usage = printUsage
	flag.Parse()

	fmt.Printf("statusd-cli connecting statusd on '%s'\n", *statusdAddr)

	// Running REPL.
	repl := NewREPL(*statusdAddr)
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
  statusd-cli -addr=<address> # connect statusd on <address>
  
Options:
`)
	flag.PrintDefaults()
}
