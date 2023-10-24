package pairing

import (
	"testing"
	"time"
)

// run this test assumes provider/consumer are running on the same device
func TestSsdp(t *testing.T) {
	quitProvider := make(chan struct{}, 1)
	StartSsdpProvider(quitProvider)
	quitConsumer := make(chan struct{}, 1)
	result := StartSsdpConsumer(quitConsumer)
	timeout := time.After(30 * time.Second)
	select {
	case service := <-result:
		t.Log("found available service with ssdp", service)
	case <-timeout:
		quitProvider <- struct{}{}
		quitConsumer <- struct{}{}
		t.Fatal("timeout when searching available service with ssdp")
	}
}

// run this test on device A before running TestStartSsdpConsumer
// device A and B should be on the same LAN
func TestStartSsdpProvider(t *testing.T) {
	quitProvider := make(chan struct{}, 1)
	StartSsdpProvider(quitProvider)
	// 1 minute should be enough to wait for TestStartSsdpConsumer to run, no? increase if needed
	time.Sleep(60 * time.Second)
}

// run this test on device B
func TestStartSsdpConsumer(t *testing.T) {
	quitConsumer := make(chan struct{}, 1)
	result := StartSsdpConsumer(quitConsumer)
	timeout := time.After(10 * time.Second)
	select {
	case service := <-result:
		t.Log("found available service with ssdp", service)
	case <-timeout:
		quitConsumer <- struct{}{}
		t.Fatal("timeout when searching available service with ssdp")
	}
}
