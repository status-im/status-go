package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
)

func TestShouldPauseCuratedCommunitiesUpdateLoop(t *testing.T) {
	m := &Messenger{}

	require.False(t, m.shouldPauseCuratedCommunitiesUpdateLoop())

	m.setConnectionState(connection.State{Expensive: true})
	require.True(t, m.shouldPauseCuratedCommunitiesUpdateLoop())
}

func TestSubscribeToCuratedCommunityDescriptions(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	m, err := newTestMessenger(t, messagingEnv, testMessengerConfig{})
	require.NoError(t, err)

	newCommunityKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	newCommunityID := cryptotypes.HexBytes(crypto.CompressPubkey(&newCommunityKey.PublicKey)).String()

	existingCommunityKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	existingCommunityID := cryptotypes.HexBytes(crypto.CompressPubkey(&existingCommunityKey.PublicKey)).String()
	existingFilters, err := m.messaging.InitPublicChats(types2.ChatsToInitialize{{
		ChatID:      existingCommunityID,
		PubsubTopic: types2.DefaultShard().PubsubTopic(),
	}})
	require.NoError(t, err)
	require.Len(t, existingFilters, 1)
	require.False(t, existingFilters[0].IsEphemeral())

	require.NoError(t, m.subscribeToCuratedCommunityDescriptions([]string{newCommunityID, existingCommunityID}))

	newFilter := m.messaging.ChatFilterByChatID(newCommunityID)
	require.NotNil(t, newFilter)
	require.True(t, newFilter.IsEphemeral())
	require.False(t, m.messaging.ChatFilterByChatID(existingCommunityID).IsEphemeral())
}
