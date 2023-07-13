package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/requests"
)

func TestWalletConnectSessionsSuite(t *testing.T) {
	suite.Run(t, new(WalletConnectSessionsSuite))
}

type WalletConnectSessionsSuite struct {
	MessengerBaseTestSuite
}

func (s *WalletConnectSessionsSuite) TestCreateReadAndDeleteSessions() {
	peerID1 := "0643983b-0000-2222-1111-b05fdac338zd1"
	peerID2 := "0643983b-0000-2222-1111-b05fdac338zd2"

	dappName1 := "b"
	dappName2 := "a"
	url := "some-url"

	sessionInfo := "some dummy text that looks like a json"

	wcSession1 := &requests.AddWalletConnectSession{
		PeerID:   peerID1,
		DAppName: dappName1,
		DAppURL:  url,
		Info:     sessionInfo,
	}

	wcSession2 := &requests.AddWalletConnectSession{
		PeerID:   peerID2,
		DAppName: dappName2,
		DAppURL:  url,
		Info:     sessionInfo,
	}

	err := s.m.AddWalletConnectSession(wcSession1)
	s.Require().NoError(err)

	err = s.m.AddWalletConnectSession(wcSession2)
	s.Require().NoError(err)

	response, err := s.m.GetWalletConnectSession()
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response, 2)
	s.Require().Equal(response[0].PeerID, peerID2)
	s.Require().Equal(response[0].DAppName, dappName2)
	s.Require().Equal(response[0].DAppURL, url)
	s.Require().Equal(response[0].Info, sessionInfo)
	s.Require().Equal(response[1].PeerID, peerID1)
	s.Require().Equal(response[1].DAppName, dappName1)
	s.Require().Equal(response[1].DAppURL, url)
	s.Require().Equal(response[1].Info, sessionInfo)

	errWhileDeletion := s.m.DestroyWalletConnectSession(peerID1)
	s.Require().NoError(errWhileDeletion)

	shouldNotBeEmpty, errWhileFetchingAgain := s.m.GetWalletConnectSession()
	s.Require().NoError(errWhileFetchingAgain)
	s.Require().NotNil(shouldNotBeEmpty)
	s.Require().Len(shouldNotBeEmpty, 1)
	s.Require().Equal(shouldNotBeEmpty[0].PeerID, peerID2)
}
