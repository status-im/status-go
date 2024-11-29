package healthmanager

import (
	"context"
	"sync"

	status_common "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/healthmanager/aggregator"
	"github.com/status-im/status-go/healthmanager/rpcstatus"
)

// BlockchainFullStatus contains the full status of the blockchain, including provider statuses.
type BlockchainFullStatus struct {
	Status                    rpcstatus.ProviderStatus                       `json:"status"`
	StatusPerChain            map[uint64]rpcstatus.ProviderStatus            `json:"statusPerChain"`
	StatusPerChainPerProvider map[uint64]map[string]rpcstatus.ProviderStatus `json:"statusPerChainPerProvider"`
}

// BlockchainStatus contains the status of the blockchain
type BlockchainStatus struct {
	Status         rpcstatus.ProviderStatus            `json:"status"`
	StatusPerChain map[uint64]rpcstatus.ProviderStatus `json:"statusPerChain"`
}

type BlockchainHealthManager struct {
	mu                  sync.RWMutex
	aggregator          *aggregator.Aggregator
	subscriptionManager *SubscriptionManager

	providers   map[uint64]*ProvidersHealthManager
	cancelFuncs map[uint64]context.CancelFunc // Map chainID to cancel functions
	lastStatus  *BlockchainStatus
	wg          sync.WaitGroup
}

// NewBlockchainHealthManager creates a new instance of BlockchainHealthManager.
func NewBlockchainHealthManager() *BlockchainHealthManager {
	agg := aggregator.NewAggregator("blockchain")
	return &BlockchainHealthManager{
		aggregator:          agg,
		providers:           make(map[uint64]*ProvidersHealthManager),
		cancelFuncs:         make(map[uint64]context.CancelFunc),
		subscriptionManager: &SubscriptionManager{subscribers: make(map[chan struct{}]struct{})},
	}
}

// RegisterProvidersHealthManager registers the provider health manager.
// It removes any existing provider for the same chain before registering the new one.
func (b *BlockchainHealthManager) RegisterProvidersHealthManager(ctx context.Context, phm *ProvidersHealthManager) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	chainID := phm.ChainID()

	// Check if a provider for the given chainID is already registered and remove it
	if _, exists := b.providers[chainID]; exists {
		// Cancel the existing context
		if cancel, cancelExists := b.cancelFuncs[chainID]; cancelExists {
			cancel()
		}
		// Remove the old registration
		delete(b.providers, chainID)
		delete(b.cancelFuncs, chainID)
	}

	// Proceed with the registration
	b.providers[chainID] = phm

	// Create a new context for the provider
	providerCtx, cancel := context.WithCancel(ctx)
	b.cancelFuncs[chainID] = cancel

	statusCh := phm.Subscribe()
	b.wg.Add(1)
	go func(phm *ProvidersHealthManager, statusCh chan struct{}, providerCtx context.Context) {
		defer status_common.LogOnPanic()
		defer func() {
			phm.Unsubscribe(statusCh)
			b.wg.Done()
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
	clear(b.providers)

	b.mu.Unlock()
	b.wg.Wait()
}

// Subscribe allows clients to receive notifications about changes.
func (b *BlockchainHealthManager) Subscribe() chan struct{} {
	return b.subscriptionManager.Subscribe()
}

// Unsubscribe removes a subscriber from receiving notifications.
func (b *BlockchainHealthManager) Unsubscribe(ch chan struct{}) {
	b.subscriptionManager.Unsubscribe(ch)
}

// aggregateAndUpdateStatus collects statuses from all providers and updates the overall and short status.
func (b *BlockchainHealthManager) aggregateAndUpdateStatus(ctx context.Context) {
	newShortStatus := b.aggregateStatus()

	// If status has changed, update the last status and emit notifications
	if b.shouldUpdateStatus(newShortStatus) {
		b.updateStatus(newShortStatus)
		b.emitBlockchainHealthStatus(ctx)
	}
}

// aggregateStatus aggregates provider statuses and returns the new short status.
func (b *BlockchainHealthManager) aggregateStatus() BlockchainStatus {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Collect statuses from all providers
	providerStatuses := make([]rpcstatus.ProviderStatus, 0)
	for _, provider := range b.providers {
		providerStatuses = append(providerStatuses, provider.Status())
	}

	// Update the aggregator with the new list of provider statuses
	b.aggregator.UpdateBatch(providerStatuses)

	// Get the new aggregated full and short status
	return b.getStatusPerChain()
}

// shouldUpdateStatus checks if the status has changed and needs to be updated.
func (b *BlockchainHealthManager) shouldUpdateStatus(newShortStatus BlockchainStatus) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.lastStatus == nil || !compareShortStatus(newShortStatus, *b.lastStatus)
}

// updateStatus updates the last known status with the new status.
func (b *BlockchainHealthManager) updateStatus(newShortStatus BlockchainStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastStatus = &newShortStatus
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
	b.subscriptionManager.Emit(ctx)
}

func (b *BlockchainHealthManager) GetFullStatus() BlockchainFullStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	statusPerChainPerProvider := make(map[uint64]map[string]rpcstatus.ProviderStatus)

	for chainID, phm := range b.providers {
		providerStatuses := phm.GetStatuses()
		statusPerChainPerProvider[chainID] = providerStatuses
	}

	statusPerChain := b.getStatusPerChain()

	return BlockchainFullStatus{
		Status:                    statusPerChain.Status,
		StatusPerChain:            statusPerChain.StatusPerChain,
		StatusPerChainPerProvider: statusPerChainPerProvider,
	}
}

func (b *BlockchainHealthManager) getStatusPerChain() BlockchainStatus {
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

func (b *BlockchainHealthManager) GetStatusPerChain() BlockchainStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.getStatusPerChain()
}

// Status returns the current aggregated status.
func (b *BlockchainHealthManager) Status() rpcstatus.ProviderStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.aggregator.GetAggregatedStatus()
}
