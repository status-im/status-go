package healthmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/healthmanager/rpcstatus"
)

type ProvidersHealthManagerSuite struct {
	suite.Suite
	phm *ProvidersHealthManager
}

// SetupTest initializes the ProvidersHealthManager before each test
func (s *ProvidersHealthManagerSuite) SetupTest() {
	s.phm = NewProvidersHealthManager(1)
}

// Helper method to update providers and wait for a notification on the given channel
func (s *ProvidersHealthManagerSuite) updateAndWait(ch <-chan struct{}, statuses []rpcstatus.RpcProviderCallStatus, expectedChainStatus rpcstatus.StatusType, timeout time.Duration) {
	s.phm.Update(context.Background(), statuses)

	select {
	case <-ch:
		// Received notification
	case <-time.After(timeout):
		s.Fail("Timeout waiting for chain status update")
	}

	s.assertChainStatus(expectedChainStatus)
}

// Helper method to update providers and wait for a notification on the given channel
func (s *ProvidersHealthManagerSuite) updateAndExpectNoNotification(ch <-chan struct{}, statuses []rpcstatus.RpcProviderCallStatus, expectedChainStatus rpcstatus.StatusType, timeout time.Duration) {
	s.phm.Update(context.Background(), statuses)

	select {
	case <-ch:
		s.Fail("Unexpected status update")
	case <-time.After(timeout):
		// No notification as expected
	}

	s.assertChainStatus(expectedChainStatus)
}

// Helper method to assert the current chain status
func (s *ProvidersHealthManagerSuite) assertChainStatus(expected rpcstatus.StatusType) {
	actual := s.phm.Status().Status
	s.Equal(expected, actual, fmt.Sprintf("Expected chain status to be %s", expected))
}

func (s *ProvidersHealthManagerSuite) TestInitialStatus() {
	s.assertChainStatus(rpcstatus.StatusDown)
}

func (s *ProvidersHealthManagerSuite) TestUpdateProviderStatuses() {
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)
	s.updateAndWait(ch, []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: nil},
		{Name: "Provider2", Timestamp: time.Now(), Err: errors.New("connection error")},
	}, rpcstatus.StatusUp, time.Second)

	statusMap := s.phm.GetStatuses()
	s.Len(statusMap, 2, "Expected 2 provider statuses")
	s.Equal(rpcstatus.StatusUp, statusMap["Provider1"].Status, "Expected Provider1 status to be Up")
	s.Equal(rpcstatus.StatusDown, statusMap["Provider2"].Status, "Expected Provider2 status to be Down")
}

func (s *ProvidersHealthManagerSuite) TestChainStatusUpdatesOnce() {
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)
	s.assertChainStatus(rpcstatus.StatusDown)

	// Update providers to Down
	statuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: errors.New("error")},
		{Name: "Provider2", Timestamp: time.Now(), Err: nil},
	}
	s.updateAndWait(ch, statuses, rpcstatus.StatusUp, time.Second)
	s.updateAndExpectNoNotification(ch, statuses, rpcstatus.StatusUp, 10*time.Millisecond)
}

func (s *ProvidersHealthManagerSuite) TestSubscribeReceivesOnlyOnChange() {
	ch := s.phm.Subscribe()
	defer s.phm.Unsubscribe(ch)

	// Update provider to Up and wait for notification
	upStatuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: nil},
	}
	s.updateAndWait(ch, upStatuses, rpcstatus.StatusUp, time.Second)

	// Update provider to Down and wait for notification
	downStatuses := []rpcstatus.RpcProviderCallStatus{
		{Name: "Provider1", Timestamp: time.Now(), Err: errors.New("some critical error")},
	}
	s.updateAndWait(ch, downStatuses, rpcstatus.StatusDown, time.Second)

	s.updateAndExpectNoNotification(ch, downStatuses, rpcstatus.StatusDown, 10*time.Millisecond)
}

func (s *ProvidersHealthManagerSuite) TestConcurrency() {
	var wg sync.WaitGroup
	providerCount := 1000

	s.phm.Update(context.Background(), []rpcstatus.RpcProviderCallStatus{
		{Name: "ProviderUp", Timestamp: time.Now(), Err: nil},
	})

	ctx := context.Background()
	for i := 0; i < providerCount-1; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			providerName := fmt.Sprintf("Provider%d", i)
			var err error
			if i%2 == 0 {
				err = errors.New("error")
			}
			s.phm.Update(ctx, []rpcstatus.RpcProviderCallStatus{
				{Name: providerName, Timestamp: time.Now(), Err: err},
			})
		}(i)
	}
	wg.Wait()

	statuses := s.phm.GetStatuses()
	s.Len(statuses, providerCount, "Expected 1000 provider statuses")

	chainStatus := s.phm.Status().Status
	s.Equal(chainStatus, rpcstatus.StatusUp, "Expected chain status to be either Up or Down")
}

func TestProvidersHealthManagerSuite(t *testing.T) {
	suite.Run(t, new(ProvidersHealthManagerSuite))
}
