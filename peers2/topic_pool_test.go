package peers2

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/stretchr/testify/assert"
)

type discoveryMock struct {
	sync.Mutex

	running bool
	period  time.Duration
}

func (d *discoveryMock) Running() bool {
	d.Lock()
	defer d.Unlock()
	return d.running
}
func (d *discoveryMock) Start() error {
	d.Lock()
	d.running = true
	d.Unlock()
	return nil
}
func (d *discoveryMock) Stop() error {
	d.Lock()
	d.running = false
	d.Unlock()
	return nil
}
func (d *discoveryMock) Register(topic string, stop chan struct{}) error { return nil }
func (d *discoveryMock) Discover(_ string, period <-chan time.Duration, _ chan<- *discv5.Node, _ chan<- bool) error {
	for {
		p, ok := <-period
		if !ok {
			return nil
		}
		d.Lock()
		d.period = p
		d.Unlock()
	}
}

func TestTopicPoolBaseStartAndStop(t *testing.T) {
	topicPool := NewTopicPoolBase(&discoveryMock{}, discv5.Topic("test-topic"))
	assert.NotNil(t, topicPool.peersHandler)

	topicPool.Start(nil)
	assert.NotNil(t, topicPool.quit)
	assert.NotNil(t, topicPool.period)

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
	topicPool.period = defaultFastSlowDiscoverPeriod()
	found, lookup, topicPool.discoverDone = topicPool.discover(topicPool.period.channel())
	topicPool.handlerDone = topicPool.handleFoundPeers(nil, found, lookup)

	// spam with found nodes
	go func() {
		found <- discv5.NewNode(discv5.NodeID{0x01}, net.IPv4(10, 0, 0, 1), 30303, 30303)
	}()

	// finally call Stop()
	time.Sleep(time.Millisecond * 50)
	topicPool.Stop()

	// make sure some found nodes were handled by TopicPool
	assert.NotEqual(t, 0, handler.nodes)
}

func TestTopicPoolWithLimits(t *testing.T) {
	var err error

	topicPool := NewTopicPoolWithLimits(NewTopicPoolBase(&discoveryMock{}, discv5.Topic("test-topic")), 1, 2)
	period := topicPool.period.channel()
	topicPool.Start(nil)
	defer topicPool.Stop()

	// use fast sync by default
	assert.Equal(t, defaultFastSync, <-period)

	// add a peer
	err = topicPool.ConfirmAdded(discover.NodeID{0x01})
	assert.NoError(t, err)
	assert.Len(t, topicPool.connectedPeers, 1)
	assert.Equal(t, defaultSlowSync, <-period)
	assert.False(t, topicPool.Satisfied())

	// add the same peer
	err = topicPool.ConfirmAdded(discover.NodeID{0x01})
	assert.NoError(t, err)
	assert.Len(t, topicPool.connectedPeers, 1)
	assert.False(t, topicPool.Satisfied())

	// add a new peer
	err = topicPool.ConfirmAdded(discover.NodeID{0x02})
	assert.NoError(t, err)
	assert.Len(t, topicPool.connectedPeers, 2)
	assert.True(t, topicPool.Satisfied())

	// remove the last peer
	err = topicPool.ConfirmDropped(discover.NodeID{0x02})
	assert.NoError(t, err)
	assert.Len(t, topicPool.connectedPeers, 1)
	assert.False(t, topicPool.Satisfied())
}
