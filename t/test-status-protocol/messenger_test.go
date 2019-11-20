package status_protocol

import (
	"testing"

	ethnode "github.com/status-im/status-go/status-eth-node"
	protocol "github.com/status-im/status-go/status-protocol"
)

func TestNewMessenger(t *testing.T) {
	m := protocol.NewMessenger(&ethnode.Node{})
	if m == nil {
		t.Fatal("NewMessenger result should not be nil")
	}
}
