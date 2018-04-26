package peers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TopicPoolSyncModeSuite struct {
	suite.Suite

	fastMode      time.Duration
	slowMode      time.Duration
	fastModeLimit time.Duration

	sync   *syncStrategy
	period <-chan time.Duration
}

func TestTopicPoolSyncModeSuite(t *testing.T) {
	suite.Run(t, new(TopicPoolSyncModeSuite))
}

func (s *TopicPoolSyncModeSuite) SetupTest() {
	s.fastMode = time.Millisecond
	s.slowMode = time.Millisecond * 100
	s.fastModeLimit = time.Millisecond * 50
	s.sync = newSyncStrategy(s.fastMode, s.slowMode, s.fastModeLimit)
	s.period = s.sync.Start()
}

func (s *TopicPoolSyncModeSuite) TearDown() {
	s.sync.Stop()
}

func (s *TopicPoolSyncModeSuite) TestSyncStart() {
	s.Equal(s.fastMode, <-s.period)
}

func (s *TopicPoolSyncModeSuite) TestSyncUpdate() {
	s.Equal(s.fastMode, <-s.period)

	// should stay with fast mode
	s.sync.Update(0, 1, 2)
	select {
	case <-s.period:
		s.FailNow("period should not be updated")
	default:
		// pass
	}

	// should switch to slow mode due to reaching lower limit
	s.sync.Update(1, 1, 2)
	s.Equal(s.slowMode, <-s.period)

	// should stay with low mode
	s.sync.Update(1, 1, 2)
	select {
	case <-s.period:
		s.FailNow("period should not be updated")
	default:
		// pass
	}

	// when being in slow mode, after passing fast mode limit time,
	// nothing should change
	time.Sleep(s.fastModeLimit)
	select {
	case <-s.period:
		s.FailNow("period should not be updated")
	default:
		// pass
	}
}

func (s *TopicPoolSyncModeSuite) TestSyncLimitFastPeriod() {
	// test for Start()
	s.Equal(s.fastMode, <-s.period)
	time.Sleep(s.fastModeLimit)
	s.Equal(s.slowMode, <-s.period)

	// test for Update()
	s.sync.Update(0, 1, 2)
	s.Equal(s.fastMode, <-s.period)
	time.Sleep(s.fastModeLimit)
	s.Equal(s.slowMode, <-s.period)
}
