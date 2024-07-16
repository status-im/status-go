package communities

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

var membersReevaluationTick = 10 * time.Second
var membersReevaluationInterval = 8 * time.Hour
var membersReevaluationCooldown = 5 * time.Minute

type reevaluationExecutionType int

const (
	reevaluationExecutionRegular reevaluationExecutionType = iota
	reevaluationExecutionOnDemand
	reevaluationExecutionForced
)

type reevaluationFunc = func(reevaluationExecutionType) (stop bool, err error)

type membersReevaluationTask struct {
	startedAt  time.Time
	endedAt    time.Time
	demandedAt time.Time
	execute    reevaluationFunc
	mutex      sync.Mutex
}

type membersReevaluationScheduler struct {
	tasks  sync.Map // stores `membersReevaluationTask`
	forces sync.Map // stores `chan struct{}`
	quit   chan struct{}
	logger *zap.Logger
}

func (t *membersReevaluationTask) shouldExecute(force bool) *reevaluationExecutionType {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if force {
		result := reevaluationExecutionForced
		return &result
	}

	now := time.Now()

	if !t.endedAt.After(now.Add(-membersReevaluationCooldown)) {
		return nil
	}

	if t.endedAt.After(now.Add(-membersReevaluationInterval)) {
		result := reevaluationExecutionRegular
		return &result
	}

	if t.startedAt.Before(t.demandedAt) {
		result := reevaluationExecutionOnDemand
		return &result
	}

	return nil
}

func (t *membersReevaluationTask) setStartTime(time time.Time) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.startedAt = time
}

func (t *membersReevaluationTask) setEndTime(time time.Time) (elapsed time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.endedAt = time
	return t.endedAt.Sub(t.startedAt)
}

func (t *membersReevaluationTask) setDemandTime(time time.Time) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.demandedAt = time
}

func (s *membersReevaluationScheduler) getTask(communityID string) (*membersReevaluationTask, error) {
	t, exists := s.tasks.Load(communityID)
	if !exists {
		return nil, errors.New("task doesn't exist")
	}

	task, ok := t.(*membersReevaluationTask)
	if !ok {
		return nil, errors.New("invalid task type")
	}

	return task, nil
}

func (s *membersReevaluationScheduler) iterate(communityID string, force bool) (stop bool) {
	task, err := s.getTask(communityID)
	if err != nil {
		return true
	}

	executionType := task.shouldExecute(force)
	if executionType == nil {
		return false
	}

	task.setStartTime(time.Now())

	stop, err = task.execute(*executionType)
	if err != nil {
		s.logger.Error("can't reevaluate members", zap.Error(err))
		return stop
	}

	elapsed := task.setEndTime(time.Now())

	s.logger.Info("reevaluation finished",
		zap.String("communityID", communityID),
		zap.Duration("elapsed", elapsed),
	)

	return stop
}

func (s *membersReevaluationScheduler) loop(communityID string, reevaluator reevaluationFunc, setupDone chan struct{}) {
	_, exists := s.tasks.Load(communityID)
	if exists {
		setupDone <- struct{}{}
		return
	}

	s.tasks.Store(communityID, &membersReevaluationTask{execute: reevaluator})
	defer s.tasks.Delete(communityID)

	force := make(chan struct{}, 10)
	s.forces.Store(communityID, force)
	defer s.forces.Delete(communityID)

	ticker := time.NewTicker(membersReevaluationTick)
	defer ticker.Stop()

	setupDone <- struct{}{}

	// Perform the first iteration immediately
	stop := s.iterate(communityID, true)
	if stop {
		return
	}

	for {
		select {
		case <-ticker.C:
			stop := s.iterate(communityID, false)
			if stop {
				return
			}

		case <-force:
			stop := s.iterate(communityID, true)
			if stop {
				return
			}

		case <-s.quit:
			return
		}
	}
}

func (s *membersReevaluationScheduler) Start(communityID string, reevaluator reevaluationFunc) {
	setupDone := make(chan struct{})
	go s.loop(communityID, reevaluator, setupDone)
	<-setupDone
}

func (s *membersReevaluationScheduler) Push(communityID string) error {
	task, err := s.getTask(communityID)
	if err != nil {
		return err
	}
	task.setDemandTime(time.Now())
	return nil
}

func (s *membersReevaluationScheduler) Force(communityID string) error {
	t, exists := s.forces.Load(communityID)
	if !exists {
		return errors.New("scheduler not started yet")
	}

	force, ok := t.(chan struct{})
	if !ok {
		return errors.New("invalid cast")
	}

	force <- struct{}{}
	return nil
}
