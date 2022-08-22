package protocol

import (
	"context"
	"crypto/ecdsa"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/waku"
)

func TestBrowserSuite(t *testing.T) {
	suite.Run(t, new(BrowserSuite))
}

type BrowserSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *BrowserSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *BrowserSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *BrowserSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerBackupSuite) TestBrowsersOrderedNewestFirst() {
	msngr := s.newMessenger()
	testBrowsers := []*browsers.Browser{
		{
			ID:        "1",
			Name:      "first",
			Dapp:      true,
			Timestamp: 10,
		},
		{
			ID:        "2",
			Name:      "second",
			Dapp:      true,
			Timestamp: 50,
		},
		{
			ID:           "3",
			Name:         "third",
			Dapp:         true,
			Timestamp:    100,
			HistoryIndex: 0,
			History:      []string{"zero"},
		},
	}
	for i := 0; i < len(testBrowsers); i++ {
		s.Require().NoError(msngr.AddBrowser(context.TODO(), *testBrowsers[i]))
	}

	sort.Slice(testBrowsers, func(i, j int) bool {
		return testBrowsers[i].Timestamp > testBrowsers[j].Timestamp
	})

	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Equal(testBrowsers, rst)
}

func (s *MessengerBackupSuite) TestBrowsersHistoryIncluded() {
	msngr := s.newMessenger()
	browser := &browsers.Browser{
		ID:           "1",
		Name:         "first",
		Dapp:         true,
		Timestamp:    10,
		HistoryIndex: 1,
		History:      []string{"one", "two"},
	}
	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 1)
	s.Require().Equal(browser, rst[0])
}

func (s *MessengerBackupSuite) TestBrowsersReplaceOnUpdate() {
	msngr := s.newMessenger()
	browser := &browsers.Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: 10,
		History:   []string{"one", "two"},
	}
	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	browser.Dapp = false
	browser.History = []string{"one", "three"}
	browser.Timestamp = 107
	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 1)
	s.Require().Equal(browser, rst[0])
}

func (s *MessengerBackupSuite) TestDeleteBrowser() {
	msngr := s.newMessenger()
	browser := &browsers.Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: 10,
		History:   []string{"one", "two"},
	}

	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 1)

	s.Require().NoError(msngr.DeleteBrowser(context.TODO(), browser.ID))
	rst, err = msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 0)
}
