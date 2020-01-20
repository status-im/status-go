package ext

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
)

func TestRegisterSameRequests(t *testing.T) {
	registry := NewRequestsRegistry(10 * time.Second)
	topics := []types.TopicType{{1}}
	require.NoError(t, registry.Register(types.Hash{1}, topics))
	require.Error(t, registry.Register(types.Hash{2}, topics))
}

func TestRegisterSameRequestsWithoutDelay(t *testing.T) {
	registry := NewRequestsRegistry(0)
	topics := []types.TopicType{{1}}
	require.NoError(t, registry.Register(types.Hash{1}, topics))
	require.NoError(t, registry.Register(types.Hash{2}, topics))
}

func TestRegisterDifferentRequests(t *testing.T) {
	registry := NewRequestsRegistry(10 * time.Second)
	require.NoError(t, registry.Register(types.Hash{1}, []types.TopicType{{1}}))
	require.NoError(t, registry.Register(types.Hash{2}, []types.TopicType{{2}}))
}

func TestUnregisterReplacedRequest(t *testing.T) {
	registry := NewRequestsRegistry(0)
	unreg := types.Hash{1}
	topics := []types.TopicType{{1}}
	require.NoError(t, registry.Register(unreg, topics))
	replacement := types.Hash{2}
	require.NoError(t, registry.Register(replacement, topics))
	// record should be replaced with types.Hash{2}, so when we will remove unreg it will not affect topics map
	registry.Unregister(unreg)
	record, exist := registry.uidToTopics[replacement]
	require.True(t, exist, "replaced record should exist")
	require.Equal(t, replacement, registry.byTopicsHash[record].lastUID)
}
