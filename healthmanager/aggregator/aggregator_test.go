package aggregator

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/healthmanager/rpcstatus"
)

// StatusAggregatorTestSuite defines the test suite for Aggregator.
type StatusAggregatorTestSuite struct {
	suite.Suite
	aggregator *Aggregator
}

// SetupTest runs before each test in the suite.
func (suite *StatusAggregatorTestSuite) SetupTest() {
	suite.aggregator = NewAggregator("TestAggregator")
}

// TestNewAggregator verifies that a new Aggregator is initialized correctly.
func (suite *StatusAggregatorTestSuite) TestNewAggregator() {
	assert.Equal(suite.T(), "TestAggregator", suite.aggregator.name, "Aggregator name should be set correctly")
	assert.Empty(suite.T(), suite.aggregator.providerStatuses, "Aggregator should have no providers initially")
}

// TestRegisterProvider verifies that providers are registered correctly.
func (suite *StatusAggregatorTestSuite) TestRegisterProvider() {
	providerName := "Provider1"
	suite.aggregator.RegisterProvider(providerName)

	assert.Len(suite.T(), suite.aggregator.providerStatuses, 1, "Expected 1 provider after registration")
	ps, exists := suite.aggregator.providerStatuses[providerName]
	assert.True(suite.T(), exists, "Provider1 should be registered")

	// Verify that the new fields are initialized to zero
	assert.Equal(suite.T(), time.Duration(0), ps.TotalDuration, "TotalDuration should be initialized to zero")
	assert.Equal(suite.T(), int64(0), ps.TotalRequests, "TotalRequests should be initialized to zero")
	assert.Equal(suite.T(), int64(0), ps.TotalTimeoutCount, "TotalTimeoutCount should be initialized to zero")

	// Attempt to register the same provider again
	suite.aggregator.RegisterProvider(providerName)
	assert.Len(suite.T(), suite.aggregator.providerStatuses, 1, "Duplicate registration should not increase provider count")
}

// TestUpdate verifies that updating a provider's status works correctly.
func (suite *StatusAggregatorTestSuite) TestUpdate() {
	providerName := "Provider1"
	suite.aggregator.RegisterProvider(providerName)

	now := time.Now()

	// Update existing provider to up with metrics
	statusUp := rpcstatus.ProviderStatus{
		Name:              providerName,
		Status:            rpcstatus.StatusUp,
		LastSuccessAt:     now,
		TotalDuration:     100 * time.Millisecond,
		TotalRequests:     5,
		TotalTimeoutCount: 1,
	}
	suite.aggregator.Update(statusUp)

	ps, exists := suite.aggregator.providerStatuses[providerName]
	assert.True(suite.T(), exists, "Provider1 should exist after update")
	assert.Equal(suite.T(), rpcstatus.StatusUp, ps.Status, "Provider1 status should be 'up'")
	assert.Equal(suite.T(), now, ps.LastSuccessAt, "Provider1 LastSuccessAt should be updated")
	assert.Equal(suite.T(), 100*time.Millisecond, ps.TotalDuration, "Provider1 TotalDuration should be updated")
	assert.Equal(suite.T(), int64(5), ps.TotalRequests, "Provider1 TotalRequests should be updated")
	assert.Equal(suite.T(), int64(1), ps.TotalTimeoutCount, "Provider1 TotalTimeoutCount should be updated")

	// Update existing provider with additional metrics
	statusUpdate := rpcstatus.ProviderStatus{
		Name:              providerName,
		Status:            rpcstatus.StatusUp,
		LastSuccessAt:     now.Add(1 * time.Hour),
		TotalDuration:     50 * time.Millisecond,
		TotalRequests:     3,
		TotalTimeoutCount: 0,
	}
	suite.aggregator.Update(statusUpdate)

	ps, exists = suite.aggregator.providerStatuses[providerName]
	assert.True(suite.T(), exists, "Provider1 should exist after second update")
	assert.Equal(suite.T(), rpcstatus.StatusUp, ps.Status, "Provider1 status should be 'up'")
	assert.Equal(suite.T(), now.Add(1*time.Hour), ps.LastSuccessAt, "Provider1 LastSuccessAt should be updated")
	assert.Equal(suite.T(), 150*time.Millisecond, ps.TotalDuration, "Provider1 TotalDuration should be accumulated")
	assert.Equal(suite.T(), int64(8), ps.TotalRequests, "Provider1 TotalRequests should be accumulated")
	assert.Equal(suite.T(), int64(1), ps.TotalTimeoutCount, "Provider1 TotalTimeoutCount should be accumulated")

	// Update existing provider to down
	nowDown := now.Add(2 * time.Hour)
	statusDown := rpcstatus.ProviderStatus{
		Name:              providerName,
		Status:            rpcstatus.StatusDown,
		LastErrorAt:       nowDown,
		TotalDuration:     25 * time.Millisecond,
		TotalRequests:     1,
		TotalTimeoutCount: 1,
	}
	suite.aggregator.Update(statusDown)

	ps, exists = suite.aggregator.providerStatuses[providerName]
	assert.True(suite.T(), exists, "Provider1 should exist after third update")
	assert.Equal(suite.T(), rpcstatus.StatusDown, ps.Status, "Provider1 status should be 'down'")
	assert.Equal(suite.T(), nowDown, ps.LastErrorAt, "Provider1 LastErrorAt should be updated")
	assert.Equal(suite.T(), 175*time.Millisecond, ps.TotalDuration, "Provider1 TotalDuration should be accumulated")
	assert.Equal(suite.T(), int64(9), ps.TotalRequests, "Provider1 TotalRequests should be accumulated")
	assert.Equal(suite.T(), int64(2), ps.TotalTimeoutCount, "Provider1 TotalTimeoutCount should be accumulated")

	// Update a non-registered provider via Update (should add it)
	provider2 := "Provider2"
	statusUp2 := rpcstatus.ProviderStatus{
		Name:              provider2,
		Status:            rpcstatus.StatusUp,
		LastSuccessAt:     now,
		TotalDuration:     75 * time.Millisecond,
		TotalRequests:     2,
		TotalTimeoutCount: 0,
	}
	suite.aggregator.Update(statusUp2)

	assert.Len(suite.T(), suite.aggregator.providerStatuses, 2, "Expected 2 providers after updating a new provider")
	ps2, exists := suite.aggregator.providerStatuses[provider2]
	assert.True(suite.T(), exists, "Provider2 should be added via Update")
	assert.Equal(suite.T(), rpcstatus.StatusUp, ps2.Status, "Provider2 status should be 'up'")
	assert.Equal(suite.T(), 75*time.Millisecond, ps2.TotalDuration, "Provider2 TotalDuration should be set")
	assert.Equal(suite.T(), int64(2), ps2.TotalRequests, "Provider2 TotalRequests should be set")
	assert.Equal(suite.T(), int64(0), ps2.TotalTimeoutCount, "Provider2 TotalTimeoutCount should be set")
}

