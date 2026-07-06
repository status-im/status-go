package protocol

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/testutils"
)

func TestMessengerMessagesTrackingSuite(t *testing.T) {
	suite.Run(t, new(MessengerMessagesTrackingSuite))
}

type MessageEventsInterceptorMock struct {
	MessageEventsInterceptor

	enabled          bool
	lock             sync.Mutex
	identifiersQueue [][][]byte
}

func (i *MessageEventsInterceptorMock) MessagesSent(identifiers [][]byte) {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.enabled {
		i.MessageEventsInterceptor.MessagesSent(identifiers)
	} else {
		i.identifiersQueue = append(i.identifiersQueue, identifiers)
	}
}

func (i *MessageEventsInterceptorMock) Enable() {
	i.lock.Lock()
	defer i.lock.Unlock()

	for _, identifiers := range i.identifiersQueue {
		i.MessageEventsInterceptor.MessagesSent(identifiers)
	}
	i.enabled = true
}

type MessengerMessagesTrackingSuite struct {
	MessengerBaseTestSuite

	bobInterceptor *MessageEventsInterceptorMock
	bob            *Messenger

	aliceInterceptor *MessageEventsInterceptorMock
	alice            *Messenger
}

func (s *MessengerMessagesTrackingSuite) SetupTest() {
	s.MessengerBaseTestSuite.setupMessaging()
	s.bob, s.bobInterceptor = s.newMessenger()
	s.alice, s.aliceInterceptor = s.newMessenger()
}

func (s *MessengerMessagesTrackingSuite) newMessenger() (*Messenger, *MessageEventsInterceptorMock) {
	messenger, err := newRunningTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{})
	s.Require().NoError(err)

	interceptor := &MessageEventsInterceptorMock{
		MessageEventsInterceptor: MessageEventsInterceptor{
			Messenger: messenger,
		},
	}

	err = messenger.messaging.SetMessageEventsHandler(interceptor)
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

	// enables "MessagesSent" callback processing
	s.bobInterceptor.Enable()

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 1 * time.Second
	}

	// Message should be marked as sent eventually
	err = testutils.RetryWithBackOff(func() error {
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
