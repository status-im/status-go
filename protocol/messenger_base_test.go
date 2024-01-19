package protocol

import (
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"

	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

const DefaultProfileDisplayName = ""

func TestMessengerCollapsedComunityCategoriesSuite(t *testing.T) {
	suite.Run(t, new(MessengerCollapsedCommunityCategoriesSuite))
}

func (s *MessengerBaseTestSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

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
	shh    types.Waku
	logger *zap.Logger
}

func newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey, logger *zap.Logger, extraOptions []Option) (*Messenger, error) {
	options := []Option{
		WithAppSettings(settings.Settings{
			DisplayName:               DefaultProfileDisplayName,
			ProfilePicturesShowTo:     1,
			ProfilePicturesVisibility: 1,
			URLUnfurlingMode:          settings.URLUnfurlingAlwaysAsk,
		}, params.NodeConfig{}),
	}
	options = append(options, extraOptions...)

	m, err := newTestMessenger(shh, testMessengerConfig{
		privateKey: privateKey,
		logger:     logger,
	}, options)
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
