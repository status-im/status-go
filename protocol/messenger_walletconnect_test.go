package protocol

import (
	"crypto/ecdsa"
	"fmt"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"testing"

	"github.com/status-im/status-go/eth-node/types"
)

func TestWalletConnectSessionsSuite(t *testing.T) {
	suite.Run(t, new(WalletConnectSessionsSuite))
}

type WalletConnectSessionsSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
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

func (s *WalletConnectSessionsSuite) TestMotherShip() {
	peerId := "0643983b-0000-2222-1111-b05fdac338zd"
	connectorInfo := "{:connected true,:accounts #js [0x3Ed3ab4A64C7D412bF628aDe9722c910ab20cE86], :chainId 1, :bridge https://c.bridge.walletconnect.org, :key c4ae6c97875ab90e64678f8fbeaeff5e38408f0d6ea3f58628556bc25bcc5092, :clientId 0643983b-0000-2222-1111-b05fdac338zd, :clientMeta #js {:name Status Wallet, :description Status is a secure messaging app, crypto wallet, and Web3 browser built with state of the art technology., :url #, :icons #js [https://statusnetwork.com/img/press-kit-status-logo.svg]}, :peerId 0643983b-0000-2222-1111-b05fdac338zd, :peerMeta #js {:name 1inch dApp, :description DeFi / DEX aggregator with the most liquidity and the best rates on Ethereum, Binance Smart Chain, Optimism, Polygon, 1inch dApp is an entry point to the 1inch Network's tech., :url https://app.1inch.io, :icons #js [https://app.1inch.io/assets/images/1inch_logo_without_text.svg https://app.1inch.io/assets/images/logo.png]}, :handshakeId 1657776235200377, :handshakeTopic 0643983b-0000-2222-1111-b05fdac338zd}"

	_, err := s.m.addWalletConnectSession(peerId, connectorInfo)
	s.Require().NoError(err)

	response, err := s.m.getWalletConnectSession()
	fmt.Println(response)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Equal(response.PeerId, peerId)
	s.Require().Equal(response.ConnectorInfo, connectorInfo)

}
