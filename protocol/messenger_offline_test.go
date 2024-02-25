package protocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

type MessengerOfflineSuite struct {
	suite.Suite

	owner *Messenger
	bob   *Messenger
	alice *Messenger

	ownerWaku types.Waku
	bobWaku   types.Waku
	aliceWaku types.Waku

	logger *zap.Logger
}

func TestMessengerOfflineSuite(t *testing.T) {
	suite.Run(t, new(MessengerOfflineSuite))
}

func (s *MessengerOfflineSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	wakuNodes := CreateWakuV2Network(&s.Suite, s.logger, false, []string{"owner", "bob", "alice"})

	ownerLogger := s.logger.With(zap.String("name", "owner"))
	s.ownerWaku = wakuNodes[0]
	s.owner = s.newMessenger(s.ownerWaku, ownerLogger)

	bobLogger := s.logger.With(zap.String("name", "bob"))
	s.bobWaku = wakuNodes[1]
	s.bob = s.newMessenger(s.bobWaku, bobLogger)

	aliceLogger := s.logger.With(zap.String("name", "alice"))
	s.aliceWaku = wakuNodes[2]
	s.alice = s.newMessenger(s.aliceWaku, aliceLogger)

	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.owner.communitiesManager.RekeyInterval = 50 * time.Millisecond
}

func (s *MessengerOfflineSuite) TearDownTest() {
	if s.owner != nil {
		s.Require().NoError(s.owner.Shutdown())
	}
	if s.ownerWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.ownerWaku).Stop())
	}

	if s.bob != nil {
		s.Require().NoError(s.bob.Shutdown())
	}
	if s.bobWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.bobWaku).Stop())
	}
	if s.alice != nil {
		s.Require().NoError(s.alice.Shutdown())
	}
	if s.aliceWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.aliceWaku).Stop())
	}
	_ = s.logger.Sync()
}

func (s *MessengerOfflineSuite) newMessenger(waku types.Waku, logger *zap.Logger) *Messenger {
	return newTestCommunitiesMessenger(&s.Suite, waku, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			logger: s.logger,
			extraOptions: []Option{
				WithResendParams(3, 3),
			},
		},
	})
}

func (s *MessengerOfflineSuite) advertiseCommunityTo(community *communities.Community, owner *Messenger, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, owner, user)
}

func (s *MessengerOfflineSuite) joinCommunity(community *communities.Community, owner *Messenger, user *Messenger) {
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, owner, user, request, "")
}

func (s *MessengerOfflineSuite) TestCommunityOfflineEdit() {
	community, chat := createCommunity(&s.Suite, s.owner)

	chatID := chat.ID
	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "some text"

	ctx := context.Background()

	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	_, err := s.alice.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)
	s.checkMessageDelivery(ctx, inputMessage)

	// Simulate going offline
	wakuv2 := gethbridge.GetGethWakuV2From(s.aliceWaku)
	wakuv2.SkipPublishToTopic(true)

	resp, err := s.alice.SendChatMessage(ctx, inputMessage)
	messageID := types.Hex2Bytes(resp.Messages()[0].ID)
	s.Require().NoError(err)

	// Check that message is re-sent once back online
	wakuv2.SkipPublishToTopic(false)
	time.Sleep(5 * time.Second)

	s.checkMessageDelivery(ctx, inputMessage)

	editedText := "some text edited"
	editedMessage := &requests.EditMessage{
		ID:   messageID,
		Text: editedText,
	}

	wakuv2.SkipPublishToTopic(true)
	sendResponse, err := s.alice.EditMessage(ctx, editedMessage)
	s.Require().NotNil(sendResponse)
	s.Require().NoError(err)

	// Check that message is re-sent once back online
	wakuv2.SkipPublishToTopic(false)
	time.Sleep(5 * time.Second)
	inputMessage.Text = editedText

	s.checkMessageDelivery(ctx, inputMessage)
}

func (s *MessengerOfflineSuite) checkMessageDelivery(ctx context.Context, inputMessage *common.Message) {
	var response *MessengerResponse
	// Pull message and make sure org is received
	err := tt.RetryWithBackOff(func() error {
		var err error
		response, err = s.owner.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.messages) == 0 {
			return errors.New("message not received")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(inputMessage.Text, response.Messages()[0].Text)

	// check if response contains the chat we're interested in
	// we use this instead of checking just the length of the chat because
	// a CommunityDescription message might be received in the meantime due to syncing
	// hence response.Chats() might contain the general chat, and the new chat;
	// or only the new chat if the CommunityDescription message has not arrived
	found := false
	for _, chat := range response.Chats() {
		if chat.ID == inputMessage.ChatId {
			found = true
		}
	}
	s.Require().True(found)
}
