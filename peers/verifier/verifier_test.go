package verifier

import (
	"testing"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/stretchr/testify/require"
)

func TestLocalVerifierForNodeIDTypes(t *testing.T) {
	nodeID := discover.NodeID{1}

	v := NewLocalVerifier([]discover.NodeID{discover.NodeID{1}})
	require.True(t, v.VerifyNode(nil, nodeID))
	require.False(t, v.VerifyNode(nil, discover.NodeID{2}))
}
