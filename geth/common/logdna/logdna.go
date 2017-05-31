package logdna

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// IngestBaseURL is the base URL for the LogDNA ingest API.
const IngestBaseURL = "https://logs.logdna.com/logs/ingest"

// FlushInterval is the default interval at which logs are being flushed
const FlushInterval = 15 * time.Second

// BufferSize is the default size of buffer before logs are flushed
const BufferSize = 50

// Config is used to configure logging clients
type Config struct {
	APIKey        string        // LogDNA API access key
	HostName      string        // unique ID describing the source of the incoming logs
	AppName       string        // application name that generates logs (consider including app version)
	FlushInterval time.Duration // how often to flush logs
	BufferSize    int           // how many items to aggregate before flushing
}

// Client is a client to the LogDNA logging service.
type Client struct {
	sync.Mutex
	config  *Config
	payload payload
	apiURL  *url.URL
	stopped chan struct{} // channel to wait for termination
}

// logLine a single log line in the LogDNA ingest API JSON payload.
type logLine struct {
	Timestamp int64  `json:"timestamp"`
	AppName   string `json:"app"`
	Level     string `json:"level"`
	Line      string `json:"line"`
}

// payload is the JSON payload that will be sent to the LogDNA ingest API.
// it serves as buffer, and accumulates log lines (up until buffer size is
// reached or flush interval is triggered)
type payload struct {
	Lines []logLine `json:"lines"`
}

// NewClient creates new client
func NewClient(config *Config) (*Client, error) {
	if config.BufferSize == 0 {
		config.BufferSize = BufferSize
	}

	if config.FlushInterval == 0 {
		config.FlushInterval = FlushInterval
	}

	apiURL, err := makeIngestURL(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		config:  config,
		apiURL:  apiURL,
		stopped: make(chan struct{}),
	}, nil
}

// Log adds a single line to remote logger queue (actual delivery happens asynchronously)
func (c *Client) Log(t time.Time, level string, line string) error {
	logLine := logLine{
		Timestamp: t.UnixNano() / 1000000,
		AppName:   c.config.AppName,
		Level:     level,
		Line:      line,
	}

	c.Lock()
	defer c.Unlock()

	c.payload.Lines = append(c.payload.Lines, logLine)
	if len(c.payload.Lines) >= c.config.BufferSize {
		go c.Flush()
	}

	return nil
}

// Flush dumps pending logs to remote service
func (c *Client) Flush() error {
	c.Lock()
	defer c.Unlock()

	if len(c.payload.Lines) == 0 { // nothing to flush
		return nil
	}

	payloadJSON, err := json.Marshal(c.payload)
	if err != nil {
		return err
	}
	c.payload.Lines = nil

	go func() {
		resp, err := http.Post(c.apiURL.String(), "application/json", bytes.NewReader(payloadJSON))
		if err != nil {
			return
		}
		resp.Body.Close()
	}()

	return nil
}

// Start begins log flushing loop
func (c *Client) Start() {
	go func() {
		for {
			select {
			case <-time.After(c.config.FlushInterval):
				c.Flush()
			case <-c.stopped:
				c.stopped = make(chan struct{})
				return
			}
		}
	}()
}

// Stop terminates log flushing loop
func (c *Client) Stop() {
	close(c.stopped)
}

// makeIngestURL creates a new URL to LogDNA ingest API endpoint.
// The URL is populated with API key and other required parameters.
func makeIngestURL(config *Config) (*url.URL, error) {
	u, err := url.Parse(IngestBaseURL)
	if err != nil {
		return nil, err
	}

	u.User = url.User(config.APIKey)
	values := url.Values{}
	values.Set("hostname", config.HostName)
	values.Set("now", strconv.FormatInt(time.Time{}.UnixNano(), 10))
	u.RawQuery = values.Encode()

	return u, nil
}
