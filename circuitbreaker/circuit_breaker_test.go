package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/healthmanager/provider_errors"
)

const success = "Success"

func TestCircuitBreaker_ExecuteSuccessSingle(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Timeout:                1000,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 10,
		SleepWindow:            10,
		ErrorPercentThreshold:  10,
	})

	expectedResult := success
	circuitName := "SuccessSingle"
	cmd := NewCommand(context.TODO(), []*Functor{
		NewFunctor(func() ([]interface{}, error) {
			return []any{expectedResult}, nil
		}, circuitName, "")},
	)

	result := cb.Execute(cmd)
	require.NoError(t, result.Error())
	require.Equal(t, expectedResult, result.Result()[0].(string))
	require.False(t, result.Cancelled())
}

func TestCircuitBreaker_ExecuteMultipleFallbacksFail(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Timeout:                10,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 10,
		SleepWindow:            10,
		ErrorPercentThreshold:  10,
	})

	circuitName := fmt.Sprintf("ExecuteMultipleFallbacksFail_%d", time.Now().Nanosecond()) // unique name to avoid conflicts with go tests `-count` option
	errSecProvFailed := errors.New("provider 2 failed")
	errThirdProvFailed := errors.New("provider 3 failed")
	cmd := NewCommand(context.TODO(), []*Functor{
		NewFunctor(func() ([]interface{}, error) {
			time.Sleep(100 * time.Millisecond) // will cause hystrix: timeout
			return []any{success}, nil
		}, circuitName+"1", ""),
		NewFunctor(func() ([]interface{}, error) {
			return nil, errSecProvFailed
		}, circuitName+"2", ""),
		NewFunctor(func() ([]interface{}, error) {
			return nil, errThirdProvFailed
		}, circuitName+"3", ""),
	})

	result := cb.Execute(cmd)
	require.Error(t, result.Error())
	assert.True(t, errors.Is(result.Error(), hystrix.ErrTimeout))
	assert.True(t, errors.Is(result.Error(), errSecProvFailed))
	assert.True(t, errors.Is(result.Error(), errThirdProvFailed))
}

func TestCircuitBreaker_ExecuteMultipleFallbacksFailButLastSuccessStress(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Timeout:                10,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 10,
		SleepWindow:            10,
		ErrorPercentThreshold:  10,
	})

	expectedResult := success
	circuitName := fmt.Sprintf("LastSuccessStress_%d", time.Now().Nanosecond()) // unique name to avoid conflicts with go tests `-count` option

	// These are executed sequentially, but I had an issue with the test failing
	// because of the open circuit
	for i := 0; i < 1000; i++ {
		cmd := NewCommand(context.TODO(), []*Functor{
			NewFunctor(func() ([]interface{}, error) {
				return nil, errors.New("provider 1 failed")
			}, circuitName+"1", ""),
			NewFunctor(func() ([]interface{}, error) {
				return nil, errors.New("provider 2 failed")
			}, circuitName+"2", ""),
			NewFunctor(func() ([]interface{}, error) {
				return []any{expectedResult}, nil
			}, circuitName+"3", ""),
		},
		)

		result := cb.Execute(cmd)
		require.NoError(t, result.Error())
		require.Equal(t, expectedResult, result.Result()[0].(string))
	}
}

func TestCircuitBreaker_ExecuteSwitchToWorkingProviderOnVolumeThresholdReached(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		RequestVolumeThreshold: 10,
	})

	expectedResult := success
	circuitName := fmt.Sprintf("SwitchToWorkingProviderOnVolumeThresholdReached_%d", time.Now().Nanosecond()) // unique name to avoid conflicts with go tests `-count` option

	prov1Called := 0
	prov2Called := 0
	prov3Called := 0
	// These are executed sequentially
	for i := 0; i < 20; i++ {
		cmd := NewCommand(context.TODO(), []*Functor{
			NewFunctor(func() ([]interface{}, error) {
				prov1Called++
				return nil, errors.New("provider 1 failed")
			}, circuitName+"1", ""),
			NewFunctor(func() ([]interface{}, error) {
				prov2Called++
				return nil, errors.New("provider 2 failed")
			}, circuitName+"2", ""),
			NewFunctor(func() ([]interface{}, error) {
				prov3Called++
				return []any{expectedResult}, nil
			}, circuitName+"3", ""),
		})

		result := cb.Execute(cmd)
		require.NoError(t, result.Error())
		require.Equal(t, expectedResult, result.Result()[0].(string))
	}

	assert.Equal(t, 10, prov1Called)
	assert.Equal(t, 10, prov2Called)
	assert.Equal(t, 20, prov3Called)
}

