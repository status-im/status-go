package publisher

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/testutils"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}

type PublisherTestSuite struct {
	suite.Suite
	publisher *Publisher
	logger    *zap.Logger
}

func (p *PublisherTestSuite) SetupTest(installationID string) {
	p.logger = testutils.MustCreateTestLogger()
	p.publisher = New(p.logger)
}

func (p *PublisherTestSuite) TearDownTest() {
	_ = p.logger.Sync()
}

// TODO(adam): provide more tests