// TestComputeAggregatedStatus_NoProviders verifies aggregated status when no providers are registered.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_NoProviders() {
	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusDown, aggStatus.Status, "Aggregated status should be 'down' when no providers are registered")
	assert.True(suite.T(), aggStatus.LastSuccessAt.IsZero(), "LastSuccessAt should be zero when no providers are registered")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when no providers are registered")
}

// TestComputeAggregatedStatus_AllUnknown verifies aggregated status when all providers are unknown.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_AllUnknown() {
	// Register multiple providers with unknown status
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")
	suite.aggregator.RegisterProvider("Provider3")

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUnknown, aggStatus.Status, "Aggregated status should be 'unknown' when all providers are unknown")
	assert.True(suite.T(), aggStatus.LastSuccessAt.IsZero(), "LastSuccessAt should be zero when all providers are unknown")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when all providers are unknown")
}

// TestComputeAggregatedStatus_AllUp verifies aggregated status when all providers are up.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_AllUp() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")

	now1 := time.Now()
	now2 := now1.Add(1 * time.Hour)

	// Update all providers to up with metrics
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:              "Provider1",
		Status:            rpcstatus.StatusUp,
		LastSuccessAt:     now1,
		TotalDuration:     100 * time.Millisecond,
		TotalRequests:     5,
		TotalTimeoutCount: 1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:              "Provider2",
		Status:            rpcstatus.StatusUp,
		LastSuccessAt:     now2,
		TotalDuration:     200 * time.Millisecond,
		TotalRequests:     10,
		TotalTimeoutCount: 2,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when all providers are up")
	assert.Equal(suite.T(), now2, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the latest success time")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when all providers are up")

	// Verify aggregated metrics
	assert.Equal(suite.T(), 300*time.Millisecond, aggStatus.TotalDuration, "TotalDuration should be the sum of all providers")
	assert.Equal(suite.T(), int64(15), aggStatus.TotalRequests, "TotalRequests should be the sum of all providers")
	assert.Equal(suite.T(), int64(3), aggStatus.TotalTimeoutCount, "TotalTimeoutCount should be the sum of all providers")
}

// TestComputeAggregatedStatus_AllDown verifies aggregated status when all providers are down.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_AllDown() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")

	now1 := time.Now()
	now2 := now1.Add(1 * time.Hour)

	// Update all providers to down
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:        "Provider1",
		Status:      rpcstatus.StatusDown,
		LastErrorAt: now1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:        "Provider2",
		Status:      rpcstatus.StatusDown,
		LastErrorAt: now2,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusDown, aggStatus.Status, "Aggregated status should be 'down' when all providers are down")
	assert.Equal(suite.T(), now2, aggStatus.LastErrorAt, "LastErrorAt should reflect the latest error time")
	assert.True(suite.T(), aggStatus.LastSuccessAt.IsZero(), "LastSuccessAt should be zero when all providers are down")
}

