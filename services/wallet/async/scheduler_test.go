package async

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	noActionPerformed = "no action performed"
	taskCalled        = "task called"
	taskResultCalled  = "task result called"
)

func TestScheduler_Enqueue_Simple(t *testing.T) {
	s := NewScheduler()
	callChan := make(chan string, 10)

	testFunction := func(policy ReplacementPolicy, failTest bool) {
		testTask := TaskType{1, policy}
		ignored := s.Enqueue(testTask, func(ctx context.Context) (interface{}, error) {
			callChan <- taskCalled
			if failTest {
				return nil, errors.New("test error")
			}
			return 123, nil
		}, func(res interface{}, taskType TaskType, err error) {
			if failTest {
				require.Error(t, err)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, 123, res)
			}
			require.Equal(t, testTask, taskType)
			callChan <- taskResultCalled
		})
		require.False(t, ignored)

		lastRes := noActionPerformed
		done := false
		for !done {
			select {
			case callRes := <-callChan:
				if callRes == taskCalled {
					require.Equal(t, noActionPerformed, lastRes)
				} else if callRes == taskResultCalled {
					require.Equal(t, taskCalled, lastRes)
					done = true
				} else {
					require.Fail(t, "unexpected result", `"%s" for policy %d`, callRes, policy)
				}
				lastRes = callRes
			case <-time.After(1 * time.Second):
				require.Fail(t, "test not completed in time", `last result: "%s" for policy %d`, lastRes, policy)
			}
		}

		require.Equal(t, taskResultCalled, lastRes)
	}

	testFailed := false
	for i := 0; i < 2; i++ {
		testFailed = (i == 0)
		for policy := range []ReplacementPolicy{ReplacementPolicyCancelOld, ReplacementPolicyIgnoreNew} {
			testFunction(policy, testFailed)
		}
	}
}

// Validate the task is cancelled when a new one is scheduled and that the third one will overwrite the second one
func TestScheduler_Enqueue_VerifyReplacementPolicyCancelOld(t *testing.T) {
	s := NewScheduler()

	type testStage string
	const (
		stage1FirstTaskStarted                testStage = "First task started"
		stage2ThirdEnqueueOverwroteSecondTask testStage = "Third Enqueue overwrote second task"
		stage3ExitingFirstCancelledTask       testStage = "Exiting first cancelled task"
		stage5ThirdTaskRunning                testStage = "Third task running"
		stage6ThirdTaskResponse               testStage = "Third task response"
	)

	testStages := []testStage{
		stage1FirstTaskStarted,
		stage2ThirdEnqueueOverwroteSecondTask,
		stage3ExitingFirstCancelledTask,
		stage5ThirdTaskRunning,
		stage6ThirdTaskResponse,
	}

	callChan := make(chan testStage, len(testStages))
	var firstRunWG, secondRunWG, thirdRunWG sync.WaitGroup

	firstRunWG.Add(1)
	secondRunWG.Add(1)
	thirdRunWG.Add(1)

	stage4AsyncFirstTaskCanceledResponse := false

	testTask := TaskType{1, ReplacementPolicyCancelOld}
	for i := 0; i < 2; i++ {
		currentIndex := i
		ignored := s.Enqueue(testTask, func(workCtx context.Context) (interface{}, error) {
			callChan <- stage1FirstTaskStarted

			// Mark first task running so that the second Enqueue will cancel this one and overwrite it
			firstRunWG.Done()

			// Wait for the first task to be cancelled by the second one
			select {
			case <-workCtx.Done():
				require.ErrorAs(t, workCtx.Err(), &context.Canceled)

				// Unblock the third Enqueue call
				secondRunWG.Done()

				// Block the second task from running until the third one is overwriting the second one that didn't run
				thirdRunWG.Wait()
				callChan <- stage3ExitingFirstCancelledTask
			case <-time.After(1 * time.Second):
				require.Fail(t, "task not cancelled in time")
			}
			return nil, workCtx.Err()
		}, func(res interface{}, taskType TaskType, err error) {
			switch currentIndex {
			case 0:
				// First task was cancelled by the second one Enqueue call
				stage4AsyncFirstTaskCanceledResponse = true

				require.ErrorAs(t, err, &context.Canceled)
			case 1:
				callChan <- stage2ThirdEnqueueOverwroteSecondTask

				// Unblock the first task from blocking execution of the third one
				// also validate that the third Enqueue call overwrote running the second one
				thirdRunWG.Done()

				require.True(t, errors.Is(err, ErrTaskOverwritten))
			}
		})
		require.False(t, ignored)

		// Wait first task to run
		firstRunWG.Wait()
	}
	// Wait for the second task to be cancelled before running the third one
	secondRunWG.Wait()

	ignored := s.Enqueue(testTask, func(ctx context.Context) (interface{}, error) {
		callChan <- stage5ThirdTaskRunning
		return 123, errors.New("test error")
	}, func(res interface{}, taskType TaskType, err error) {
		require.Error(t, err)
		require.Equal(t, testTask, taskType)
		require.Equal(t, 123, res)

		callChan <- stage6ThirdTaskResponse
	})
	require.False(t, ignored)

	lastRes := noActionPerformed
	expectedTestStageIndex := 0
	for i := 0; i < len(testStages); i++ {
		select {
		case callRes := <-callChan:
			require.Equal(t, testStages[expectedTestStageIndex], callRes, "task stage out of order; expected %s, got %s", testStages[expectedTestStageIndex], callRes)
			expectedTestStageIndex++
		case <-time.After(1 * time.Second):
			require.Fail(t, "test not completed in time", `last result: "%s" for cancel task policy`, lastRes)
		}
	}
	require.True(t, stage4AsyncFirstTaskCanceledResponse)
}

