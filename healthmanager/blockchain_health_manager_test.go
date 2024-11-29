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

type BlockchainHealthManagerSuite struct {
	suite.Suite
	manager *BlockchainHealthManager
	ctx     context.Context
	cancel  context.CancelFunc
}

func (s *BlockchainHealthManagerSuite) SetupTest() {
	s.manager = NewBlockchainHealthManager()
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *BlockchainHealthManagerSuite) TearDownTest() {
	s.manager.Stop()
	s.cancel()
}

// Helper method to update providers and wait for a notification on the given channel
func (s *BlockchainHealthManagerSuite) waitForUpdate(ch <-chan struct{}, expectedChainStatus rpcstatus.StatusType, timeout time.Duration) {
	select {
	case <-ch:
		// Received notification
	case <-time.After(timeout):
		s.Fail("Timeout waiting for chain status update")
	}

	s.assertBlockChainStatus(expectedChainStatus)
}

// Helper method to assert the current chain status
func (s *BlockchainHealthManagerSuite) assertBlockChainStatus(expected rpcstatus.StatusType) {
	actual := s.manager.Status().Status
	s.Equal(expected, actual, fmt.Sprintf("Expected blockchain status to be %s", expected))
}

// Test registering a provider health manager
func (s *BlockchainHealthManagerSuite) TestRegisterProvidersHealthManager() {
	phm := NewProvidersHealthManager(1) // Create a real ProvidersHealthManager
	err := s.manager.RegisterProvidersHealthManager(context.Background(), phm)
	s.Require().NoError(err)

	// Verify that the provider is registered
	s.Require().NotNil(s.manager.providers[1])
}

// Test status updates and notifications
func (s *BlockchainHealthManagerSuite) TestStatusUpdateNotification() {
	phm := NewProvidersHealthManager(1)
	err := s.manager.RegisterProvidersHealthManager(context.Background(), phm)
	s.Require().NoError(err)
	ch := s.manager.Subscribe()

	// Update the provider status
	phm.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{
		{Name: "providerName", Timestamp: time.Now(), Err: nil},
	})

	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)
}

// Test getting the full status
func (s *BlockchainHealthManagerSuite) TestGetFullStatus() {
	phm1 := NewProvidersHealthManager(1)
	phm2 := NewProvidersHealthManager(2)
	ctx := context.Background()
	err := s.manager.RegisterProvidersHealthManager(ctx, phm1)
	s.Require().NoError(err)
	err = s.manager.RegisterProvidersHealthManager(ctx, phm2)
	s.Require().NoError(err)
	ch := s.manager.Subscribe()

	// Update the provider status
	phm1.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{
		{Name: "providerName1", Timestamp: time.Now(), Err: nil},
	})
	phm2.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{
		{Name: "providerName2", Timestamp: time.Now(), Err: errors.New("connection error")},
	})

	s.waitForUpdate(ch, rpcstatus.StatusUp, 10*time.Millisecond)
	fullStatus := s.manager.GetFullStatus()
	s.Len(fullStatus.StatusPerChainPerProvider, 2, "Expected statuses for 2 chains")
}

func (s *BlockchainHealthManagerSuite) TestConcurrentSubscriptionUnsubscription() {
	var wg sync.WaitGroup
	subscribersCount := 100

	// Concurrently add and remove subscribers
	for i := 0; i < subscribersCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			subCh := s.manager.Subscribe()
			time.Sleep(10 * time.Millisecond)
			s.manager.Unsubscribe(subCh)
		}()
	}

	wg.Wait()
	// After all subscribers are removed, there should be no active subscribers
	s.Equal(0, len(s.manager.subscriptionManager.subscribers), "Expected no subscribers after unsubscription")
}
func (s *BlockchainHealthManagerSuite) TestConcurrency() {
	var wg sync.WaitGroup
	chainsCount := 10
	providersCount := 100
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	for i := 1; i <= chainsCount; i++ {
		phm := NewProvidersHealthManager(uint64(i))
		err := s.manager.RegisterProvidersHealthManager(ctx, phm)
		s.Require().NoError(err)
	}

	ch := s.manager.Subscribe()
	defer s.manager.Unsubscribe(ch)

	for i := 1; i <= chainsCount; i++ {
		wg.Add(1)
		go func(chainID uint64) {
			defer wg.Done()
			phm := s.manager.providers[chainID]
			for j := 0; j < providersCount; j++ {
				wg.Add(1)
				err := errors.New("connection error")
				if j == providersCount-1 {
					err = nil
				}
				name := fmt.Sprintf("provider-%d", j)

				go func(name string, err error) {
					defer wg.Done()
					phm.Update(ctx, []rpcstatus.RpcProviderCallStatus{
						{Name: name, Timestamp: time.Now(), Err: err},
					})
				}(name, err)
			}
		}(uint64(i))
	}

	wg.Wait()

	s.waitForUpdate(ch, rpcstatus.StatusUp, 2*time.Second)
}

func (s *BlockchainHealthManagerSuite) TestMultipleStartAndStop() {
	s.manager.Stop()

	s.manager.Stop()

	// Ensure that the manager is in a clean state after multiple starts and stops
	s.Equal(0, len(s.manager.cancelFuncs), "Expected no cancel functions after stop")
}

