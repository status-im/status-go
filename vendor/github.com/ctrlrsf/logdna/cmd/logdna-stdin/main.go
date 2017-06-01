package main

import "bufio"
import "flag"
import "fmt"
import "os"
import "time"
import "github.com/ctrlrsf/logdna"

func main() {
	apiKey := os.Getenv("LOGDNA_API_KEY")

	if apiKey == "" {
		fmt.Println("Set LOGDNA_API_KEY env var")
		os.Exit(1)
	}

	hostname := flag.String("hostname", "", "hostname you want logs to appear from in LogDNA viewer")
	logFileName := flag.String("log-file-name", "", "log file or app name you want logs to appear as in LogDNA viewer")

	flag.Parse()

	if *hostname == "" {
		fmt.Println("Error: hostname flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *logFileName == "" {
		fmt.Println("Error: log-file-name flag is required")
		flag.Usage()
		os.Exit(1)
	}

	cfg := logdna.Config{}
	cfg.APIKey = apiKey
	cfg.Hostname = *hostname
	cfg.LogFile = *logFileName

	client := logdna.NewClient(cfg)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		client.Log(time.Time{}, scanner.Text())
	}

	if scanner.Err() != nil {
		fmt.Fprintln(os.Stderr, "Error reading from stdin: %v", scanner.Err())
		client.Flush()
		os.Exit(1)
	}

	client.Flush()
}
