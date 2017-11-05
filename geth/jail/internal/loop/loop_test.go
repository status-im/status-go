package loop_test

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/status-go/geth/jail/internal/loop/looptask"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/vm"
	"github.com/stretchr/testify/suite"
)

func (s *LoopSuite) TestAddAndReady() {
	t := looptask.NewIdleTask()

	err := s.loop.Add(t)
	s.NoError(err)

	err = s.loop.Ready(t)
	s.NoError(err)

	s.cancel()

	// Wait for the context to cancel and loop to close
	time.Sleep(100 * time.Millisecond)

	err = s.loop.Add(t)
	s.Error(err)

	err = s.loop.Ready(t)
	s.Error(err)
}

type LoopSuite struct {
	suite.Suite

	loop   *loop.Loop
	cancel context.CancelFunc
}

func (s *LoopSuite) SetupTest() {
	o := otto.New()
	vm := vm.New(o)
	s.loop = loop.New(vm)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop.Run(ctx)
}

func TestLoopSuite(t *testing.T) {
	suite.Run(t, new(LoopSuite))
}
