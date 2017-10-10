package notification

import (
	"testing"

	"errors"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/notification/message"
	t "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestNotificationTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationTestSuite))
}

type NotificationTestSuite struct {
	t.BaseTestSuite
	messagingMock     *common.MockMessagingProvider
	messagingMockCtrl *gomock.Controller
}

func (s *NotificationTestSuite) SetupTest() {
	s.messagingMockCtrl = gomock.NewController(s.T())
	s.messagingMock = common.NewMockMessagingProvider(s.messagingMockCtrl)
}

func (s *NotificationTestSuite) TearDownTest() {
	s.messagingMockCtrl.Finish()
}

func (s *NotificationTestSuite) TestNewNotification() {
	manager := New(nil)
	s.Require().NotNil(manager)
}

func (s *NotificationTestSuite) TestNotifySuccess() {
	token := "test"
	msg := getMessage()

	s.messagingMock.EXPECT().SetMessage([]string{token}, msg.Body).Times(1)
	s.messagingMock.EXPECT().SetPayload(msg.Payload).Times(1)
	s.messagingMock.EXPECT().Send().Return(nil).Times(1)

	manager := New(s.messagingMock)
	res, err := manager.Notify(token, msg)

	s.Require().Equal(token, res)
	s.Require().NoError(err)
}

func (s *NotificationTestSuite) TestNotifyError() {
	token := "test"
	msg := getMessage()
	expectedError := errors.New("error")

	s.messagingMock.EXPECT().SetMessage([]string{token}, msg.Body).Times(1)
	s.messagingMock.EXPECT().SetPayload(msg.Payload).Times(1)
	s.messagingMock.EXPECT().Send().Return(expectedError).Times(1)

	manager := New(s.messagingMock)
	_, err := manager.Notify(token, msg)

	s.Require().Equal(expectedError, err)
}

func getMessage() *message.Message {
	return &message.Message{
		Body: map[string]string{
			"msg": "Hello World1",
			"sum": "Happy Day",
		},
		Payload: &message.Payload{
			Title: "test notification",
		},
	}
}
