package try

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTry(t *testing.T) {
	MaxRetries = 20
	SomeFunction := func() (string, error) {
		return "", nil
	}
	err := Do(func(attempt int) (bool, error) {
		var err error
		_, err = SomeFunction()
		return attempt < 5, err // try 5 times
	})
	require.NoError(t, err)
}

func TestTryPanic(t *testing.T) {
	SomeFunction := func() (string, error) {
		panic("something went badly wrong")
	}
	err := Do(func(attempt int) (retry bool, err error) {
		retry = attempt < 5 // try 5 times
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		_, err = SomeFunction()
		return
	})
	require.Error(t, err)
}

func TestTryDoSuccessful(t *testing.T) {
	callCount := 0
	err := Do(func(attempt int) (bool, error) {
		callCount++
		return attempt < 5, nil
	})
	require.NoError(t, err)
	require.Equal(t, callCount, 1)
}

func TestTryDoFailed(t *testing.T) {
	wrongErr := errors.New("something went wrong")
	callCount := 0
	err := Do(func(attempt int) (bool, error) {
		callCount++
		return attempt < 5, wrongErr
	})
	require.Equal(t, err, wrongErr)
	require.Equal(t, callCount, 5)
}

func TestTryPanics(t *testing.T) {
	wrongErr := errors.New("something went wrong")
	callCount := 0
	err := Do(func(attempt int) (retry bool, err error) {
		retry = attempt < 5
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		callCount++
		if attempt > 2 {
			panic("I don't like three")
		}
		err = wrongErr
		return
	})
	require.Equal(t, err.Error(), "panic: I don't like three")
	require.Equal(t, callCount, 5)
}

func TestRetryLimit(t *testing.T) {
	err := Do(func(attempt int) (bool, error) {
		return true, errors.New("nope")
	})
	require.Error(t, err)
	require.Equal(t, IsMaxRetries(err), true)
}
