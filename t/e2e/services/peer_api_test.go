package services

import (
	"testing"

	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"

	. "github.com/status-im/status-go/t/utils"
)

func TestPeerAPISuite(t *testing.T) {
	s := new(PeerAPISuite)
	s.upstream = false
	suite.Run(t, s)
}

func TestPeerAPISuiteUpstream(t *testing.T) {
	s := new(PeerAPISuite)
	s.upstream = true
	suite.Run(t, s)
}

type PeerAPISuite struct {
	BaseJSONRPCSuite
	upstream bool
}

func (s *PeerAPISuite) TestAccessiblePeerAPIs() {
	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	err := s.SetupTest(s.upstream, true, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()
	// These peer APIs should be available
	s.AssertAPIMethodExported("peer_discover")
}
