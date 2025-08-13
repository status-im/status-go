package protocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

const minimumResendDelay = 500 * time.Millisecond
const waitForResentDelay = minimumResendDelay + 100*time.Millisecond

type MessengerOfflineSuite struct {
	MessengerBaseTestSuite

	owner *Messenger
	bob   *Messenger
	alice *Messenger

	mockedBalances          communities.BalancesByChain
	collectiblesManagerMock *CollectiblesManagerMock
	collectiblesServiceMock *CollectiblesServiceMock
	accountsTestData        map[string][]string
	accountsPasswords       map[string]string
}

func TestMessengerOfflineSuite(t *testing.T) {
	suite.Run(t, new(MessengerOfflineSuite))
}

func (s *MessengerOfflineSuite) SetupTest() {
	s.MessengerBaseTestSuite.setupMessaging()

	s.collectiblesServiceMock = &CollectiblesServiceMock{}
	s.collectiblesManagerMock = &CollectiblesManagerMock{}
	s.accountsTestData = make(map[string][]string)
	s.accountsPasswords = make(map[string]string)

	s.owner = s.newMessenger("", []string{})
	s.bob = s.newMessenger(bobPassword, []string{bobAccountAddress})
	s.alice = s.newMessenger(alicePassword, []string{aliceAddress1})

	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.owner.communitiesManager.RekeyInterval = 50 * time.Millisecond
}

func (s *MessengerOfflineSuite) TearDownTest() {
	s.Require().NoError(s.owner.Shutdown())
	s.Require().NoError(s.bob.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	s.MessengerBaseTestSuite.TearDownTest()
}

func (s *MessengerOfflineSuite) newMessenger(password string, accounts []string) *Messenger {
	return newTestCommunitiesMessenger(&s.Suite, s.messagingEnv, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			extraOptions: []Option{
				WithResendParams(minimumResendDelay, 1),
			},
		},
		walletAddresses:     accounts,
		password:            password,
		mockedBalances:      &s.mockedBalances,
		collectiblesManager: s.collectiblesManagerMock,
	})
}

func (s *MessengerOfflineSuite) advertiseCommunityTo(community *communities.Community, owner *Messenger, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, owner, user)
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
	joinCommunity(&s.Suite, community.ID(), s.owner, s.alice, aliceAccountAddress, []string{aliceAddress1})

	_, err := s.alice.SendChatMessage(ctx, inputMessage)
	s.Require().NoError(err)
	s.checkMessageDelivery(ctx, inputMessage)

	// Simulate going offline
	getOnline := s.messagingEnv.SimulateOffline()

	resp, err := s.alice.SendChatMessage(ctx, inputMessage)
	messageID := types.Hex2Bytes(resp.Messages()[0].ID)
	s.Require().NoError(err)

	// Check that message is re-sent once back online
	getOnline()

	s.checkMessageDelivery(ctx, inputMessage)

	editedText := "some text edited"
	editedMessage := &requests.EditMessage{
		ID:   messageID,
		Text: editedText,
	}

	// Simulate going offline
	getOnline = s.messagingEnv.SimulateOffline()
	sendResponse, err := s.alice.EditMessage(ctx, editedMessage)
	s.Require().NotNil(sendResponse)
	s.Require().NoError(err)

	// Check that message is re-sent once back online
	getOnline()
	time.Sleep(waitForResentDelay)
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
