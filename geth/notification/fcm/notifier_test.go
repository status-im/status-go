package fcm

import (
	"errors"
	"testing"

	"github.com/NaySoftware/go-fcm"
	"github.com/golang/mock/gomock"
	t "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestFCMClientTestSuite(t *testing.T) {
	suite.Run(t, new(FCMProviderTestSuite))
}

type FCMProviderTestSuite struct {
	t.BaseTestSuite

	fcmClientMock     *MockFirebaseClient
	fcmClientMockCtrl *gomock.Controller
}

func (s *FCMProviderTestSuite) SetupTest() {
	s.fcmClientMockCtrl = gomock.NewController(s.T())
	s.fcmClientMock = NewMockFirebaseClient(s.fcmClientMockCtrl)
}

func (s *FCMProviderTestSuite) TearDownTest() {
	s.fcmClientMockCtrl.Finish()
}

func (s *FCMProviderTestSuite) TestNewFCMClient() {
	fcmClient := Notifier{s.fcmClientMock}

	s.Require().NotNil(fcmClient)
}

func (s *FCMProviderTestSuite) TestNotifySuccess() {
	fcmPayload := getPayload()
	ids := []string{"1"}
	body := interface{}("body")

	s.fcmClientMock.EXPECT().SetNotificationPayload(fcmPayload).Times(1)
	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, body).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, nil).Times(1)
	fcmClient := Notifier{s.fcmClientMock}

	err := fcmClient.Notify(body, ids...)

	s.Require().NoError(err)
}

func (s *FCMProviderTestSuite) TestNotifyError() {
	expectedError := errors.New("error")

	fcmPayload := getPayload()
	ids := []string{"1"}
	body := interface{}("body")

	s.fcmClientMock.EXPECT().SetNotificationPayload(fcmPayload).Times(1)
	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, body).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, expectedError).Times(1)
	fcmClient := Notifier{s.fcmClientMock}

	err := fcmClient.Notify(body, ids...)

	s.Require().Equal(expectedError, err)
}

func getPayload() *fcm.NotificationPayload {
	return &fcm.NotificationPayload{Title: "Status - new message", Body: "ping"}
}
