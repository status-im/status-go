package node

import (
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	common2 "github.com/status-im/status-go/common"
)

type testBackgroundService struct {
	pauseCalls  int
	resumeCalls int
}

func (s *testBackgroundService) Start() error    { return nil }
func (s *testBackgroundService) Stop() error     { return nil }
func (s *testBackgroundService) APIs() []rpc.API { return nil }

func (s *testBackgroundService) PauseBackground() error {
	s.pauseCalls++
	return nil
}

func (s *testBackgroundService) ResumeForeground() error {
	s.resumeCalls++
	return nil
}

func TestPauseResumeBackgroundLifecycleHooks(t *testing.T) {
	bg := &testBackgroundService{}
	node := &StatusNode{
		services: []common2.StatusService{bg},
	}

	require.NoError(t, node.PauseBackground())
	require.Equal(t, 1, bg.pauseCalls)
	require.NoError(t, node.ResumeForeground())
	require.Equal(t, 1, bg.resumeCalls)
}
