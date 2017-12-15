package loop

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"

	"github.com/status-im/status-go/geth/jail/internal/vm"
	"github.com/stretchr/testify/suite"
)

func TestLoopSuite(t *testing.T) {
	suite.Run(t, new(LoopSuite))
}

type LoopSuite struct {
	suite.Suite

	loop   *Loop
	cancel context.CancelFunc

	mockTask     *MockTask
	loopMockCtrl *gomock.Controller
}

func (s *LoopSuite) SetupTest() {
	s.loopMockCtrl = gomock.NewController(s.T())
	s.mockTask = NewMockTask(s.loopMockCtrl)

	vm := vm.New()
	s.loop = New(vm)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop.Run(ctx)
}

func (s *LoopSuite) TearDownTest() {
	s.loopMockCtrl.Finish()
}

func (s *LoopSuite) TestAddAndReady() {
	s.mockTask.EXPECT().SetID(int64(1)).Times(1)
	s.mockTask.EXPECT().GetID().Times(1)
	s.mockTask.EXPECT().Cancel().Times(1)

	err := s.loop.Add(s.mockTask)
	s.NoError(err)

	err = s.loop.Ready(s.mockTask)
	s.NoError(err)

	s.cancel()

	// Wait for the context to cancel and loop to close
	time.Sleep(100 * time.Millisecond)
}

func (s *LoopSuite) TestLoopErrorWhenClosed() {
	s.cancel()

	// Wait for the context to cancel and loop to close
	time.Sleep(100 * time.Millisecond)

	err := s.loop.Add(s.mockTask)
	s.Error(err)

	err = s.loop.Ready(s.mockTask)
	s.Error(err)
}
