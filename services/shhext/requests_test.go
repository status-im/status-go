package shhext

import (
	"testing"
	"time"

	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
	"github.com/stretchr/testify/require"
)

func TestRegisterSameRequests(t *testing.T) {
	registry := NewRequestsRegistry(10 * time.Second)
	topics := []whispertypes.TopicType{{1}}
	require.NoError(t, registry.Register(protocol.Hash{1}, topics))
	require.Error(t, registry.Register(protocol.Hash{2}, topics))
}

func TestRegisterSameRequestsWithoutDelay(t *testing.T) {
	registry := NewRequestsRegistry(0)
	topics := []whispertypes.TopicType{{1}}
	require.NoError(t, registry.Register(protocol.Hash{1}, topics))
	require.NoError(t, registry.Register(protocol.Hash{2}, topics))
}

func TestRegisterDifferentRequests(t *testing.T) {
	registry := NewRequestsRegistry(10 * time.Second)
	require.NoError(t, registry.Register(protocol.Hash{1}, []whispertypes.TopicType{{1}}))
	require.NoError(t, registry.Register(protocol.Hash{2}, []whispertypes.TopicType{{2}}))
}

func TestUnregisterReplacedRequest(t *testing.T) {
	registry := NewRequestsRegistry(0)
	unreg := protocol.Hash{1}
	topics := []whispertypes.TopicType{{1}}
	require.NoError(t, registry.Register(unreg, topics))
	replacement := protocol.Hash{2}
	require.NoError(t, registry.Register(replacement, topics))
	// record should be replaced with protocol.Hash{2}, so when we will remove unreg it will not affect topics map
	registry.Unregister(unreg)
	record, exist := registry.uidToTopics[replacement]
	require.True(t, exist, "replaced record should exist")
	require.Equal(t, replacement, registry.byTopicsHash[record].lastUID)
}
