package verifier

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/stretchr/testify/require"
)

func TestLocalVerifierForNodeIDTypes(t *testing.T) {
	nodeID := discover.NodeID{1}

	v := NewLocalVerifier([]discover.NodeID{{1}})
	require.True(t, v.VerifyNode(context.TODO(), nodeID))
	require.False(t, v.VerifyNode(context.TODO(), discover.NodeID{2}))
}
