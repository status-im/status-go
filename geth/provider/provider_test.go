package provider_test

// Basic imports
import (
	"testing"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/provider"
	"github.com/stretchr/testify/suite"
)

var noRunningNodeMsg = "there is no running node"

type ServiceProviderTestSuite struct {
	suite.Suite
	p *provider.ServiceProvider
	n *node.NodeManager
}

func (s *ServiceProviderTestSuite) SetupTest() {
	s.n = node.NewNodeManager()
	s.p = provider.New(s.n)
}

func (s *ServiceProviderTestSuite) TestNodeManagerForNonStartedNode() {
	n := s.p.NodeManager()
	s.Equal(s.n, n)
}

func (s *ServiceProviderTestSuite) TestAccountManagerForNonStartedNode() {
	a, err := s.p.AccountManager()
	s.Nil(err)
	s.NotNil(a)
}

func (s *ServiceProviderTestSuite) TestJailManagerForNonStartedNode() {
	a := s.p.JailManager()
	s.NotNil(a)
}

func (s *ServiceProviderTestSuite) TestTxQueueManagerForNonStartedNode() {
	a := s.p.TxQueueManager()
	s.NotNil(a)
}

func (s *ServiceProviderTestSuite) TestGettersForNonStartedNode() {
	var flagtests = []struct {
		fn          func(p *provider.ServiceProvider) error
		expectation string
	}{
		{func(p *provider.ServiceProvider) error { _, err := p.Node(); return err }, noRunningNodeMsg},
		{func(p *provider.ServiceProvider) error { _, err := p.Account(); return err }, noRunningNodeMsg},
		{func(p *provider.ServiceProvider) error { _, err := p.AccountKeyStore(); return err }, noRunningNodeMsg},
		{func(p *provider.ServiceProvider) error { _, err := p.Whisper(); return err }, noRunningNodeMsg},
	}

	for _, tt := range flagtests {
		err := tt.fn(s.p)
		s.Equal(tt.expectation, err.Error())
	}
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceProviderTestSuite))
}
