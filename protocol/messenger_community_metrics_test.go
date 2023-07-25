package protocol

import (
	"testing"

	"github.com/status-im/status-go/protocol/requests"
	"github.com/stretchr/testify/suite"
)

func TestMessengerCommunityMetricsSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunityMetricsSuite))
}

type MessengerCommunityMetricsSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMessageMetrics() {
	request := &requests.CommunityMetricsRequest{
		CommunityID:    []byte("0x654321"),
		Type:           requests.CommunityMetricsRequestMessages,
		StartTimestamp: 1690279200,
		EndTimestamp:   1690282800, // one hour
		MaxCount:       10,
	}
	// Send contact request
	resp, err := s.m.CollectCommunityMetrics(request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}
