package transactions

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConditionalRepeater_RunOnce(t *testing.T) {
	var wg sync.WaitGroup
	runCount := 0
	wg.Add(1)
	taskRunner := NewConditionalRepeater(1*time.Nanosecond, func(ctx context.Context) bool {
		runCount++
		defer wg.Done()
		return WorkDone
	})
	taskRunner.RunUntilDone()
	// Wait for task to run
	wg.Wait()
	taskRunner.Stop()
	require.Greater(t, runCount, 0)
}

func TestConditionalRepeater_RunUntilDone_MultipleCalls(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(5)
	runCount := 0
	taskRunner := NewConditionalRepeater(1*time.Nanosecond, func(ctx context.Context) bool {
		runCount++
		wg.Done()
		return runCount == 5
	})
	for i := 0; i < 10; i++ {
		taskRunner.RunUntilDone()
	}
	// Wait for all tasks to run
	wg.Wait()
	taskRunner.Stop()
	require.Greater(t, runCount, 4)
}

func TestConditionalRepeater_Stop(t *testing.T) {
	var taskRunningWG, taskCanceledWG, taskFinishedWG sync.WaitGroup
	taskRunningWG.Add(1)
	taskCanceledWG.Add(1)
	taskFinishedWG.Add(1)
	taskRunner := NewConditionalRepeater(1*time.Nanosecond, func(ctx context.Context) bool {
		defer taskFinishedWG.Done()
		select {
		case <-ctx.Done():
			require.Fail(t, "task should not be canceled yet")
		default:
		}

		// Wait to caller to stop the task
		taskRunningWG.Done()
		taskCanceledWG.Wait()

		select {
		case <-ctx.Done():
			require.Error(t, ctx.Err())
		default:
			require.Fail(t, "task should be canceled")
		}

		return WorkDone
	})
	taskRunner.RunUntilDone()
	taskRunningWG.Wait()

	taskRunner.Stop()
	taskCanceledWG.Done()

	taskFinishedWG.Wait()
}
