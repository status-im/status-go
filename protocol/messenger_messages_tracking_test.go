package protocol

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/signal"
)

func TestMessengerMessagesTrackingSuite(t *testing.T) {
	suite.Run(t, new(MessengerMessagesTrackingSuite))
}

type EnvelopeSignalHandlerMock struct{}

// EnvelopeSent triggered when envelope delivered atleast to 1 peer.
func (h EnvelopeSignalHandlerMock) EnvelopeSent(identifiers [][]byte) {
	signal.SendEnvelopeSent(identifiers)
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (h EnvelopeSignalHandlerMock) EnvelopeExpired(identifiers [][]byte, err error) {
	signal.SendEnvelopeExpired(identifiers, err)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (h EnvelopeSignalHandlerMock) MailServerRequestCompleted(requestID types.Hash, lastEnvelopeHash types.Hash, cursor []byte, err error) {
	signal.SendMailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (h EnvelopeSignalHandlerMock) MailServerRequestExpired(hash types.Hash) {
	signal.SendMailServerRequestExpired(hash)
}

type EnvelopeEventsInterceptorMock struct {
	EnvelopeEventsInterceptor

	enabled          bool
	lock             sync.Mutex
	identifiersQueue [][][]byte
}

func (i *EnvelopeEventsInterceptorMock) EnvelopeSent(identifiers [][]byte) {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.enabled {
		i.EnvelopeEventsInterceptor.EnvelopeSent(identifiers)
	} else {
		i.identifiersQueue = append(i.identifiersQueue, identifiers)
	}
}

func (i *EnvelopeEventsInterceptorMock) Enable() {
	i.lock.Lock()
	defer i.lock.Unlock()

	for _, identifiers := range i.identifiersQueue {
		i.EnvelopeEventsInterceptor.EnvelopeSent(identifiers)
	}
	i.enabled = true
}

type MessengerMessagesTrackingSuite struct {
	suite.Suite

	bobWaku        types.Waku
	bobInterceptor *EnvelopeEventsInterceptorMock
	bob            *Messenger

	aliceWaku        types.Waku
	aliceInterceptor *EnvelopeEventsInterceptorMock
	alice            *Messenger

	logger *zap.Logger
}

func (s *MessengerMessagesTrackingSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	wakuNodes := CreateWakuV2Network(&s.Suite, s.logger, false, []string{"bob", "alice"})

	s.bobWaku = wakuNodes[0]
	s.bob, s.bobInterceptor = s.newMessenger(s.bobWaku, s.logger.With(zap.String("name", "bob")))

	s.aliceWaku = wakuNodes[1]
	s.alice, s.aliceInterceptor = s.newMessenger(s.aliceWaku, s.logger.With(zap.String("name", "alice")))

	_, err := s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerMessagesTrackingSuite) TearDownTest() {
	if s.bob != nil {
		TearDownMessenger(&s.Suite, s.bob)

	}
	if s.bobWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.bobWaku).Stop())
	}

	if s.alice != nil {
		TearDownMessenger(&s.Suite, s.alice)
	}
	if s.aliceWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.aliceWaku).Stop())
	}

	_ = s.logger.Sync()
}

func (s *MessengerMessagesTrackingSuite) newMessenger(waku types.Waku, logger *zap.Logger) (*Messenger, *EnvelopeEventsInterceptorMock) {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	envelopesMonitorConfig := &transport.EnvelopesMonitorConfig{
		EnvelopeEventsHandler:            EnvelopeSignalHandlerMock{},
		MaxAttempts:                      1,
		AwaitOnlyMailServerConfirmations: false,
		IsMailserver:                     func(peer types.EnodeID) bool { return false },
		Logger:                           s.logger,
	}

	messenger, err := newMessengerWithKey(waku, privateKey, s.logger, []Option{WithEnvelopesMonitorConfig(envelopesMonitorConfig)})
	s.Require().NoError(err)

	interceptor := &EnvelopeEventsInterceptorMock{
		EnvelopeEventsInterceptor: EnvelopeEventsInterceptor{
			EnvelopeEventsHandler: envelopesMonitorConfig.EnvelopeEventsHandler,
			Messenger:             messenger,
		},
	}

	err = messenger.transport.SetEnvelopeEventsHandler(interceptor)
	s.Require().NoError(err)

	return messenger, interceptor
}

func (s *MessengerMessagesTrackingSuite) testMessageMarkedAsSent(textSize int) {
	//when message sent, its sent field should be "false" until we got confirmation
	chat := CreatePublicChat("test-chat", s.bob.getTimesource())
	err := s.bob.SaveChat(chat)
	s.Require().NoError(err)
	inputMessage := buildTestMessage(*chat)
	inputMessage.Text = string(make([]byte, textSize))

	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	rawMessage, err := s.bob.persistence.RawMessageByID(inputMessage.ID)
	s.Require().NoError(err)
	s.Require().False(rawMessage.Sent)

	// enables "EnvelopeSent" callback processing
	s.bobInterceptor.Enable()

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 1 * time.Second
	}

	// Message should be marked as sent eventually
	err = tt.RetryWithBackOff(func() error {
		rawMessage, err = s.bob.persistence.RawMessageByID(inputMessage.ID)
		if err != nil || !rawMessage.Sent {
			return errors.New("message not marked as sent")
		}
		return nil
	}, options)
	s.Require().NoError(err)
}

func (s *MessengerMessagesTrackingSuite) TestMessageMarkedAsSent() {
	s.testMessageMarkedAsSent(1)
}
func (s *MessengerMessagesTrackingSuite) TestSegmentedMessageMarkedAsSent() {
	s.testMessageMarkedAsSent(4 * 1024 * 1024) // 4MB - ensure message is segmented
}