func TestCircuitBreaker_ExecuteHealthCheckOnWindowTimeout(t *testing.T) {
	sleepWindow := 10
	cb := NewCircuitBreaker(Config{
		RequestVolumeThreshold: 1, // 1 failed request is enough to trip the circuit
		SleepWindow:            sleepWindow,
		ErrorPercentThreshold:  1, // Trip on first error
	})

	expectedResult := success
	circuitName := fmt.Sprintf("SwitchToWorkingProviderOnWindowTimeout_%d", time.Now().Nanosecond()) // unique name to avoid conflicts with go tests `-count` option

	prov1Called := 0
	prov2Called := 0
	// These are executed sequentially
	for i := 0; i < 10; i++ {
		cmd := NewCommand(context.TODO(), []*Functor{
			NewFunctor(func() ([]interface{}, error) {
				prov1Called++
				return nil, errors.New("provider 1 failed")
			}, circuitName+"1", ""),
			NewFunctor(func() ([]interface{}, error) {
				prov2Called++
				return []any{expectedResult}, nil
			}, circuitName+"2", ""),
		})

		result := cb.Execute(cmd)
		require.NoError(t, result.Error())
		require.Equal(t, expectedResult, result.Result()[0].(string))
	}

	assert.Less(t, prov1Called, 3) // most of the time only 1 call is made, but occasionally 2 can happen
	assert.Equal(t, 10, prov2Called)
	assert.True(t, CircuitExists(circuitName+"1"))
	assert.True(t, IsCircuitOpen(circuitName+"1"))

	// Wait for the sleep window to expire
	time.Sleep(time.Duration(sleepWindow+1) * time.Millisecond)
	cmd := NewCommand(context.TODO(), []*Functor{
		NewFunctor(func() ([]interface{}, error) {
			prov1Called++
			return []any{expectedResult}, nil // Now it is working
		}, circuitName+"1", ""),
		NewFunctor(func() ([]interface{}, error) {
			prov2Called++
			return []any{expectedResult}, nil
		}, circuitName+"2", ""),
	})
	result := cb.Execute(cmd)
	require.NoError(t, result.Error())

	assert.Less(t, prov1Called, 4) // most of the time only 2 calls are made, but occasionally 3 can happen
	assert.Equal(t, 10, prov2Called)
}

func TestCircuitBreaker_CommandCancel(t *testing.T) {
	cb := NewCircuitBreaker(Config{})

	circuitName := fmt.Sprintf("CommandCancel_%d", time.Now().Nanosecond()) // unique name to avoid conflicts with go tests `-count` option

	prov1Called := 0
	prov2Called := 0

	var ctx context.Context
	expectedErr := errors.New("provider 1 failed")

	cmd := NewCommand(ctx, nil)
	cmd.Add(NewFunctor(func() ([]interface{}, error) {
		prov1Called++
		cmd.Cancel()
		return nil, expectedErr
	}, circuitName+"1", ""))
	cmd.Add(NewFunctor(func() ([]interface{}, error) {
		prov2Called++
		return nil, errors.New("provider 2 failed")
	}, circuitName+"2", ""))

	result := cb.Execute(cmd)
	require.True(t, errors.Is(result.Error(), expectedErr))
	require.True(t, result.Cancelled())

	assert.Equal(t, 1, prov1Called)
	assert.Equal(t, 0, prov2Called)
}

func TestCircuitBreaker_EmptyOrNilCommand(t *testing.T) {
	cb := NewCircuitBreaker(Config{})
	cmd := NewCommand(context.TODO(), nil)
	result := cb.Execute(cmd)
	require.Error(t, result.Error())
	result = cb.Execute(nil)
	require.Error(t, result.Error())
}

func TestCircuitBreaker_CircuitExistsAndClosed(t *testing.T) {
	timestamp := time.Now().Nanosecond()
	nonExCircuit := fmt.Sprintf("nonexistent_%d", timestamp) // unique name to avoid conflicts with go tests `-count` option
	require.False(t, CircuitExists(nonExCircuit))

	cb := NewCircuitBreaker(Config{})
	cmd := NewCommand(context.TODO(), nil)
	existCircuit := fmt.Sprintf("existing_%d", timestamp) // unique name to avoid conflicts with go tests `-count` option
	// We add it twice as otherwise it's only used for the fallback
	cmd.Add(NewFunctor(func() ([]interface{}, error) {
		return nil, nil
	}, existCircuit, ""))

	cmd.Add(NewFunctor(func() ([]interface{}, error) {
		return nil, nil
	}, existCircuit, ""))
	_ = cb.Execute(cmd)
	require.True(t, CircuitExists(existCircuit))
	require.False(t, IsCircuitOpen(existCircuit))
}

