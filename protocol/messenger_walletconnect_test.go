package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestWalletConnectSessionsSuite(t *testing.T) {
	suite.Run(t, new(WalletConnectSessionsSuite))
}

type WalletConnectSessionsSuite struct {
	MessengerBaseTestSuite
}

func (s *WalletConnectSessionsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	// We start the messenger in order to receive installations
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *WalletConnectSessionsSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *WalletConnectSessionsSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
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
