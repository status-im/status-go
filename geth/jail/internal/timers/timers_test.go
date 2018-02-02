package timers_test

import (
	"context"
	"testing"
	"time"

	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/timers"
	"github.com/status-im/status-go/geth/jail/internal/vm"
	"github.com/stretchr/testify/suite"
)

func (s *TimersSuite) TestSetTimeout() {
	err := s.vm.Set("__capture", func() {
		s.ch <- struct{}{}
	})
	s.NoError(err)

	err = s.loop.Eval(`setTimeout(function(n) {
		if (Date.now() - n < 50) {
			throw new Error('timeout was called too soon');
		}
		__capture();
	}, 50, Date.now());`)
	s.NoError(err)

	select {
	case <-s.ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
		return
	}
}

func (s *TimersSuite) TestClearTimeout() {
	err := s.vm.Set("__shouldNeverRun", func() {
		s.Fail("should never run")
	})
	s.NoError(err)

	err = s.loop.Eval(`clearTimeout(setTimeout(function() {
		__shouldNeverRun();
	}, 50));`)
	s.NoError(err)

	<-time.After(100 * time.Millisecond)
}

func (s *TimersSuite) TestSetInterval() {
	err := s.vm.Set("__done", func() {
		s.ch <- struct{}{}
	})
	s.NoError(err)

	err = s.loop.Eval(`
		var c = 0;
		var iv = setInterval(function() {
			if (c === 1) {
				clearInterval(iv);
				__done();
			}
			c++;
		}, 50);
	`)
	s.NoError(err)

	select {
	case <-s.ch:
		value, err := s.vm.Get("c")
		s.NoError(err)
		n, err := value.ToInteger()
		s.NoError(err)
		s.Equal(2, int(n))
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
	}
}

func (s *TimersSuite) TestClearIntervalImmediately() {
	err := s.vm.Set("__shouldNeverRun", func() {
		s.Fail("should never run")
	})
	s.NoError(err)

	err = s.loop.Eval(`clearInterval(setInterval(function() {
		__shouldNeverRun();
	}, 50));`)
	s.NoError(err)

	<-time.After(100 * time.Millisecond)
}

func (s *TimersSuite) TestImmediateTimer() {
	err := s.vm.Set("__done", func() {
		s.ch <- struct{}{}
	})
	s.NoError(err)

	err = s.loop.Eval(`
		var v = setImmediate(function() {
			__done();
		});
	`)
	s.NoError(err)

	select {
	case <-s.ch:
		value, err := s.vm.Get("v")
		s.NoError(err)
		s.NotNil(value)
	case <-time.After(100 * time.Millisecond):
		s.Fail("test timed out")
	}
}

type TimersSuite struct {
	suite.Suite

	loop *loop.Loop
	vm   *vm.VM

	ch chan struct{}
}

func (s *TimersSuite) SetupTest() {
	s.vm = vm.New()
	s.loop = loop.New(s.vm)

	go s.loop.Run(context.Background()) //nolint: errcheck

	err := timers.Define(s.vm, s.loop)
	s.NoError(err)

	s.ch = make(chan struct{})
}

func TestTimersSuite(t *testing.T) {
	suite.Run(t, new(TimersSuite))
}
