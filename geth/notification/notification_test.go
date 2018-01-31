package notification

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

func TestNotificationClientTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}

type NotifierTestSuite struct {
	suite.Suite

	fcmClientMock     *MockClient
	fcmClientMockCtrl *gomock.Controller
}

func (s *NotifierTestSuite) SetupTest() {
	s.fcmClientMockCtrl = gomock.NewController(s.T())
	s.fcmClientMock = NewMockClient(s.fcmClientMockCtrl)
}

func (s *NotifierTestSuite) TearDownTest() {
	s.fcmClientMockCtrl.Finish()
}

func (s *NotifierTestSuite) TestNotifySuccess() {
	payload := getPayload()
	ids := []string{"1"}
	msg := make(map[string]string)
	body := "body"
	msg["msg"] = body

	s.fcmClientMock.EXPECT().SetNotificationPayload(payload).Times(1)
	s.fcmClientMock.EXPECT().NewRegIdsMsg(ids, msg).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, nil).Times(1)
	fcmClient := Notification{s.fcmClientMock}

	err := fcmClient.Send(body, payload, ids...)

	s.NoError(err)
}

func (s *NotifierTestSuite) TestNotifyError() {
	expectedError := errors.New("error")
	payload := getPayload()
	ids := []string{"1"}
	msg := make(map[string]string)
	body := "body"
	msg["msg"] = body

	s.fcmClientMock.EXPECT().SetNotificationPayload(&payload).Times(1)
	s.fcmClientMock.EXPECT().NewRegIdsMsg(ids, msg).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, expectedError).Times(1)
	fcmClient := Notification{s.fcmClientMock}

	err := fcmClient.Send(body, payload, ids...)

	s.Equal(expectedError, err)
}

func getPayload() Payload {
	return Payload{Title: "Status - new message", Body: "sum"}
}
