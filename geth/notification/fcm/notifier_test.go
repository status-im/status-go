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
	suite.Run(t, new(NotifierTestSuite))
}

type NotifierTestSuite struct {
	t.BaseTestSuite

	fcmClientMock     *MockfirebaseClient
	fcmClientMockCtrl *gomock.Controller
}

func (s *NotifierTestSuite) SetupTest() {
	s.fcmClientMockCtrl = gomock.NewController(s.T())
	s.fcmClientMock = NewMockfirebaseClient(s.fcmClientMockCtrl)
}

func (s *NotifierTestSuite) TearDownTest() {
	s.fcmClientMockCtrl.Finish()
}

func (s *NotifierTestSuite) TestNotifySuccess() {
	fcmPayload := getPayload()
	ids := []string{"1"}
	body := interface{}("body")

	s.fcmClientMock.EXPECT().SetNotificationPayload(fcmPayload).Times(1)
	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, body).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, nil).Times(1)
	fcmClient := Notifier{s.fcmClientMock}

	err := fcmClient.Notify(body, ids...)

	s.NoError(err)
}

func (s *NotifierTestSuite) TestNotifyError() {
	expectedError := errors.New("error")
	fcmPayload := getPayload()
	ids := []string{"1"}
	body := interface{}("body")

	s.fcmClientMock.EXPECT().SetNotificationPayload(fcmPayload).Times(1)
	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, body).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, expectedError).Times(1)
	fcmClient := Notifier{s.fcmClientMock}

	err := fcmClient.Notify(body, ids...)

	s.Equal(expectedError, err)
}

func getPayload() *fcm.NotificationPayload {
	return &fcm.NotificationPayload{Title: "Status - new message", Body: "ping"}
}
