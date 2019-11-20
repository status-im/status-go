package status_protocol

import (
	"testing"

	ethnode "github.com/status-im/status-go/status-eth-node"
)

func TestNewMessenger(t *testing.T) {
	m := NewMessenger(&ethnode.Node{})
	if m == nil {
		t.Fatal("NewMessenger result should not be nil")
	}
}
