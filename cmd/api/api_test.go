package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/status-go/cmd/api"
)

// TestStartStopServer tests starting the server without any client
// connection. It is actively killed by using a cancel context.
func TestStartStopServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s, err := api.NewServer(ctx, "localhost", "12345")
	if err != nil {
		t.Errorf("cannot create server: %v", err)
	}
	if s == nil {
		t.Errorf("no server returned")
	}

	if s.Err() != nil {
		t.Errorf("server has returned error: %v", s.Err())
	}

	cancel()

	// Sadly have to wait until cancel() has terminated server.
	time.Sleep(1 * time.Millisecond)

	if s.Err() != context.Canceled {
		t.Errorf("server has returned illegl termination reason: %v", s.Err())
	}
}

// TestConnectClient test starting the server and connecting it
// with a client.
func TestConnectClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := api.NewServer(ctx, "[::1]", "12345")
	if err != nil {
		t.Errorf("cannot create server: %v", err)
	}

	c, err := api.NewClient("[::1]", "12345")
	if err != nil {
		t.Errorf("cannot create client: %v", err)
	}

	addrs, err := c.AdminGetAddresses()
	if err != nil {
		t.Errorf("cannot retrieve addresses: %v", err)
	}
	if len(addrs) == 0 {
		t.Errorf("retrieved no addresses")
	}

	if s.Err() != nil {
		t.Errorf("server didn't survived request: %v", s.Err())
	}
}
