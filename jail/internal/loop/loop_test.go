package loop

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/status-im/status-go/jail/internal/vm"
	"github.com/stretchr/testify/suite"
)

// DummyTask is something that satisfies the loop.Task interface for testing.
type DummyTask struct {
	canceled int32
	executed int32
}

func (*DummyTask) SetID(int64)  {}
func (*DummyTask) GetID() int64 { return 1 }
func (d *DummyTask) Cancel()    { atomic.StoreInt32(&d.canceled, 1) }
func (d *DummyTask) Execute(*vm.VM, *Loop) error {
	atomic.StoreInt32(&d.executed, 1)
	return nil
}

func (d *DummyTask) Canceled() bool {
	return atomic.LoadInt32(&d.canceled) == 1
}

func (d *DummyTask) Executed() bool {
	return atomic.LoadInt32(&d.executed) == 1
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

	newVM := vm.New()
	s.loop = New(newVM)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go func() {
		a := s.Assertions // Cache assertions reference as otherwise we'd incur in a race condition
		err := s.loop.Run(ctx)
		a.Equal(context.Canceled, err)
	}()
}

func (s *LoopSuite) TestAddAndReady() {

	err := s.loop.Add(s.task)
	s.NoError(err)
	s.False(s.task.Canceled())

	err = s.loop.Ready(s.task)
	s.NoError(err)

	// Wait to process task
	time.Sleep(100 * time.Millisecond)
	s.True(s.task.Executed())

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
	s.True(s.task.Canceled())
}

func (s *LoopSuite) TestImmediateExecution() {
	err := s.loop.AddAndExecute(s.task)

	// Wait for the task to execute
	time.Sleep(100 * time.Millisecond)

	s.NoError(err)
	s.True(s.task.Executed())
	s.False(s.task.Canceled())

	s.cancel()
}

func (s *LoopSuite) TestImmediateExecutionErrorWhenClosed() {
	s.cancel()

	// Wait for the context to cancel and loop to close
	time.Sleep(100 * time.Millisecond)

	err := s.loop.AddAndExecute(s.task)

	s.Error(err)
	s.False(s.task.Executed())

}
