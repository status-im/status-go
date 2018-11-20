package fcm

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

func TestFCMClientTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTestSuite))
}

type NotifierTestSuite struct {
	suite.Suite

	fcmClientMock     *MockFirebaseClient
	fcmClientMockCtrl *gomock.Controller
}

func (s *NotifierTestSuite) SetupTest() {
	s.fcmClientMockCtrl = gomock.NewController(s.T())
	s.fcmClientMock = NewMockFirebaseClient(s.fcmClientMockCtrl)
}

func (s *NotifierTestSuite) TearDownTest() {
	s.fcmClientMockCtrl.Finish()
}

func (s *NotifierTestSuite) TestSendSuccess() {
	ids := []string{"1"}
	dataPayload := make(map[string]string)
	dataPayload["from"] = "a"
	dataPayload["to"] = "b"
	dataPayloadByteArray, err := json.Marshal(dataPayload)
	s.Require().NoError(err)
	dataPayloadJSON := string(dataPayloadByteArray)

	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, dataPayload).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, nil).Times(1)
	fcmClient := Notification{s.fcmClientMock}

	err = fcmClient.Send(dataPayloadJSON, ids...)

	s.NoError(err)
}

func (s *NotifierTestSuite) TestSendError() {
	expectedError := errors.New("error")
	ids := []string{"2"}
	dataPayload := make(map[string]string)
	dataPayload["from"] = "c"
	dataPayload["to"] = "d"
	dataPayloadByteArray, err := json.Marshal(dataPayload)
	s.Require().NoError(err)
	dataPayloadJSON := string(dataPayloadByteArray)

	s.fcmClientMock.EXPECT().NewFcmRegIdsMsg(ids, dataPayload).Times(1)
	s.fcmClientMock.EXPECT().Send().Return(nil, expectedError).Times(1)
	fcmClient := Notification{s.fcmClientMock}

	err = fcmClient.Send(dataPayloadJSON, ids...)

	s.Equal(expectedError, err)
}

func (s *NotifierTestSuite) TestSendWithInvalidJSON() {
	ids := []string{"3"}
	dataPayloadJSON := "{a=b}"

	fcmClient := Notification{s.fcmClientMock}

	err := fcmClient.Send(dataPayloadJSON, ids...)
	s.Require().Error(err)

	_, ok := err.(*json.SyntaxError)
	s.True(ok)
}
