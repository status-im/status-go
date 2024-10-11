package aggregator

import (
	"sync"
	"time"

	"github.com/status-im/status-go/healthmanager/rpcstatus"
)

// Aggregator manages and aggregates the statuses of multiple providers.
type Aggregator struct {
	mu               sync.RWMutex
	name             string
	providerStatuses map[string]*rpcstatus.ProviderStatus
}

// NewAggregator creates a new instance of Aggregator with the given name.
func NewAggregator(name string) *Aggregator {
	return &Aggregator{
		name:             name,
		providerStatuses: make(map[string]*rpcstatus.ProviderStatus),
	}
}

// RegisterProvider adds a new provider to the aggregator.
// If the provider already exists, it does nothing.
func (a *Aggregator) RegisterProvider(providerName string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.providerStatuses[providerName]; !exists {
		a.providerStatuses[providerName] = &rpcstatus.ProviderStatus{
			Name:   providerName,
			Status: rpcstatus.StatusUnknown,
		}
	}
}

// Update modifies the status of a specific provider.
// If the provider is not already registered, it adds the provider.
func (a *Aggregator) Update(providerStatus rpcstatus.ProviderStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update existing provider status or add a new provider.
	if ps, exists := a.providerStatuses[providerStatus.Name]; exists {
		ps.Status = providerStatus.Status
		if providerStatus.Status == rpcstatus.StatusUp {
			ps.LastSuccessAt = providerStatus.LastSuccessAt
		} else if providerStatus.Status == rpcstatus.StatusDown {
			ps.LastErrorAt = providerStatus.LastErrorAt
			ps.LastError = providerStatus.LastError
		}
	} else {
		a.providerStatuses[providerStatus.Name] = &rpcstatus.ProviderStatus{
			Name:          providerStatus.Name,
			LastSuccessAt: providerStatus.LastSuccessAt,
			LastErrorAt:   providerStatus.LastErrorAt,
			LastError:     providerStatus.LastError,
			Status:        providerStatus.Status,
		}
	}
}

// UpdateBatch processes a batch of provider statuses.
func (a *Aggregator) UpdateBatch(statuses []rpcstatus.ProviderStatus) {
	for _, status := range statuses {
		a.Update(status)
	}
}

// ComputeAggregatedStatus calculates the overall aggregated status based on individual provider statuses.
// The logic is as follows:
// - If any provider is up, the aggregated status is up.
// - If no providers are up but at least one is unknown, the aggregated status is unknown.
// - If all providers are down, the aggregated status is down.
func (a *Aggregator) ComputeAggregatedStatus() rpcstatus.ProviderStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var lastSuccessAt, lastErrorAt time.Time
	var lastError error
	anyUp := false
	anyUnknown := false

	for _, ps := range a.providerStatuses {
		switch ps.Status {
		case rpcstatus.StatusUp:
			anyUp = true
			if ps.LastSuccessAt.After(lastSuccessAt) {
				lastSuccessAt = ps.LastSuccessAt
			}
		case rpcstatus.StatusUnknown:
			anyUnknown = true
		case rpcstatus.StatusDown:
			if ps.LastErrorAt.After(lastErrorAt) {
				lastErrorAt = ps.LastErrorAt
				lastError = ps.LastError
			}
		}
	}

	aggregatedStatus := rpcstatus.ProviderStatus{
		Name:          a.name,
		LastSuccessAt: lastSuccessAt,
		LastErrorAt:   lastErrorAt,
		LastError:     lastError,
	}
	if len(a.providerStatuses) == 0 {
		aggregatedStatus.Status = rpcstatus.StatusDown
	} else if anyUp {
		aggregatedStatus.Status = rpcstatus.StatusUp
	} else if anyUnknown {
		aggregatedStatus.Status = rpcstatus.StatusUnknown
	} else {
		aggregatedStatus.Status = rpcstatus.StatusDown
	}

	return aggregatedStatus
}

func (a *Aggregator) GetAggregatedStatus() rpcstatus.ProviderStatus {
	return a.ComputeAggregatedStatus()
}

func (a *Aggregator) GetStatuses() map[string]rpcstatus.ProviderStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	statusesCopy := make(map[string]rpcstatus.ProviderStatus)
	for k, v := range a.providerStatuses {
		statusesCopy[k] = *v
	}
	return statusesCopy
}
