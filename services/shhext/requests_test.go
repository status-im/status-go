package shhext

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

func TestRegisterSameRequests(t *testing.T) {
	registry := NewRequestsRegistry(10 * time.Second)
	topics := []whisper.TopicType{{1}}
	require.NoError(t, registry.Register(common.Hash{1}, topics))
	require.Error(t, registry.Register(common.Hash{2}, topics))
}

func TestRegisterSameRequestsWithoutDelay(t *testing.T) {
	registry := NewRequestsRegistry(0)
	topics := []whisper.TopicType{{1}}
	require.NoError(t, registry.Register(common.Hash{1}, topics))
	require.NoError(t, registry.Register(common.Hash{2}, topics))
}

func TestRegisterDifferentRequests(t *testing.T) {
	registry := NewRequestsRegistry(10 * time.Second)
	require.NoError(t, registry.Register(common.Hash{1}, []whisper.TopicType{{1}}))
	require.NoError(t, registry.Register(common.Hash{2}, []whisper.TopicType{{2}}))
}

func TestUnregisterReplacedRequest(t *testing.T) {
	registry := NewRequestsRegistry(0)
	unreg := common.Hash{1}
	topics := []whisper.TopicType{{1}}
	require.NoError(t, registry.Register(unreg, topics))
	replacement := common.Hash{2}
	require.NoError(t, registry.Register(replacement, topics))
	// record should be replaced with common.Hash{2}, so when we will remove unreg it will not affect topics map
	registry.Unregister(unreg)
	record, exist := registry.uidToTopics[replacement]
	require.True(t, exist, "replaced record should exist")
	require.Equal(t, replacement, registry.byTopicsHash[record].lastUID)
}
