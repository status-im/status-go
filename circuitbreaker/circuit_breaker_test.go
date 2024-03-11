package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const success = "Success"

func TestCircuitBreaker_ExecuteSuccessSingle(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		CommandName:            "SuccessSingle", // unique name to avoid conflicts with go tests `-count` option
		Timeout:                1000,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 10,
		SleepWindow:            10,
		ErrorPercentThreshold:  10,
	})

	expectedResult := success
	cmd := Command{
		functors: []*Functor{
			NewFunctor(func() ([]interface{}, error) {
				return []any{expectedResult}, nil
			}),
		},
	}

	result := cb.Execute(cmd)
	require.NoError(t, result.Error())
	require.Equal(t, expectedResult, result.Result()[0].(string))
}

func TestCircuitBreaker_ExecuteMultipleFallbacksFail(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		CommandName:            "MultipleFail", // unique name to avoid conflicts with go tests `-count` option
		Timeout:                10,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 10,
		SleepWindow:            10,
		ErrorPercentThreshold:  10,
	})

	cmd := Command{
		functors: []*Functor{
			NewFunctor(func() ([]interface{}, error) {
				time.Sleep(100 * time.Millisecond) // will cause hystrix: timeout
				return []any{success}, nil
			}),
			NewFunctor(func() ([]interface{}, error) {
				return nil, errors.New("provider 2 failed")
			}),
			NewFunctor(func() ([]interface{}, error) {
				return nil, errors.New("provider 3 failed")
			}),
		},
	}

	result := cb.Execute(cmd)
	require.Error(t, result.Error())
}

func TestCircuitBreaker_ExecuteMultipleFallbacksFailButLastSuccessStress(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		CommandName:            "LastSuccessStress", // unique name to avoid conflicts with go tests `-count` option
		Timeout:                10,
		MaxConcurrentRequests:  100,
		RequestVolumeThreshold: 10,
		SleepWindow:            10,
		ErrorPercentThreshold:  10,
	})

	expectedResult := success

	// These are executed sequentially, but I had an issue with the test failing
	// because of the open circuit
	for i := 0; i < 1000; i++ {
		cmd := Command{
			functors: []*Functor{
				NewFunctor(func() ([]interface{}, error) {
					return nil, errors.New("provider 1 failed")
				}),
				NewFunctor(func() ([]interface{}, error) {
					return nil, errors.New("provider 2 failed")
				}),
				NewFunctor(func() ([]interface{}, error) {
					return []any{expectedResult}, nil
				}),
			},
		}

		result := cb.Execute(cmd)
		require.NoError(t, result.Error())
		require.Equal(t, expectedResult, result.Result()[0].(string))
	}
}
