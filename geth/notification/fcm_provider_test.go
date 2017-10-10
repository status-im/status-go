package notification

import (
	"errors"
	"testing"

	"github.com/NaySoftware/go-fcm"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/notification/message"
	t "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestFCMClientTestSuite(t *testing.T) {
	suite.Run(t, new(FCMProviderTestSuite))
}

type FCMProviderTestSuite struct {
	t.BaseTestSuite

	fcmClientMock     *common.MockFirebaseClient
	fcmClientMockCtrl *gomock.Controller
}

func (s *FCMProviderTestSuite) SetupTest() {
	s.fcmClientMockCtrl = gomock.NewController(s.T())
	s.fcmClientMock = common.NewMockFirebaseClient(s.fcmClientMockCtrl)
}

func (s *FCMProviderTestSuite) TearDownTest() {
	s.fcmClientMockCtrl.Finish()
}

func (s *FCMProviderTestSuite) TestNewFCMClient() {
	fcmClient := NewFCMProvider(s.fcmClientMock)

	s.Require().NotNil(fcmClient)
}

func (s *FCMProviderTestSuite) TestSetMessage() {
	ids := []string{"1"}
	body := interface{}("body")

	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, body).Times(1)
	fcmClient := NewFCMProvider(s.fcmClientMock)

	fcmClient.SetMessage(ids, body)
}

func (s *FCMProviderTestSuite) TestSetPayload() {
	title := "title"
	body := "body"
	payload := &message.Payload{Title: title, Body: body}
	fcmPayload := &fcm.NotificationPayload{Title: title, Body: body}

	s.fcmClientMock.EXPECT().SetNotificationPayload(fcmPayload).Times(1)
	fcmClient := NewFCMProvider(s.fcmClientMock)

	fcmClient.SetPayload(payload)
}

func (s *FCMProviderTestSuite) TestSendSuccess() {
	s.fcmClientMock.EXPECT().Send().Return(nil, nil).Times(1)
	fcmClient := NewFCMProvider(s.fcmClientMock)

	err := fcmClient.Send()

	s.Require().NoError(err)
}

func (s *FCMProviderTestSuite) TestSendError() {
	expectedError := errors.New("error")

	s.fcmClientMock.EXPECT().Send().Return(nil, expectedError).Times(1)
	fcmClient := NewFCMProvider(s.fcmClientMock)

	err := fcmClient.Send()

	s.Require().Equal(expectedError, err)
}
