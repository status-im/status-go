package node_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestWhisperTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperTestSuite))
}

type WhisperTestSuite struct {
	BaseTestSuite
}

func (s *WhisperTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.Require().NotNil(s.NodeManager)
	s.Require().IsType(&node.NodeManager{}, s.NodeManager)
}

func (s *WhisperTestSuite) TestWhisperFilterRace() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	s.StartTestNode(params.RinkebyNetworkID, false)
	defer s.StopTestNode()

	whisperService, err := s.NodeManager.WhisperService()
	require.NoError(err)
	require.NotNil(whisperService)

	whisperAPI := whisper.NewPublicWhisperAPI(whisperService)
	require.NotNil(whisperAPI)

	accountManager := node.NewAccountManager(s.NodeManager)
	require.NotNil(accountManager)

	// account1
	_, accountKey1, err := accountManager.AddressToDecryptedAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(err)
	accountKey1Hex := common.ToHex(crypto.FromECDSAPub(&accountKey1.PrivateKey.PublicKey))

	_, err = whisperService.AddKeyPair(accountKey1.PrivateKey)
	require.NoError(err)
	ok, err := whisperAPI.HasKeyPair(accountKey1Hex)
	require.NoError(err)
	require.True(ok, "identity not injected")

	// account2
	_, accountKey2, err := accountManager.AddressToDecryptedAccount(TestConfig.Account2.Address, TestConfig.Account2.Password)
	require.NoError(err)
	accountKey2Hex := common.ToHex(crypto.FromECDSAPub(&accountKey2.PrivateKey.PublicKey))
	_, err = whisperService.AddKeyPair(accountKey2.PrivateKey)
	require.NoError(err)
	ok, err = whisperAPI.HasKeyPair(accountKey2Hex)
	require.NoError(err)
	require.True(ok, "identity not injected")

	// race filter addition
	filterAdded := make(chan struct{})
	allFiltersAdded := make(chan struct{})

	go func() {
		counter := 10
		for range filterAdded {
			counter--
			if counter <= 0 {
				break
			}
		}

		close(allFiltersAdded)
	}()

	for i := 0; i < 10; i++ {
		go func() {
			// nolint: errcheck
			whisperAPI.Subscribe(whisper.WhisperFilterArgs{
				Sig: accountKey1Hex,
				Key: accountKey2Hex,
				Topics: [][]byte{
					{0x4e, 0x03, 0x65, 0x7a}, {0x34, 0x60, 0x7c, 0x9b}, {0x21, 0x41, 0x7d, 0xf9},
				},
			})
			filterAdded <- struct{}{}
		}()
	}

	<-allFiltersAdded
}