func TestScheduler_Enqueue_VerifyReplacementPolicyIgnoreNew(t *testing.T) {
	s := NewScheduler()
	callChan := make(chan string, 10)
	workloadWG := sync.WaitGroup{}
	taskCallCount := 0
	resultCallCount := 0

	workloadWG.Add(1)
	testTask := TaskType{1, ReplacementPolicyIgnoreNew}
	ignored := s.Enqueue(testTask, func(workCtx context.Context) (interface{}, error) {
		workloadWG.Wait()
		require.NoError(t, workCtx.Err())
		taskCallCount++
		callChan <- taskCalled
		return 123, nil
	}, func(res interface{}, taskType TaskType, err error) {
		require.NoError(t, err)
		require.Equal(t, testTask, taskType)
		require.Equal(t, 123, res)
		resultCallCount++
		callChan <- taskResultCalled
	})
	require.False(t, ignored)

	ignored = s.Enqueue(testTask, func(ctx context.Context) (interface{}, error) {
		require.Fail(t, "unexpected call")
		return nil, errors.New("unexpected call")
	}, func(res interface{}, taskType TaskType, err error) {
		require.Fail(t, "unexpected result call")
	})
	require.True(t, ignored)
	workloadWG.Done()

	lastRes := noActionPerformed
	done := false
	for !done {
		select {
		case callRes := <-callChan:
			if callRes == taskCalled {
				require.Equal(t, noActionPerformed, lastRes)
			} else if callRes == taskResultCalled {
				require.Equal(t, taskCalled, lastRes)
				done = true
			} else {
				require.Fail(t, "unexpected result", `"%s" for ignore task policy`, callRes)
			}
			lastRes = callRes
		case <-time.After(1 * time.Second):
			require.Fail(t, "test not completed in time", `last result: "%s" for ignore task policy`, lastRes)
		}
	}

	require.Equal(t, 1, resultCallCount)
	require.Equal(t, 1, taskCallCount)

	require.Equal(t, taskResultCalled, lastRes)
}

