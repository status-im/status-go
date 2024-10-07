package healthmanager

import (
	"context"
	"fmt"
	"github.com/status-im/status-go/healthmanager/aggregator"
	"github.com/status-im/status-go/healthmanager/rpcstatus"
	"sync"
)

// BlockchainFullStatus contains the full status of the blockchain, including provider statuses.
type BlockchainFullStatus struct {
	Status                    rpcstatus.ProviderStatus                       `json:"status"`
	StatusPerChainPerProvider map[uint64]map[string]rpcstatus.ProviderStatus `json:"statusPerChainPerProvider"`
}

// BlockchainHealthManager manages the state of all providers and aggregates their statuses.
type BlockchainHealthManager struct {
	mu          sync.RWMutex
	aggregator  *aggregator.Aggregator
	subscribers []chan struct{}

	providers   map[uint64]*ProvidersHealthManager
	cancelFuncs map[uint64]context.CancelFunc // Map chainID to cancel functions
	lastStatus  BlockchainStatus
	wg          sync.WaitGroup
}

// NewBlockchainHealthManager creates a new instance of BlockchainHealthManager.
func NewBlockchainHealthManager() *BlockchainHealthManager {
	agg := aggregator.NewAggregator("blockchain")
	return &BlockchainHealthManager{
		aggregator:  agg,
		providers:   make(map[uint64]*ProvidersHealthManager),
		cancelFuncs: make(map[uint64]context.CancelFunc),
	}
}

// RegisterProvidersHealthManager registers the provider health manager.
// It prevents registering the same provider twice for the same chain.
func (b *BlockchainHealthManager) RegisterProvidersHealthManager(ctx context.Context, phm *ProvidersHealthManager) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if the provider for the given chainID is already registered
	if _, exists := b.providers[phm.ChainID()]; exists {
		// Log a warning or return an error to indicate that the provider is already registered
		return fmt.Errorf("provider for chainID %d is already registered", phm.ChainID())
	}

	// Proceed with the registration
	b.providers[phm.ChainID()] = phm

	// Create a new context for the provider
	providerCtx, cancel := context.WithCancel(ctx)
	b.cancelFuncs[phm.ChainID()] = cancel

	statusCh := phm.Subscribe()
	b.wg.Add(1)
	go func(phm *ProvidersHealthManager, statusCh chan struct{}, providerCtx context.Context) {
		defer func() {
			b.wg.Done()
			phm.Unsubscribe(statusCh)
		}()
		for {
			select {
			case <-statusCh:
				// When the provider updates its status, check the statuses of all providers
				b.aggregateAndUpdateStatus(providerCtx)
			case <-providerCtx.Done():
				// Stop processing when the context is cancelled
				return
			}
		}
	}(phm, statusCh, providerCtx)

	return nil
}

// Stop stops the event processing and unsubscribes.
func (b *BlockchainHealthManager) Stop() {
	b.mu.Lock()

	for _, cancel := range b.cancelFuncs {
		cancel()
	}
	clear(b.cancelFuncs)

	b.mu.Unlock()
	b.wg.Wait()
}

// Subscribe allows clients to receive notifications about changes.
func (b *BlockchainHealthManager) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber from receiving notifications.
func (b *BlockchainHealthManager) Unsubscribe(ch chan struct{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove the subscriber channel from the list
	for i, subscriber := range b.subscribers {
		if subscriber == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// aggregateAndUpdateStatus collects statuses from all providers and updates the overall and short status.
func (b *BlockchainHealthManager) aggregateAndUpdateStatus(ctx context.Context) {
	b.mu.Lock()

	// Collect statuses from all providers
	providerStatuses := make([]rpcstatus.ProviderStatus, 0)
	for _, provider := range b.providers {
		providerStatuses = append(providerStatuses, provider.Status())
	}

	// Update the aggregator with the new list of provider statuses
	b.aggregator.UpdateBatch(providerStatuses)

	// Get the new aggregated full and short status
	newShortStatus := b.getShortStatus()
	b.mu.Unlock()

	// Compare full and short statuses and emit if changed
	if !compareShortStatus(newShortStatus, b.lastStatus) {
		b.emitBlockchainHealthStatus(ctx)
		b.mu.Lock()
		defer b.mu.Unlock()
		b.lastStatus = newShortStatus
	}
}

// compareShortStatus compares two BlockchainStatus structs and returns true if they are identical.
func compareShortStatus(newStatus, previousStatus BlockchainStatus) bool {
	if newStatus.Status.Status != previousStatus.Status.Status {
		return false
	}

	if len(newStatus.StatusPerChain) != len(previousStatus.StatusPerChain) {
		return false
	}

	for chainID, newChainStatus := range newStatus.StatusPerChain {
		if prevChainStatus, ok := previousStatus.StatusPerChain[chainID]; !ok || newChainStatus.Status != prevChainStatus.Status {
			return false
		}
	}

	return true
}

// emitBlockchainHealthStatus sends a notification to all subscribers about the new blockchain status.
func (b *BlockchainHealthManager) emitBlockchainHealthStatus(ctx context.Context) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, subscriber := range b.subscribers {
		select {
		case <-ctx.Done():
			// Stop sending notifications when the context is cancelled
			return
		case subscriber <- struct{}{}:
		default:
			// Skip notification if the subscriber's channel is full
		}
	}
}

func (b *BlockchainHealthManager) GetFullStatus() BlockchainFullStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	statusPerChainPerProvider := make(map[uint64]map[string]rpcstatus.ProviderStatus)

	for chainID, phm := range b.providers {
		providerStatuses := phm.GetStatuses()
		statusPerChainPerProvider[chainID] = providerStatuses
	}

	blockchainStatus := b.aggregator.GetAggregatedStatus()

	return BlockchainFullStatus{
		Status:                    blockchainStatus,
		StatusPerChainPerProvider: statusPerChainPerProvider,
	}
}

func (b *BlockchainHealthManager) getShortStatus() BlockchainStatus {
	statusPerChain := make(map[uint64]rpcstatus.ProviderStatus)

	for chainID, phm := range b.providers {
		chainStatus := phm.Status()
		statusPerChain[chainID] = chainStatus
	}

	blockchainStatus := b.aggregator.GetAggregatedStatus()

	return BlockchainStatus{
		Status:         blockchainStatus,
		StatusPerChain: statusPerChain,
	}
}

func (b *BlockchainHealthManager) GetShortStatus() BlockchainStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.getShortStatus()
}

// Status returns the current aggregated status.
func (b *BlockchainHealthManager) Status() rpcstatus.ProviderStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.aggregator.GetAggregatedStatus()
}