func TestCircuitBreaker_Fallback(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		RequestVolumeThreshold: 1, // 1 failed request is enough to trip the circuit
		SleepWindow:            50000,
		ErrorPercentThreshold:  1, // Trip on first error
	})

	circuitName := fmt.Sprintf("Fallback_%d", time.Now().Nanosecond()) // unique name to avoid conflicts with go tests `-count` option

	prov1Called := 0

	var ctx context.Context
	expectedErr := errors.New("provider 1 failed")

	// we start with 2, and we open the first
	for {
		cmd := NewCommand(ctx, nil)
		cmd.Add(NewFunctor(func() ([]interface{}, error) {
			return nil, expectedErr
		}, circuitName+"1", ""))
		cmd.Add(NewFunctor(func() ([]interface{}, error) {
			return nil, errors.New("provider 2 failed")
		}, circuitName+"2", ""))

		result := cb.Execute(cmd)
		require.NotNil(t, result.Error())
		if IsCircuitOpen(circuitName + "1") {
			break
		}
	}

	// Make sure circuit is open
	require.True(t, CircuitExists(circuitName+"1"))
	require.True(t, IsCircuitOpen(circuitName+"1"))

	// we send a single request, it should hit the provider, at that's a fallback
	cmd := NewCommand(ctx, nil)
	cmd.Add(NewFunctor(func() ([]interface{}, error) {
		prov1Called++
		return nil, expectedErr
	}, circuitName+"1", ""))

	result := cb.Execute(cmd)
	require.True(t, errors.Is(result.Error(), expectedErr))

	assert.Equal(t, 1, prov1Called)
}

func TestCircuitBreaker_SuccessCallStatus(t *testing.T) {
	cb := NewCircuitBreaker(Config{})

	functor := NewFunctor(func() ([]any, error) {
		return []any{"success"}, nil
	}, "successCircuit", "successProvider")

	cmd := NewCommand(context.Background(), []*Functor{functor})

	result := cb.Execute(cmd)

	require.Nil(t, result.Error())
	require.False(t, result.Cancelled())
	assert.Len(t, result.Result(), 1)
	require.Equal(t, "success", result.Result()[0])
	assert.Len(t, result.FunctorCallStatuses(), 1)

	status := result.FunctorCallStatuses()[0]
	if status.Name != "successProvider" {
		t.Errorf("Expected functor name to be 'successProvider', got %s", status.Name)
	}
	if status.Err != nil {
		t.Errorf("Expected no error in functor status, got %v", status.Err)
	}
}

func TestCircuitBreaker_ErrorCallStatus(t *testing.T) {
	cb := NewCircuitBreaker(Config{})

	expectedError := errors.New("functor error")
	functor := NewFunctor(func() ([]any, error) {
		return nil, expectedError
	}, "errorCircuit", "errorProvider")

	cmd := NewCommand(context.Background(), []*Functor{functor})

	result := cb.Execute(cmd)

	require.NotNil(t, result.Error())
	require.True(t, errors.Is(result.Error(), expectedError))

	assert.Len(t, result.Result(), 0)
	assert.Len(t, result.FunctorCallStatuses(), 1)

	status := result.FunctorCallStatuses()[0]
	if status.Name != "errorProvider" {
		t.Errorf("Expected functor name to be 'errorProvider', got %s", status.Name)
	}
	if !errors.Is(status.Err, expectedError) {
		t.Errorf("Expected functor error to be '%v', got '%v'", expectedError, status.Err)
	}
}

func TestCircuitBreaker_CancelledResult(t *testing.T) {
	cb := NewCircuitBreaker(Config{Timeout: 1000})

	functor := NewFunctor(func() ([]any, error) {
		time.Sleep(500 * time.Millisecond)
		return []any{"should not be returned"}, nil
	}, "cancelCircuit", "")

	cmd := NewCommand(context.Background(), []*Functor{functor})
	cmd.Cancel()

	result := cb.Execute(cmd)

	assert.True(t, result.Cancelled())
	require.Nil(t, result.Error())
	require.Empty(t, result.Result())
	require.Empty(t, result.FunctorCallStatuses())
}

