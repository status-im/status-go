package protocol

import (
	"crypto/ecdsa"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/wakuv2"

	wakutypes "github.com/status-im/status-go/waku/types"
)

const DefaultProfileDisplayName = ""

func (s *MessengerBaseTestSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	shh, err := newTestWakuNode(s.logger)
	s.Require().NoError(err)
	s.Require().NoError(shh.Start())
	s.shh = shh

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
}

func (s *MessengerBaseTestSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.m)
	_ = s.logger.Sync()
}

func (s *MessengerBaseTestSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

type MessengerBaseTestSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    wakutypes.Waku
	logger *zap.Logger
}

func newMessengerWithKey(shh wakutypes.Waku, privateKey *ecdsa.PrivateKey, logger *zap.Logger, extraOptions []Option) (*Messenger, error) {
	options := []Option{
		WithAppSettings(settings.Settings{
			DisplayName:               DefaultProfileDisplayName,
			ProfilePicturesShowTo:     1,
			ProfilePicturesVisibility: 1,
			URLUnfurlingMode:          settings.URLUnfurlingAlwaysAsk,
		}, params.NodeConfig{}),
		WithMessageSigner(NewSignerStub()),
	}
	options = append(options, extraOptions...)

	m, err := newTestMessenger(shh, testMessengerConfig{
		privateKey:   privateKey,
		logger:       logger,
		extraOptions: options,
	})
	if err != nil {
		return nil, err
	}

	m.EnableBackedupMessagesProcessing()

	_, err = m.Start()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func newTestWakuNode(logger *zap.Logger) (wakutypes.Waku, error) {
	return wakuv2.New(
		nil,
		&wakuv2.DefaultConfig,
		logger,
		nil,
		nil,
		func([]byte, peer.AddrInfo, error) {},
		nil,
	)
}
