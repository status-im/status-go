package healthmanager

import (
	"context"
	"errors"
	"fmt"
	"github.com/status-im/status-go/healthmanager/rpcstatus"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
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
	s.updateAndWait(s.phm.Subscribe(), []rpcstatus.RpcProviderCallStatus{
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

func (s *BlockchainHealthManagerSuite) TestInterleavedChainStatusChanges() {
	// Register providers for chains 1, 2, and 3
	phm1 := NewProvidersHealthManager(1)
	phm2 := NewProvidersHealthManager(2)
	phm3 := NewProvidersHealthManager(3)
	s.manager.RegisterProvidersHealthManager(s.ctx, phm1)
	s.manager.RegisterProvidersHealthManager(s.ctx, phm2)
	s.manager.RegisterProvidersHealthManager(s.ctx, phm3)

	// Subscribe to status updates
	ch := s.manager.Subscribe()
	defer s.manager.Unsubscribe(ch)

	// Initially, all chains are up
	phm1.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider_chain1", Timestamp: time.Now(), Err: nil}})
	phm2.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider_chain2", Timestamp: time.Now(), Err: nil}})
	phm3.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider_chain3", Timestamp: time.Now(), Err: nil}})

	// Wait for the status to propagate
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)

	// Now chain 1 goes down, and chain 3 goes down at the same time
	phm1.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider_chain1", Timestamp: time.Now(), Err: errors.New("connection error")}})
	phm3.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider_chain3", Timestamp: time.Now(), Err: errors.New("connection error")}})

	// Wait for the status to reflect the changes
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)

	// Check that short status correctly reflects the mixed state
	shortStatus := s.manager.GetShortStatus()
	s.Equal(rpcstatus.StatusUp, shortStatus.Status.Status)
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[1].Status) // Chain 1 is down
	s.Equal(rpcstatus.StatusUp, shortStatus.StatusPerChain[2].Status)   // Chain 2 is still up
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[3].Status) // Chain 3 is down
}

func (s *BlockchainHealthManagerSuite) TestDelayedChainUpdate() {
	// Register providers for chains 1 and 2
	phm1 := NewProvidersHealthManager(1)
	phm2 := NewProvidersHealthManager(2)
	s.manager.RegisterProvidersHealthManager(s.ctx, phm1)
	s.manager.RegisterProvidersHealthManager(s.ctx, phm2)

	// Subscribe to status updates
	ch := s.manager.Subscribe()
	defer s.manager.Unsubscribe(ch)

	// Initially, both chains are up
	phm1.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain1", Timestamp: time.Now(), Err: nil}})
	phm2.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain2", Timestamp: time.Now(), Err: nil}})
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)

	// Chain 2 goes down
	phm2.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain2", Timestamp: time.Now(), Err: errors.New("connection error")}})
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)

	// Chain 1 goes down after a delay
	phm1.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain1", Timestamp: time.Now(), Err: errors.New("connection error")}})
	s.waitForUpdate(ch, rpcstatus.StatusDown, 100*time.Millisecond)

	// Check that short status reflects the final state where both chains are down
	shortStatus := s.manager.GetShortStatus()
	s.Equal(rpcstatus.StatusDown, shortStatus.Status.Status)
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[1].Status) // Chain 1 is down
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[2].Status) // Chain 2 is down
}

func TestProvidersHealthManagerSuite(t *testing.T) {
	suite.Run(t, new(ProvidersHealthManagerSuite))
}