func TestCircuitBreaker_MultipleFunctorsResult(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Timeout:                1000,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 20,
		SleepWindow:            5000,
		ErrorPercentThreshold:  50,
	})

	functor1 := NewFunctor(func() ([]any, error) {
		return nil, errors.New("functor1 error")
	}, "circuit1", "provider1")

	functor2 := NewFunctor(func() ([]any, error) {
		return []any{"success from functor2"}, nil
	}, "circuit2", "provider2")

	cmd := NewCommand(context.Background(), []*Functor{functor1, functor2})

	result := cb.Execute(cmd)

	require.Nil(t, result.Error())

	require.Len(t, result.Result(), 1)
	require.Equal(t, result.Result()[0], "success from functor2")
	statuses := result.FunctorCallStatuses()
	require.Len(t, statuses, 2)

	require.Equal(t, statuses[0].Name, "provider1")
	require.NotNil(t, statuses[0].Err)

	require.Equal(t, statuses[1].Name, "provider2")
	require.Nil(t, statuses[1].Err)
}

func TestCircuitBreaker_LastFunctorDirectExecution(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Timeout:                10, // short timeout to open circuit
		MaxConcurrentRequests:  1,
		RequestVolumeThreshold: 1,
		SleepWindow:            1000,
		ErrorPercentThreshold:  1,
	})

	failingFunctor := NewFunctor(func() ([]any, error) {
		time.Sleep(20 * time.Millisecond)
		return nil, errors.New("should time out")
	}, "circuitName", "providerName")

	successFunctor := NewFunctor(func() ([]any, error) {
		return []any{"success without circuit"}, nil
	}, "circuitName", "providerName")

	cmd := NewCommand(context.Background(), []*Functor{failingFunctor, successFunctor})

	require.False(t, IsCircuitOpen("circuitName"))
	result := cb.Execute(cmd)

	require.True(t, CircuitExists("circuitName"))
	require.Nil(t, result.Error())

	require.Len(t, result.Result(), 1)
	require.Equal(t, result.Result()[0], "success without circuit")

	statuses := result.FunctorCallStatuses()
	require.Len(t, statuses, 2)

	require.Equal(t, statuses[0].Name, "providerName")
	require.NotNil(t, statuses[0].Err)

	require.Equal(t, statuses[1].Name, "providerName")
	require.Nil(t, statuses[1].Err)
}

func TestFunctorCallStatus_NewFields(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Timeout: 100, // Short timeout for testing
	})

	t.Run("Success", func(t *testing.T) {
		// Create a functor
		responseData := []any{"item1", "item2", "item3"}
		functor := NewFunctor(func() ([]any, error) {
			return responseData, nil
		}, "successCircuit", "provider1")

		cmd := NewCommand(context.Background(), []*Functor{functor})
		result := cb.Execute(cmd)

		require.NoError(t, result.Error())
		require.Len(t, result.FunctorCallStatuses(), 1)

		status := result.FunctorCallStatuses()[0]
		assert.Equal(t, "provider1", status.Name)
		assert.NoError(t, status.Err)
		assert.False(t, provider_errors.IsTimeoutErr(status.Err))

		// Verify timing information
		assert.NotZero(t, status.StartTime)
		assert.NotZero(t, status.Timestamp)
		assert.True(t, status.Timestamp.After(status.StartTime) || status.Timestamp.Equal(status.StartTime))
		duration := status.Timestamp.Sub(status.StartTime)
		assert.True(t, duration > 0)
	})

	t.Run("Error with timeout", func(t *testing.T) {
		// Create a test that directly sets the error to context.DeadlineExceeded
		expectedErr := context.DeadlineExceeded
		functor := NewFunctor(func() ([]any, error) {
			return nil, expectedErr
		}, "timeoutCircuit", "provider2")

		cmd := NewCommand(context.Background(), []*Functor{functor})
		result := cb.Execute(cmd)

		require.Error(t, result.Error())
		require.Len(t, result.FunctorCallStatuses(), 1)

		status := result.FunctorCallStatuses()[0]
		assert.Equal(t, "provider2", status.Name)
		assert.Error(t, status.Err)
		assert.True(t, provider_errors.IsTimeoutErr(status.Err))

		// Verify timing information
		assert.NotZero(t, status.StartTime)
		assert.NotZero(t, status.Timestamp)
		duration := status.Timestamp.Sub(status.StartTime)
		assert.True(t, duration >= 0)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Create a test that directly sets the error to context.Canceled
		expectedErr := context.Canceled
		functor := NewFunctor(func() ([]any, error) {
			return nil, expectedErr
		}, "cancellationCircuit", "provider5")

		cmd := NewCommand(context.Background(), []*Functor{functor})
		result := cb.Execute(cmd)

		require.Error(t, result.Error())
		require.Len(t, result.FunctorCallStatuses(), 1)

		status := result.FunctorCallStatuses()[0]
		assert.Equal(t, "provider5", status.Name)
		assert.Error(t, status.Err)
		assert.False(t, provider_errors.IsTimeoutErr(status.Err))

		// Verify timing information
		assert.NotZero(t, status.StartTime)
		assert.NotZero(t, status.Timestamp)
		duration := status.Timestamp.Sub(status.StartTime)
		assert.True(t, duration >= 0)
	})

	t.Run("Multiple functors", func(t *testing.T) {
		// First functor fails
		functor1 := NewFunctor(func() ([]any, error) {
			return nil, errors.New("first functor error")
		}, "multiCircuit1", "provider3")

		// Second functor succeeds with response
		responseData := []any{"response1", "response2"}
		functor2 := NewFunctor(func() ([]any, error) {
			return responseData, nil
		}, "multiCircuit2", "provider4")

		cmd := NewCommand(context.Background(), []*Functor{functor1, functor2})
		result := cb.Execute(cmd)

		require.NoError(t, result.Error())
		require.Len(t, result.FunctorCallStatuses(), 2)

		// Check first status (error)
		status1 := result.FunctorCallStatuses()[0]
		assert.Equal(t, "provider3", status1.Name)
		assert.Error(t, status1.Err)
		assert.False(t, provider_errors.IsTimeoutErr(status1.Err))

		// Check second status (success)
		status2 := result.FunctorCallStatuses()[1]
		assert.Equal(t, "provider4", status2.Name)
		assert.NoError(t, status2.Err)
		assert.False(t, provider_errors.IsTimeoutErr(status2.Err))
	})
}

