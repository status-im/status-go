package notification

import (
	"testing"

	"github.com/NaySoftware/go-fcm"
	t "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestFCMClientTestSuite(t *testing.T) {
	suite.Run(t, new(FCMClientTestSuite))
}

type FCMClientTestSuite struct {
	t.BaseTestSuite
}

func (s *FCMClientTestSuite) TestNewFCMClient() {
	fcmClient := NewFCMClient()
	s.Require().NotNil(fcmClient)
	s.Require().IsType(&fcm.FcmClient{}, fcmClient)
}