func (s *BlockchainHealthManagerSuite) TestUnsubscribeOneOfMultipleSubscribers() {
	// Create an instance of BlockchainHealthManager and register a provider manager
	phm := NewProvidersHealthManager(1)
	ctx, cancel := context.WithCancel(s.ctx)
	err := s.manager.RegisterProvidersHealthManager(ctx, phm)
	s.Require().NoError(err)

	defer cancel()

	// Subscribe two subscribers
	subscriber1 := s.manager.Subscribe()
	subscriber2 := s.manager.Subscribe()

	// Unsubscribe the first subscriber
	s.manager.Unsubscribe(subscriber1)

	phm.Update(ctx, []rpcstatus.RpcProviderCallStatus{
		{Name: "provider-1", Timestamp: time.Now(), Err: nil},
	})

	// Ensure the first subscriber did not receive a notification
	select {
	case _, ok := <-subscriber1:
		s.False(ok, "First subscriber channel should be closed")
	default:
		s.Fail("First subscriber channel was not closed")
	}

	// Ensure the second subscriber received a notification
	select {
	case <-subscriber2:
		// Notification received by the second subscriber
	case <-time.After(100 * time.Millisecond):
		s.Fail("Second subscriber should have received a notification")
	}
}

func (s *BlockchainHealthManagerSuite) TestMixedProviderStatusInSingleChain() {
	// Register a provider for chain 1
	phm := NewProvidersHealthManager(1)
	err := s.manager.RegisterProvidersHealthManager(s.ctx, phm)
	s.Require().NoError(err)

	// Subscribe to status updates
	ch := s.manager.Subscribe()
	defer s.manager.Unsubscribe(ch)

	// Simulate mixed statuses within the same chain (one provider up, one provider down)
	phm.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{
		{Name: "provider1_chain1", Timestamp: time.Now(), Err: nil},                 // Provider 1 is up
		{Name: "provider2_chain1", Timestamp: time.Now(), Err: errors.New("error")}, // Provider 2 is down
	})

	// Wait for the status to propagate
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)

	// Verify that the short status reflects the chain as down, since one provider is down
	shortStatus := s.manager.GetStatusPerChain()
	s.Equal(rpcstatus.StatusUp, shortStatus.Status.Status)
	s.Equal(rpcstatus.StatusUp, shortStatus.StatusPerChain[1].Status) // Chain 1 should be marked as down
}

func (s *BlockchainHealthManagerSuite) TestInterleavedChainStatusChanges() {
	// Register providers for chains 1, 2, and 3
	phm1 := NewProvidersHealthManager(1)
	phm2 := NewProvidersHealthManager(2)
	phm3 := NewProvidersHealthManager(3)
	err := s.manager.RegisterProvidersHealthManager(s.ctx, phm1)
	s.Require().NoError(err)
	err = s.manager.RegisterProvidersHealthManager(s.ctx, phm2)
	s.Require().NoError(err)
	err = s.manager.RegisterProvidersHealthManager(s.ctx, phm3)
	s.Require().NoError(err)

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
	shortStatus := s.manager.GetStatusPerChain()
	s.Equal(rpcstatus.StatusUp, shortStatus.Status.Status)
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[1].Status) // Chain 1 is down
	s.Equal(rpcstatus.StatusUp, shortStatus.StatusPerChain[2].Status)   // Chain 2 is still up
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[3].Status) // Chain 3 is down
}

func (s *BlockchainHealthManagerSuite) TestDelayedChainUpdate() {
	// Register providers for chains 1 and 2
	phm1 := NewProvidersHealthManager(1)
	phm2 := NewProvidersHealthManager(2)
	err := s.manager.RegisterProvidersHealthManager(s.ctx, phm1)
	s.Require().NoError(err)
	err = s.manager.RegisterProvidersHealthManager(s.ctx, phm2)
	s.Require().NoError(err)

	// Subscribe to status updates
	ch := s.manager.Subscribe()
	defer s.manager.Unsubscribe(ch)

	// Initially, both chains are up
	phm1.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain1", Timestamp: time.Now(), Err: nil}})
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)
	phm2.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain2", Timestamp: time.Now(), Err: nil}})
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)

	// Chain 2 goes down
	phm2.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain2", Timestamp: time.Now(), Err: errors.New("connection error")}})
	s.waitForUpdate(ch, rpcstatus.StatusUp, 100*time.Millisecond)

	// Chain 1 goes down after a delay
	phm1.Update(s.ctx, []rpcstatus.RpcProviderCallStatus{{Name: "provider1_chain1", Timestamp: time.Now(), Err: errors.New("connection error")}})
	s.waitForUpdate(ch, rpcstatus.StatusDown, 100*time.Millisecond)

	// Check that short status reflects the final state where both chains are down
	shortStatus := s.manager.GetStatusPerChain()
	s.Equal(rpcstatus.StatusDown, shortStatus.Status.Status)
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[1].Status) // Chain 1 is down
	s.Equal(rpcstatus.StatusDown, shortStatus.StatusPerChain[2].Status) // Chain 2 is down
}

func TestBlockchainHealthManagerSuite(t *testing.T) {
	suite.Run(t, new(BlockchainHealthManagerSuite))
}