func TestCircuitBreaker_ErrorLogging(t *testing.T) {
	// We can't easily capture the log output, so we'll just verify that the code runs without errors
	cb := NewCircuitBreaker(Config{})

	// Test direct execution error logging
	t.Run("Direct execution error logging", func(t *testing.T) {
		expectedError := errors.New("direct test error")
		functor := NewFunctor(func() ([]any, error) {
			return nil, expectedError
		}, "", "directProvider")

		cmd := NewCommand(context.Background(), []*Functor{functor})
		result := cb.Execute(cmd)

		require.Error(t, result.Error())
		require.Len(t, result.FunctorCallStatuses(), 1)

		// Verify the status fields are set correctly
		status := result.FunctorCallStatuses()[0]
		assert.Equal(t, "directProvider", status.Name)
		assert.Equal(t, expectedError, status.Err)
		assert.False(t, provider_errors.IsTimeoutErr(status.Err))

		// Verify timing information
		assert.NotZero(t, status.StartTime)
		assert.NotZero(t, status.Timestamp)
		duration := status.Timestamp.Sub(status.StartTime)
		assert.True(t, duration >= 0)
	})

	// Test hystrix error logging
	t.Run("Hystrix error logging", func(t *testing.T) {
		expectedError := errors.New("hystrix test error")
		functor := NewFunctor(func() ([]any, error) {
			return nil, expectedError
		}, "hystrixCircuit", "hystrixProvider")

		cmd := NewCommand(context.Background(), []*Functor{functor})
		result := cb.Execute(cmd)

		require.Error(t, result.Error())
		require.Len(t, result.FunctorCallStatuses(), 1)

		// Verify the status fields are set correctly
		status := result.FunctorCallStatuses()[0]
		assert.Equal(t, "hystrixProvider", status.Name)
		assert.Equal(t, expectedError, status.Err)
		assert.False(t, provider_errors.IsTimeoutErr(status.Err))

		// Verify timing information
		assert.NotZero(t, status.StartTime)
		assert.NotZero(t, status.Timestamp)
		duration := status.Timestamp.Sub(status.StartTime)
		assert.True(t, duration >= 0)
	})

	// Test timeout error logging
	t.Run("Timeout error logging", func(t *testing.T) {
		expectedError := context.DeadlineExceeded
		functor := NewFunctor(func() ([]any, error) {
			return nil, expectedError
		}, "timeoutCircuit", "timeoutProvider")

		cmd := NewCommand(context.Background(), []*Functor{functor})
		result := cb.Execute(cmd)

		require.Error(t, result.Error())
		require.Len(t, result.FunctorCallStatuses(), 1)

		// Verify the status fields are set correctly
		status := result.FunctorCallStatuses()[0]
		assert.Equal(t, "timeoutProvider", status.Name)
		assert.Equal(t, expectedError, status.Err)
		assert.True(t, provider_errors.IsTimeoutErr(status.Err))

		// Verify timing information
		assert.NotZero(t, status.StartTime)
		assert.NotZero(t, status.Timestamp)
		duration := status.Timestamp.Sub(status.StartTime)
		assert.True(t, duration >= 0)
	})
}
