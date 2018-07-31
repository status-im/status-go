package peers2

import (
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/stretchr/testify/assert"
)

type discoveryMock struct {
	running bool
}

func (d *discoveryMock) Running() bool                                   { return d.running }
func (d *discoveryMock) Start() error                                    { d.running = true; return nil }
func (d *discoveryMock) Stop() error                                     { d.running = false; return nil }
func (d *discoveryMock) Register(topic string, stop chan struct{}) error { return nil }
func (d *discoveryMock) Discover(_ string, period <-chan time.Duration, _ chan<- *discv5.Node, _ chan<- bool) error {
	for {
		select {
		case _, ok := <-period:
			if !ok {
				return nil
			}
		}
	}
}

func TestTopicPoolBaseStartAndStop(t *testing.T) {
	topicPool := NewTopicPoolBase(&discoveryMock{}, discv5.Topic("test-topic"))
	topicPool.Start(nil)

	assert.NotNil(t, topicPool.quit)
	// use defaults
	assert.NotNil(t, topicPool.period)
	assert.NotNil(t, topicPool.peersHandler)

	topicPool.Stop()
	assert.Nil(t, topicPool.quit)
}

type noNodesAccept struct {
	nodes int
}

func (h *noNodesAccept) Handle(*discv5.Node) bool { h.nodes++; return false }

// TestTopicPoolProperStopSequence tests if the stop process is properly executed
// in a proper order.
func TestTopicPoolProperStopSequence(t *testing.T) {
	handler := &noNodesAccept{}
	topicPool := NewTopicPoolBase(&discoveryMock{}, discv5.Topic("test-topic"), SetPeersHandler(handler))
	topicPool.quit = make(chan struct{})

	var (
		found  chan *discv5.Node
		lookup <-chan bool
	)
	found, lookup, topicPool.discoverDone = topicPool.discover(topicPool.period)
	topicPool.handlerDone = topicPool.handleFoundPeers(nil, found, lookup)

	// spam with found nodes
	go func() {
		for {
			found <- discv5.NewNode(discv5.NodeID{0x01}, net.IPv4(10, 0, 0, 1), 30303, 30303)
		}
	}()

	// finally call Stop()
	time.Sleep(time.Millisecond * 50)
	topicPool.Stop()

	// make sure some found nodes were handled by TopicPool
	assert.NotEqual(t, 0, handler.nodes)
}

func TestTopicPoolWithLimits(t *testing.T) {
	// TODO
}
