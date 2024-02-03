package async

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAtomicGroupTerminatesOnOneCommandFailed(t *testing.T) {
	ctx := context.Background()
	group := NewAtomicGroup(ctx)

	err := errors.New("error")
	group.Add(func(ctx context.Context) error {
		return err // failure
	})
	group.Add(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	group.Wait()
	require.Equal(t, err, group.Error())
}

func TestAtomicGroupWaitsAllToFinish(t *testing.T) {
	ctx := context.Background()
	group := NewAtomicGroup(ctx)

	finished := false
	group.Add(func(ctx context.Context) error {
		time.Sleep(1 * time.Millisecond)
		return nil // success
	})
	group.Add(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Millisecond):
				finished = true
				return nil
			}
		}
	})

	group.Wait()
	require.True(t, finished)
}
