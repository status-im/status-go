package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/connection"
)

func TestShouldPauseCuratedCommunitiesUpdateLoop(t *testing.T) {
	m := &Messenger{}

	require.False(t, m.shouldPauseCuratedCommunitiesUpdateLoop())

	m.setConnectionState(connection.State{Expensive: true})
	require.True(t, m.shouldPauseCuratedCommunitiesUpdateLoop())
}
