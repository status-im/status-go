package notification

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/common"
	t "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestNotificationTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationTestSuite))
}

type NotificationTestSuite struct {
	t.BaseTestSuite
	messagingMock     *common.MockMessaging
	messagingMockCtrl *gomock.Controller
}

func (s *NotificationTestSuite) SetupTest() {
	s.messagingMockCtrl = gomock.NewController(s.T())
	s.messagingMock = common.NewMockMessaging(s.messagingMockCtrl)
}

func (s *NotificationTestSuite) TearDownTest() {
	s.messagingMockCtrl.Finish()
}

func (s *NotificationTestSuite) TestNewNotification() {
	manager := New(nil)
	s.Require().NotNil(manager)
}

func (s *NotificationTestSuite) TestNotify() {
	token := "test"
	s.messagingMock.EXPECT().NewFcmRegIdsMsg([]string{token}, map[string]string{
		"msg": "Hello World1",
		"sum": "Happy Day",
	})

	manager := New(s.messagingMock)
	res := manager.Notify(token)

	s.Require().Equal(token, res)
}

func (s *NotificationTestSuite) TestSend() {
	s.messagingMock.EXPECT().Send().Times(1).Return(nil)

	manager := New(s.messagingMock)
	err := manager.Send()

	s.Require().NoError(err)
}
