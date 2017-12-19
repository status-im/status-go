package loop

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/status-go/geth/jail/internal/vm"
	"github.com/stretchr/testify/suite"
)

// DummyTask is something that satisfies the loop.Task interface for testing.
type DummyTask struct{}

func (DummyTask) SetID(int64)                 {}
func (DummyTask) GetID() int64                { return 1 }
func (DummyTask) Cancel()                     {}
func (DummyTask) Execute(*vm.VM, *Loop) error { return nil }

func TestLoopSuite(t *testing.T) {
	suite.Run(t, new(LoopSuite))
}

type LoopSuite struct {
	suite.Suite

	loop   *Loop
	cancel context.CancelFunc

	task DummyTask
}

func (s *LoopSuite) SetupTest() {
	s.task = DummyTask{}

	vm := vm.New()
	s.loop = New(vm)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop.Run(ctx)
}

func (s *LoopSuite) TestAddAndReady() {

	err := s.loop.Add(s.task)
	s.NoError(err)

	err = s.loop.Ready(s.task)
	s.NoError(err)

	// Wait to process task
	time.Sleep(100 * time.Millisecond)

	s.cancel()
}

func (s *LoopSuite) TestLoopErrorWhenClosed() {
	s.cancel()

	// Wait for the context to cancel and loop to close
	time.Sleep(100 * time.Millisecond)

	err := s.loop.Add(s.task)
	s.Error(err)

	err = s.loop.Ready(s.task)
	s.Error(err)
}
