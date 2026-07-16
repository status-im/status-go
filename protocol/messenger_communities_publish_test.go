package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/pkg/messaging"
	"github.com/status-im/status-go/pkg/pubsub"
	protocolcommon "github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
)

type publishOrgTimeSource struct{}

func (publishOrgTimeSource) GetCurrentTime() uint64 {
	return 0
}

func TestPublishOrgRejectsNonControlNode(t *testing.T) {
	controlNode, err := crypto.GenerateKey()
	require.NoError(t, err)
	member, err := crypto.GenerateKey()
	require.NoError(t, err)

	community, err := communities.New(communities.Config{
		ControlNode:          &controlNode.PublicKey,
		ID:                   &controlNode.PublicKey,
		MemberIdentity:       member,
		CommunityDescription: &protobuf.CommunityDescription{},
	}, publishOrgTimeSource{}, &communities.NoopDescriptionEncryptor{}, nil)
	require.NoError(t, err)
	require.False(t, community.IsControlNode())

	require.ErrorIs(t, (&Messenger{}).publishOrg(community, false), communities.ErrNotControlNode)
}

func TestPublishOrgAllowsControlNode(t *testing.T) {
	messagingEnv, err := messaging.NewTestMessagingEnvironment()
	require.NoError(t, err)
	require.NoError(t, messagingEnv.Setup(t))

	m, err := newTestMessenger(t, messagingEnv, testMessengerConfig{})
	require.NoError(t, err)

	controlNode, err := crypto.GenerateKey()
	require.NoError(t, err)
	community, err := communities.New(communities.Config{
		PrivateKey:           controlNode,
		ControlNode:          &controlNode.PublicKey,
		ControlDevice:        true,
		ID:                   &controlNode.PublicKey,
		MemberIdentity:       controlNode,
		CommunityDescription: &protobuf.CommunityDescription{},
	}, publishOrgTimeSource{}, &communities.NoopDescriptionEncryptor{}, nil)
	require.NoError(t, err)
	require.True(t, community.IsControlNode())

	messageEvents, unsubscribe := pubsub.Subscribe[protocolcommon.MessageEvent](m.sender.Publisher(), 1)
	defer unsubscribe()

	require.NoError(t, m.publishOrg(community, false))

	select {
	case event := <-messageEvents:
		require.NotNil(t, event.ScheduledMessage)
		require.Nil(t, event.ScheduledMessage.Recipient)
		rawMessage := event.ScheduledMessage.RawMessage
		require.NotNil(t, rawMessage)
		require.NotEmpty(t, rawMessage.ID)
		require.Equal(t, protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION, rawMessage.MessageType)
		require.Equal(t, community.IDString(), rawMessage.ContentTopic)
		require.Equal(t, []byte(community.ID()), rawMessage.CommunityID)
		require.NotNil(t, rawMessage.Sender)
		require.True(t, crypto.IsPubKeyEqual(&rawMessage.Sender.PublicKey, community.ControlNode()))
	default:
		require.Fail(t, "community description was not scheduled for publication")
	}
}
