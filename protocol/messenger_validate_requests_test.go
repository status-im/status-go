package protocol

import (
	"testing"

	"github.com/status-im/status-go/protocol/requests"

	"github.com/stretchr/testify/suite"
)

func TestMessengerValidateRequestSuite(t *testing.T) {
	suite.Run(t, new(MessengerValidateRequestSuite))
}

type MessengerValidateRequestSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerValidateRequestSuite) TestSaveNewWakuNodeRequestValidate_Enrtree() {
	r := requests.SaveNewWakuNode{NodeAddress: "enrtree://AIRVQ5DDA4FFWLRBCHJWUWOO6X6S4ZTZ5B667LQ6AJU6PEYDLRD5O@sandbox.waku.nodes.status.im"}
	err := s.m.SaveNewWakuNode(&r)
	s.Require().NoError(err)
}

func (s *MessengerValidateRequestSuite) TestSaveNewWakuNodeRequestValidate_Multiaddr() {
	r := requests.SaveNewWakuNode{NodeAddress: "/ip4/127.0.0.1/tcp/8080"}
	err := s.m.SaveNewWakuNode(&r)
	s.Require().NoError(err)
}

func (s *MessengerValidateRequestSuite) TestSaveNewWakuNodeRequestValidate_MultiaddrFail() {
	r := requests.SaveNewWakuNode{NodeAddress: "/0.0.0.0"}
	err := s.m.SaveNewWakuNode(&r)
	s.Require().Error(err)
}

func (s *MessengerValidateRequestSuite) TestSaveNewWakuNodeRequestValidate_HttpFail() {
	r := requests.SaveNewWakuNode{NodeAddress: "https://google.com"}
	err := s.m.SaveNewWakuNode(&r)
	s.Require().Error(err)
}
