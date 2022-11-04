package protocol

import (
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMakeQRCodeFromURLSuite(t *testing.T) {
	suite.Run(t, new(MakeQRCodeFromURLSuite))
}

type MakeQRCodeFromURLSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MakeQRCodeFromURLSuite) SetupTest() {
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

func (s *MakeQRCodeFromURLSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MakeQRCodeFromURLSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
}

func (s *MakeQRCodeFromURLSuite) TestMotherOfAllTests() {
	URLToTest := "2:5vd6SL:KFC:26gAouU6D6A4dCs9LK7jHmXZ3gjVdPczvX7yeusZRHTeR:3HxJ9Qr4H351dPoXjQYsdPX4tK6tV6TkdsHk1xMZEZmL:3"
	PublicKey := "0x10aded70ffee"

	err := s.m.MakeQRCodeFromURL(URLToTest, PublicKey)
	s.Require().NoError(err)

	var optionsThatAllowProfileImage = QROptions{
		AllowProfileImage: true,
		URL:               "cs2:5vd6SL:KFC:26gAouU6D6A4dCs9LK7jHmXZ3gjVdPczvX7yeusZRHTeR:3HxJ9Qr4H351dPoXjQYsdPX4tK6tV6TkdsHk1xMZEZmL:3",
	}

	var optionsThatDontAllowProfileImage = QROptions{
		AllowProfileImage: false,
		URL:               "cs2:5vd6SL:KFC:26gAouU6D6A4dCs9LK7jHmXZ3gjVdPczvX7yeusZRHTeR:3HxJ9Qr4H351dPoXjQYsdPX4tK6tV6TkdsHk1xMZEZmL:3",
	}

	err = s.m.MakeQRWithOptions(optionsThatAllowProfileImage)
	err = s.m.MakeQRWithOptions(optionsThatDontAllowProfileImage)
	s.Require().NoError(err)

}
