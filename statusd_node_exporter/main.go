package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	metricsPath = "/metrics"
)

type filtersFlag []string

func (f *filtersFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *filtersFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

var (
	filters filtersFlag
	ipcPath = flag.String("ipc", "", "path to ipc file")
	host    = flag.String("host", "", "http server host")
	port    = flag.Int("port", 9200, "http server port")
)

func usage() {
	flag.Usage()
	os.Exit(1)
}

func requiredFlag(f string) {
	log.Printf("flag -%s is required\n", f)
	usage()
}

func init() {
	flag.Var(&filters, "filter", "regular expression, can be used multiple times")
	flag.Parse()

	if *ipcPath == "" {
		requiredFlag("ipc")
	}

	if flag.NArg() > 0 {
		log.Printf("Extra args in command line: %v", flag.Args())
		usage()
	}
}

func main() {
	c, err := newCollector(*ipcPath, filters)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc(metricsPath, metricsHandler(c))
	http.HandleFunc("/", rootHandler)

	listenAddress := fmt.Sprintf("%s:%d", *host, *port)

	log.Println("Listening on", listenAddress)
	err = http.ListenAndServe(listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
