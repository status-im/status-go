package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/peersyncing"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerPeersyncingSuite(t *testing.T) {
	suite.Run(t, new(MessengerPeersyncingSuite))
}

type MessengerPeersyncingSuite struct {
	suite.Suite
	owner *Messenger
	bob   *Messenger
	alice *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerPeersyncingSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	peerSyncingLoopInterval = 500 * time.Millisecond

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.owner = s.newMessenger()
	s.bob = s.newMessenger()
	s.alice = s.newMessenger()

	s.alice.featureFlags.ResendRawMessagesDisabled = true
	s.bob.featureFlags.ResendRawMessagesDisabled = true
	s.owner.featureFlags.ResendRawMessagesDisabled = true

	s.owner.communitiesManager.RekeyInterval = 50 * time.Millisecond

	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerPeersyncingSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.owner)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.alice)
	_ = s.logger.Sync()
}

func (s *MessengerPeersyncingSuite) newMessenger() *Messenger {
	return newTestCommunitiesMessenger(&s.Suite, s.shh, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			logger: s.logger,
		},
	})
}

func (s *MessengerPeersyncingSuite) joinCommunity(community *communities.Community, owner *Messenger, user *Messenger) {
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, owner, user, request, "")
}

func (s *MessengerPeersyncingSuite) thirdPartyTest(community *communities.Community, chat *Chat) {
	// We disable resending to make sure that the message is not re-transmitted
	s.alice.featureFlags.Peersyncing = false
	s.owner.featureFlags.Peersyncing = true
	s.bob.featureFlags.Peersyncing = true
	s.owner.communitiesManager.PermissionChecker = &testPermissionChecker{}

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)

	s.joinCommunity(community, s.owner, s.alice)

	chatID := chat.ID
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"
	ctx := context.Background()

	if community.Encrypted() {

		_, err := WaitOnMessengerResponse(
			s.alice,
			func(r *MessengerResponse) bool {
				keys, err := s.alice.encryptor.GetKeysForGroup([]byte(chat.ID))
				return err == nil && len(keys) > 0
			},
			"keys not received",
		)
		s.Require().NoError(err)
	}

	// Send message, it should be received
	response, err := s.alice.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	messageID := response.Messages()[0].ID

	// Make sure the message makes it to the owner
	response, err = WaitOnMessengerResponse(
		s.owner,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.Messages()) == 1 && r.Messages()[0].ID == messageID
		},
		"message not received",
	)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	msg, err := s.owner.peersyncing.AvailableMessages()
	s.Require().NoError(err)
	s.Require().Len(msg, 1)

	// Bob joins the community
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)

	s.joinCommunity(community, s.owner, s.bob)

	// Bob should now send an offer
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return s.bob.peersyncingOffers[messageID[2:]] != 0
		},
		"offer not sent",
	)
	s.Require().NoError(err)

	// Owner should now reply to the offer
	_, err = WaitOnMessengerResponse(
		s.owner,
		func(r *MessengerResponse) bool {
			return s.owner.peersyncingRequests[s.bob.myHexIdentity()+messageID[2:]] != 0
		},
		"request not sent",
	)
	s.Require().NoError(err)

	// Bob should receive the message
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Messages()) == 1 && r.Messages()[0].ID == messageID
		},
		"message not received",
	)
	s.Require().NoError(err)

}

// Owner creates a community
// Owner sends a message
// Alice joins
// Alice receives the message
func (s *MessengerPeersyncingSuite) TestSyncWithPeerCommunitySender() {

	s.alice.featureFlags.Peersyncing = true
	s.owner.featureFlags.Peersyncing = true

	// create community and make alice join it
	community, chat := createCommunity(&s.Suite, s.owner)

	chatID := chat.ID
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"

	// Send message, it should be received
	response, err := s.owner.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	messageID := response.Messages()[0].ID

	msg, err := s.owner.peersyncing.AvailableMessages()
	s.Require().NoError(err)
	s.Require().Len(msg, 1)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	// Alice should now receive the message
	_, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			_, err := s.owner.RetrieveAll()
			if err != nil {
				return false
			}
			return len(r.Messages()) == 1 && r.Messages()[0].ID == messageID
		},
		"message not received",
	)
	s.Require().NoError(err)
}

// Owner creates a community
// Alice joins
// Alice sends a message
// Owner receives the message
// Bob joins the community
// They should retrieve the message from the owner

func (s *MessengerPeersyncingSuite) TestSyncWithPeerCommunityThirdPartyEncrypted() {
	community, chat := createEncryptedCommunity(&s.Suite, s.owner)
	s.thirdPartyTest(community, chat)
}

func (s *MessengerPeersyncingSuite) TestSyncWithPeerCommunityThirdPartyNotEncrypted() {
	community, chat := createCommunity(&s.Suite, s.owner)
	s.thirdPartyTest(community, chat)
}

func (s *MessengerPeersyncingSuite) TestCanSyncMessageWith() {
	community, chat := createCommunity(&s.Suite, s.owner)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	syncMessage := peersyncing.SyncMessage{
		ID:        []byte("test-id"),
		GroupID:   []byte(chat.ID),
		Type:      peersyncing.SyncMessageCommunityType,
		Payload:   []byte("some-payload"),
		Timestamp: 1,
	}
	s.Require().NoError(s.owner.peersyncing.Add(syncMessage))

	community, err := s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	canSyncWithBob, err := s.owner.canSyncCommunityMessageWith(chat, community, &s.bob.identity.PublicKey)
	s.Require().NoError(err)
	s.Require().False(canSyncWithBob)

	canSyncWithAlice, err := s.owner.canSyncCommunityMessageWith(chat, community, &s.alice.identity.PublicKey)
	s.Require().NoError(err)
	s.Require().True(canSyncWithAlice)
}
