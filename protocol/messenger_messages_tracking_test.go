package protocol

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
	messagingtypes "github.com/status-im/status-go/messaging/types"
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
	MessengerBaseTestSuite

	bobInterceptor *EnvelopeEventsInterceptorMock
	bob            *Messenger

	aliceInterceptor *EnvelopeEventsInterceptorMock
	alice            *Messenger
}

func (s *MessengerMessagesTrackingSuite) SetupTest() {
	s.MessengerBaseTestSuite.setupWaku()
	s.bob, s.bobInterceptor = s.newMessenger()
	s.alice, s.aliceInterceptor = s.newMessenger()
}

func (s *MessengerMessagesTrackingSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.bob)
	s.MessengerBaseTestSuite.TearDownTest()
}

func (s *MessengerMessagesTrackingSuite) newMessenger() (*Messenger, *EnvelopeEventsInterceptorMock) {
	envelopeEventsConfig := &messagingtypes.EnvelopeEventsConfig{
		EnvelopeEventsHandler:      EnvelopeSignalHandlerMock{},
		MaxMessageDeliveryAttempts: 1,
		MailServerConfirmations:    false,
	}

	messenger, err := newRunningTestMessenger(s.shh, testMessengerConfig{extraOptions: []Option{WithEnvelopeEventsConfig(envelopeEventsConfig)}})
	s.Require().NoError(err)

	interceptor := &EnvelopeEventsInterceptorMock{
		EnvelopeEventsInterceptor: EnvelopeEventsInterceptor{
			EnvelopeEventsHandler: envelopeEventsConfig.EnvelopeEventsHandler,
			Messenger:             messenger,
		},
	}

	err = messenger.messaging.SetEnvelopeEventsHandler(interceptor)
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
		if err != nil || rawMessage.SendCount < 1 {
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
