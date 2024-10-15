package healthmanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/status-im/status-go/healthmanager/aggregator"
	"github.com/status-im/status-go/healthmanager/rpcstatus"
)

type ProvidersHealthManager struct {
	mu          sync.RWMutex
	chainID     uint64
	aggregator  *aggregator.Aggregator
	subscribers sync.Map // Use sync.Map for concurrent access to subscribers
	lastStatus  *rpcstatus.ProviderStatus
}

// NewProvidersHealthManager creates a new instance of ProvidersHealthManager with the given chain ID.
func NewProvidersHealthManager(chainID uint64) *ProvidersHealthManager {
	agg := aggregator.NewAggregator(fmt.Sprintf("%d", chainID))

	return &ProvidersHealthManager{
		chainID:    chainID,
		aggregator: agg,
	}
}

// Update processes a batch of provider call statuses, updates the aggregated status, and emits chain status changes if necessary.
func (p *ProvidersHealthManager) Update(ctx context.Context, callStatuses []rpcstatus.RpcProviderCallStatus) {
	p.mu.Lock()

	// Update the aggregator with the new provider statuses
	for _, rpcCallStatus := range callStatuses {
		providerStatus := rpcstatus.NewRpcProviderStatus(rpcCallStatus)
		p.aggregator.Update(providerStatus)
	}

	newStatus := p.aggregator.GetAggregatedStatus()

	shouldEmit := p.lastStatus == nil || p.lastStatus.Status != newStatus.Status
	p.mu.Unlock()

	if !shouldEmit {
		return
	}

	p.emitChainStatus(ctx)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastStatus = &newStatus
}

// GetStatuses returns a copy of the current provider statuses.
func (p *ProvidersHealthManager) GetStatuses() map[string]rpcstatus.ProviderStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.aggregator.GetStatuses()
}

// Subscribe allows providers to receive notifications about changes.
func (p *ProvidersHealthManager) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	p.subscribers.Store(ch, struct{}{})
	return ch
}

// Unsubscribe removes a subscriber from receiving notifications.
func (p *ProvidersHealthManager) Unsubscribe(ch chan struct{}) {
	p.subscribers.Delete(ch)
	close(ch)
}

// UnsubscribeAll removes all subscriber channels.
func (p *ProvidersHealthManager) UnsubscribeAll() {
	p.subscribers.Range(func(key, value interface{}) bool {
		ch := key.(chan struct{})
		close(ch)
		p.subscribers.Delete(key)
		return true
	})
}

// Reset clears all provider statuses and resets the chain status to unknown.
func (p *ProvidersHealthManager) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.aggregator = aggregator.NewAggregator(fmt.Sprintf("%d", p.chainID))
}

// Status Returns the current aggregated status.
func (p *ProvidersHealthManager) Status() rpcstatus.ProviderStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.aggregator.GetAggregatedStatus()
}

// ChainID returns the ID of the chain.
func (p *ProvidersHealthManager) ChainID() uint64 {
	return p.chainID
}

// emitChainStatus sends a notification to all subscribers.
func (p *ProvidersHealthManager) emitChainStatus(ctx context.Context) {
	p.subscribers.Range(func(key, value interface{}) bool {
		subscriber := key.(chan struct{})
		select {
		case subscriber <- struct{}{}:
		case <-ctx.Done():
			return false // Stop sending if context is done
		default:
			// Non-blocking send; skip if the channel is full
		}
		return true
	})
}