// TestComputeAggregatedStatus_MixedUpAndUnknown verifies aggregated status with mixed up and unknown providers.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_MixedUpAndUnknown() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1") // up
	suite.aggregator.RegisterProvider("Provider2") // unknown
	suite.aggregator.RegisterProvider("Provider3") // up

	now1 := time.Now()
	now2 := now1.Add(30 * time.Minute)

	// Update some providers to up
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider1",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider3",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now2,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when at least one provider is up")
	assert.Equal(suite.T(), now2, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the latest success time")
	assert.True(suite.T(), aggStatus.LastErrorAt.IsZero(), "LastErrorAt should be zero when no providers are down")
}

// TestComputeAggregatedStatus_MixedUpAndDown verifies aggregated status with mixed up and down providers.
func (suite *StatusAggregatorTestSuite) TestComputeAggregatedStatus_MixedUpAndDown() {
	// Register providers
	suite.aggregator.RegisterProvider("Provider1") // up
	suite.aggregator.RegisterProvider("Provider2") // down
	suite.aggregator.RegisterProvider("Provider3") // up

	now1 := time.Now()
	now2 := now1.Add(15 * time.Minute)

	// Update providers with metrics
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:              "Provider1",
		Status:            rpcstatus.StatusUp,
		LastSuccessAt:     now1,
		TotalDuration:     100 * time.Millisecond,
		TotalRequests:     5,
		TotalTimeoutCount: 1,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:              "Provider2",
		Status:            rpcstatus.StatusDown,
		LastErrorAt:       now2,
		TotalDuration:     200 * time.Millisecond,
		TotalRequests:     10,
		TotalTimeoutCount: 2,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:              "Provider3",
		Status:            rpcstatus.StatusUp,
		LastSuccessAt:     now1,
		TotalDuration:     300 * time.Millisecond,
		TotalRequests:     15,
		TotalTimeoutCount: 3,
	})

	aggStatus := suite.aggregator.ComputeAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when at least one provider is up")
	assert.Equal(suite.T(), now1, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the latest success time")
	assert.Equal(suite.T(), now2, aggStatus.LastErrorAt, "LastErrorAt should reflect the latest error time")

	// Verify aggregated metrics
	assert.Equal(suite.T(), 600*time.Millisecond, aggStatus.TotalDuration, "TotalDuration should be the sum of all providers")
	assert.Equal(suite.T(), int64(30), aggStatus.TotalRequests, "TotalRequests should be the sum of all providers")
	assert.Equal(suite.T(), int64(6), aggStatus.TotalTimeoutCount, "TotalTimeoutCount should be the sum of all providers")
}

// TestGetAggregatedStatus verifies that GetAggregatedStatus returns the correct aggregated status.
func (suite *StatusAggregatorTestSuite) TestGetAggregatedStatus() {
	// Register and update providers
	suite.aggregator.RegisterProvider("Provider1")
	suite.aggregator.RegisterProvider("Provider2")

	now := time.Now()

	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:          "Provider1",
		Status:        rpcstatus.StatusUp,
		LastSuccessAt: now,
	})
	suite.aggregator.Update(rpcstatus.ProviderStatus{
		Name:        "Provider2",
		Status:      rpcstatus.StatusDown,
		LastErrorAt: now.Add(1 * time.Hour),
	})

	aggStatus := suite.aggregator.GetAggregatedStatus()

	assert.Equal(suite.T(), rpcstatus.StatusUp, aggStatus.Status, "Aggregated status should be 'up' when at least one provider is up")
	assert.Equal(suite.T(), now, aggStatus.LastSuccessAt, "LastSuccessAt should reflect the provider's success time")
	assert.Equal(suite.T(), now.Add(1*time.Hour), aggStatus.LastErrorAt, "LastErrorAt should reflect the provider's error time")
}

// TestConcurrentAccess verifies that the Aggregator is safe for concurrent use.
func (suite *StatusAggregatorTestSuite) TestConcurrentAccess() {
	// Register multiple providers
	providers := []string{"Provider1", "Provider2", "Provider3", "Provider4", "Provider5"}
	for _, p := range providers {
		suite.aggregator.RegisterProvider(p)
	}

	var wg sync.WaitGroup

	// Concurrently update providers
	for _, p := range providers {
		wg.Add(1)
		go func(providerName string) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				suite.aggregator.Update(rpcstatus.ProviderStatus{
					Name:          providerName,
					Status:        rpcstatus.StatusUp,
					LastSuccessAt: time.Now(),
				})
				suite.aggregator.Update(rpcstatus.ProviderStatus{
					Name:        providerName,
					Status:      rpcstatus.StatusDown,
					LastErrorAt: time.Now(),
				})
			}
		}(p)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Set all providers to down to ensure deterministic aggregated status
	now := time.Now()
	for _, p := range providers {
		suite.aggregator.Update(rpcstatus.ProviderStatus{
			Name:        p,
			Status:      rpcstatus.StatusDown,
			LastErrorAt: now,
		})
	}

	aggStatus := suite.aggregator.GetAggregatedStatus()
	assert.Equal(suite.T(), rpcstatus.StatusDown, aggStatus.Status, "Aggregated status should be 'down' after setting all providers to down")
}

// TestStatusAggregatorTestSuite runs the test suite.
func TestStatusAggregatorTestSuite(t *testing.T) {
	suite.Run(t, new(StatusAggregatorTestSuite))
}
