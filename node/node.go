package node

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	ethNode "github.com/ethereum/go-ethereum/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
)

// node-related errors
var (
	ErrEthServiceRegistrationFailure     = errors.New("failed to register the Ethereum service")
	ErrWhisperServiceRegistrationFailure = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure       = errors.New("failed to register the LES service")
	ErrNodeMakeFailure                   = errors.New("error creating p2p node")
	ErrNodeRunFailure                    = errors.New("error running p2p node")
	ErrNodeStartFailure                  = errors.New("error starting p2p node")
)

type Node interface {
	Config() *params.NodeConfig
	Node() *ethNode.Node
	Start() (chan struct{}, error)
}

type statusNode struct {
	sync.Mutex
	config  *params.NodeConfig
	node    *ethNode.Node
	started chan struct{}
	stopped chan struct{}
}

func New(config *params.NodeConfig) (Node, error) {
	stackConfig := defaultEmbeddedNodeConfig(config)
	n, err := ethNode.New(stackConfig)

	if err != nil {
		return nil, err
	}

	return &statusNode{
		config: config,
		node:   n,
	}, nil
}

func (sn *statusNode) Start() (chan struct{}, error) {
	sn.Lock()
	if sn.started != nil {
		sn.Unlock()
		return sn.started, nil
	}

	sn.started = make(chan struct{})
	sn.stopped = make(chan struct{})
	sn.Unlock()

	// TODO: make sure data directory exists
	// TODO: make sure keys directory exists
	// TODO: configure required node (should you need to update node's config, e.g. add bootstrap nodes, see node.Config)
	// TODO: Start Ethereum service if we are not expected to use an upstream server.
	// TODO: start Whisper service
	// TODO: return error if node already started
	// TODO: initialise logging
	// TODO: activate MailService required for Offline Inboxing

	go sn.start()

	return sn.started, nil
}

func (sn *statusNode) start() {
	defer HaltOnPanic()

	// start underlying node
	if startErr := sn.node.Start(); startErr != nil {
		close(sn.started)
		sn.Lock()
		sn.started = nil
		sn.Unlock()
		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: fmt.Errorf("%v: %v", ErrNodeStartFailure, startErr).Error(),
			},
		})

		return
	}

	// TODO: init RPC client for this node
	// TODO: PopulateStaticPeers

	// notify all subscribers that Status node is started
	close(sn.started)
	signal.Send(signal.Envelope{
		Type:  signal.EventNodeStarted,
		Event: struct{}{},
	})

	// wait up until underlying node is stopped
	sn.node.Wait()

	// notify sn.Stop() that node has been stopped
	close(sn.stopped)
	log.Info("Node is stopped")
}

func (sn *statusNode) Node() *ethNode.Node {
	return sn.node
}

func (sn *statusNode) Config() *params.NodeConfig {
	return sn.config
}