func TestScheduler_Enqueue_ValidateOrder(t *testing.T) {
	s := NewScheduler()
	waitEnqueueAll := sync.WaitGroup{}

	type failType bool
	const (
		fail failType = true
		pass failType = false
	)

	type enqueueParams struct {
		taskType   TaskType
		taskAction failType
		callIndex  int
	}
	testTask1 := TaskType{1, ReplacementPolicyCancelOld}
	testTask2 := TaskType{2, ReplacementPolicyCancelOld}
	testTask3 := TaskType{3, ReplacementPolicyIgnoreNew}
	// Task type, ReplacementPolicy: CancelOld if true IgnoreNew if false, task fail or success, index
	enqueueSequence := []enqueueParams{
		{testTask1, pass, 0}, // 1 task event
		{testTask2, pass, 0}, // 0 task event
		{testTask3, fail, 0}, // 1 task event
		{testTask3, pass, 0}, // 0 task event
		{testTask2, pass, 0}, // 1 task event
		{testTask1, pass, 0}, // 1 task event
		{testTask3, fail, 0}, // 0 run event
	}
	const taskEventCount = 4

	taskSuccessChan := make(chan enqueueParams, len(enqueueSequence))
	taskCanceledChan := make(chan enqueueParams, len(enqueueSequence))
	taskFailedChan := make(chan enqueueParams, len(enqueueSequence))
	resChan := make(chan enqueueParams, len(enqueueSequence))

	firstIgnoreNewProcessed := make(map[TaskType]bool)

	ignoredCount := 0

	waitEnqueueAll.Add(1)
	for i := 0; i < len(enqueueSequence); i++ {
		enqueueSequence[i].callIndex = i

		p := enqueueSequence[i]

		currentIndex := i

		ignored := s.Enqueue(p.taskType, func(ctx context.Context) (interface{}, error) {
			waitEnqueueAll.Wait()

			if p.taskType.Policy == ReplacementPolicyCancelOld && ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
				taskCanceledChan <- p
				return nil, ctx.Err()
			}

			if p.taskAction == fail {
				taskFailedChan <- p
				return nil, errors.New("test error")
			}
			taskSuccessChan <- p
			return 10 * (currentIndex + 1), nil
		}, func(res interface{}, taskType TaskType, err error) {
			require.Equal(t, p.taskType, taskType)
			resChan <- p
		})

		if ignored {
			ignoredCount++
		}

		if _, ok := firstIgnoreNewProcessed[p.taskType]; !ok {
			require.False(t, ignored)
			firstIgnoreNewProcessed[p.taskType] = p.taskType.Policy == ReplacementPolicyCancelOld
		} else {
			if p.taskType.Policy == ReplacementPolicyIgnoreNew {
				require.True(t, ignored)
			} else {
				require.False(t, ignored)
			}
		}
	}

	waitEnqueueAll.Done()

	taskSuccessCount := make(map[TaskType]int)
	taskCanceledCount := make(map[TaskType]int)
	taskFailedCount := make(map[TaskType]int)
	resChanCount := make(map[TaskType]int)

	// Only ignored don't generate result events
	expectedEventsCount := len(enqueueSequence) - ignoredCount + taskEventCount
	for i := 0; i < expectedEventsCount; i++ {
		// Loop for run and result calls
		select {
		case p := <-taskSuccessChan:
			taskSuccessCount[p.taskType]++
		case p := <-taskCanceledChan:
			taskCanceledCount[p.taskType]++
		case p := <-taskFailedChan:
			taskFailedCount[p.taskType]++
		case p := <-resChan:
			resChanCount[p.taskType]++
		case <-time.After(1 * time.Second):
			require.Fail(t, "test not completed in time")
		}
	}

	require.Equal(t, 1, taskSuccessCount[testTask1], "expected one task call for type: %d had %d", 1, taskSuccessCount[testTask1])
	require.Equal(t, 1, taskSuccessCount[testTask2], "expected one task call for type: %d had %d", 2, taskSuccessCount[testTask2])
	require.Equal(t, 0, taskSuccessCount[testTask3], "expected no task call for type: %d had %d", 3, taskSuccessCount[testTask3])

	require.Equal(t, 1, taskCanceledCount[testTask1], "expected one task call for type: %d had %d", 1, taskSuccessCount[testTask1])
	require.Equal(t, 0, taskCanceledCount[testTask2], "expected no task call for type: %d had %d", 2, taskSuccessCount[testTask2])
	require.Equal(t, 0, taskCanceledCount[testTask3], "expected no task call for type: %d had %d", 3, taskSuccessCount[testTask3])

	require.Equal(t, 0, taskFailedCount[testTask1], "expected no task call for type: %d had %d", 1, taskSuccessCount[testTask1])
	require.Equal(t, 0, taskFailedCount[testTask2], "expected no task call for type: %d had %d", 2, taskSuccessCount[testTask2])
	require.Equal(t, 1, taskFailedCount[testTask3], "expected one task call for type: %d had %d", 3, taskSuccessCount[testTask3])

	require.Equal(t, 2, resChanCount[testTask1], "expected two task call for type: %d had %d", 1, taskSuccessCount[testTask1])
	require.Equal(t, 2, resChanCount[testTask2], "expected tow task call for type: %d had %d", 2, taskSuccessCount[testTask2])
	require.Equal(t, 1, resChanCount[testTask3], "expected one task call for type: %d had %d", 3, taskSuccessCount[testTask3])
}

func TestScheduler_Enqueue_InResult(t *testing.T) {
	s := NewScheduler()
	callChan := make(chan int, 6)

	s.Enqueue(TaskType{ID: 1, Policy: ReplacementPolicyCancelOld},
		func(ctx context.Context) (interface{}, error) {
			callChan <- 0
			return nil, nil
		}, func(res interface{}, taskType TaskType, err error) {
			callChan <- 1
			s.Enqueue(TaskType{1, ReplacementPolicyCancelOld}, func(ctx context.Context) (interface{}, error) {
				callChan <- 2
				return nil, nil
			}, func(res interface{}, taskType TaskType, err error) {
				callChan <- 3
				s.Enqueue(TaskType{1, ReplacementPolicyCancelOld}, func(ctx context.Context) (interface{}, error) {
					callChan <- 4
					return nil, nil
				}, func(res interface{}, taskType TaskType, err error) {
					callChan <- 5
				})
			})
		},
	)
	for i := 0; i < 6; i++ {
		select {
		case res := <-callChan:
			require.Equal(t, i, res)
		case <-time.After(1 * time.Second):
			require.Fail(t, "test not completed in time")
		}
	}
}
