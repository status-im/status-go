package loop

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/status-go/geth/jail/internal/vm"
	"github.com/stretchr/testify/suite"
)

// DummyTask is something that satisfies the loop.Task interface for testing.
type DummyTask struct {
	canceled bool
	executed bool
}

func (*DummyTask) SetID(int64)  {}
func (*DummyTask) GetID() int64 { return 1 }
func (d *DummyTask) Cancel()    { d.canceled = true }
func (d *DummyTask) Execute(*vm.VM, *Loop) error {
	d.executed = true
	return nil
}

func TestLoopSuite(t *testing.T) {
	suite.Run(t, new(LoopSuite))
}

type LoopSuite struct {
	suite.Suite

	loop   *Loop
	cancel context.CancelFunc

	task *DummyTask
}

func (s *LoopSuite) SetupTest() {
	s.task = &DummyTask{}

	vm := vm.New()
	s.loop = New(vm)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop.Run(ctx)
}

func (s *LoopSuite) TestAddAndReady() {

	err := s.loop.Add(s.task)
	s.NoError(err)
	s.False(s.task.canceled)

	err = s.loop.Ready(s.task)
	s.NoError(err)

	// Wait to process task
	time.Sleep(100 * time.Millisecond)
	s.True(s.task.executed)

	s.cancel()
}

func (s *LoopSuite) TestLoopErrorWhenClosed() {
	s.cancel()

	// Wait for the context to cancel and loop to close
	time.Sleep(100 * time.Millisecond)

	err := s.loop.Add(s.task)
	s.Error(err)
	s.True(s.task.canceled)

	s.task.canceled = false
	err = s.loop.Ready(s.task)
	s.Error(err)
	s.True(s.task.canceled)
}
