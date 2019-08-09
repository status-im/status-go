package whisper

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestWhisperExtensionSuite(t *testing.T) {
	suite.Run(t, new(WhisperExtensionSuite))
}

type WhisperExtensionSuite struct {
	suite.Suite

	nodes []*node.StatusNode
}

func (s *WhisperExtensionSuite) SetupTest() {
	s.nodes = make([]*node.StatusNode, 2)
	for i := range s.nodes {
		dir, err := ioutil.TempDir("", "test-shhext-")
		s.NoError(err)
		// network id is irrelevant
		cfg, err := utils.MakeTestNodeConfigWithDataDir(fmt.Sprintf("test-shhext-%d", i), dir, 777)
		s.Require().NoError(err)
		s.nodes[i] = node.New()
		s.Require().NoError(s.nodes[i].Start(cfg))
	}
}

func (s *WhisperExtensionSuite) TearDown() {
	for _, n := range s.nodes {
		cfg := n.Config()
		s.NotNil(cfg)
		s.NoError(n.Stop())
		s.NoError(os.Remove(cfg.DataDir))
	}
}
