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
	subscribers []chan struct{}
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
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan struct{}, 1)
	p.subscribers = append(p.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber from receiving notifications.
func (p *ProvidersHealthManager) Unsubscribe(ch chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, subscriber := range p.subscribers {
		if subscriber == ch {
			p.subscribers = append(p.subscribers[:i], p.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// UnsubscribeAll removes all subscriber channels.
func (p *ProvidersHealthManager) UnsubscribeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, subscriber := range p.subscribers {
		close(subscriber)
	}
	p.subscribers = nil
}

// Reset clears all provider statuses and resets the chain status to unknown.
func (p *ProvidersHealthManager) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.aggregator = aggregator.NewAggregator(fmt.Sprintf("%d", p.chainID))
}

// Status Returns the current aggregated status
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
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, subscriber := range p.subscribers {
		select {
		case subscriber <- struct{}{}:
		case <-ctx.Done():
			return
		default:
			// Non-blocking send; skip if the channel is full
		}
	}
}
