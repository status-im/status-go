package backend

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/backend/node"
)

type CallInProcessRPCSuite struct {
	suite.Suite
	backend *StatusBackend
}

func TestCallInProcessRPCSuite(t *testing.T) {
	suite.Run(t, new(CallInProcessRPCSuite))
}

func (s *CallInProcessRPCSuite) SetupTest() {
	s.backend = NewStatusBackend(testutils.MustCreateTestLogger())
	s.backend.UpdateRootDataDir(s.T().TempDir())
}

// The node is gone between stopSession and initialize during a logout; a client
// RPC arriving then must get an error response instead of crashing the process.
func (s *CallInProcessRPCSuite) TestReturnsErrorWhileSessionIsStopped() {
	s.backend.mu.Lock()
	s.Require().NoError(s.backend.stopSession())
	s.backend.mu.Unlock()
	s.Require().Nil(s.backend.StatusNode())

	var response struct {
		Version string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	s.Require().NotPanics(func() {
		output := s.backend.CallInProcessRPC(`{"jsonrpc":"2.0","id":42,"method":"settings_getSettings","params":[]}`)
		s.Require().NoError(json.Unmarshal([]byte(output), &response))
	})
	s.Require().Equal("2.0", response.Version)
	s.Require().Equal(42, response.ID)
	s.Require().Equal(-32000, response.Error.Code)
	s.Require().Equal(node.ErrNoRunningNode.Error(), response.Error.Message)
}
