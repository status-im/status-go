package notification

import (
	"testing"

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
}