package fcm

import (
	"errors"
	"testing"

	"github.com/NaySoftware/go-fcm"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

func TestFCMClientTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}

type NotifierTestSuite struct {
	suite.Suite

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
	payload := fcmPayload
	msg := make(map[string]string)
	body := "body"
	msg["msg"] = body

	s.fcmClientMock.EXPECT().SetNotificationPayload(&fcmPayload).Times(1)
	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, msg).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, nil).Times(1)
	fcmClient := Notification{s.fcmClientMock}

	err := fcmClient.Send(body, payload, ids...)

	s.NoError(err)
}

func (s *NotifierTestSuite) TestNotifyError() {
	expectedError := errors.New("error")
	fcmPayload := getPayload()
	ids := []string{"1"}
	payload := fcmPayload
	msg := make(map[string]string)
	body := "body"
	msg["msg"] = body

	s.fcmClientMock.EXPECT().SetNotificationPayload(&fcmPayload).Times(1)
	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, msg).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, expectedError).Times(1)
	fcmClient := Notification{s.fcmClientMock}

	err := fcmClient.Send(body, payload, ids...)

	s.Equal(expectedError, err)
}

func getPayload() fcm.NotificationPayload {
	return fcm.NotificationPayload{Title: "Status - new message", Body: "sum"}
}
