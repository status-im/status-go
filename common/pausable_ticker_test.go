package common

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// makePauseCh returns a channel and a helper to send pause/resume signals.
func makePauseCh() (chan bool, func(bool)) {
	ch := make(chan bool, 1)
	send := func(paused bool) {
		// Drain stale value first to avoid blocking the sender.
		select {
		case <-ch:
		default:
		}
		ch <- paused
	}
	return ch, send
}

func TestPausableTicker_TicksWhenRunning(t *testing.T) {
	pauseCh, send := makePauseCh()

	var count atomic.Int32
	pt := NewPausableTicker(PausableTickerConfig{
		Interval: 10 * time.Millisecond,
		OnTick:   func() { count.Add(1) },
	}, pauseCh)

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		pt.Run(quit)
	}()

	send(false) // start running

	require.Eventually(t, func() bool {
		return count.Load() >= 3
	}, 500*time.Millisecond, 5*time.Millisecond, "expected at least 3 ticks while running")

	close(quit)
	<-done
}

func TestPausableTicker_NoTicksWhenPaused(t *testing.T) {
	pauseCh, send := makePauseCh()

	var count atomic.Int32
	pt := NewPausableTicker(PausableTickerConfig{
		Interval: 10 * time.Millisecond,
		OnTick:   func() { count.Add(1) },
	}, pauseCh)

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		pt.Run(quit)
	}()

	send(true) // start paused
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, int32(0), count.Load(), "expected no ticks while paused")

	close(quit)
	<-done
}

func TestPausableTicker_PauseStopsTicks(t *testing.T) {
	pauseCh, send := makePauseCh()

	var count atomic.Int32
	pt := NewPausableTicker(PausableTickerConfig{
		Interval: 10 * time.Millisecond,
		OnTick:   func() { count.Add(1) },
	}, pauseCh)

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		pt.Run(quit)
	}()

	send(false) // running
	require.Eventually(t, func() bool {
		return count.Load() >= 2
	}, 500*time.Millisecond, 5*time.Millisecond)

	send(true) // pause
	time.Sleep(50 * time.Millisecond)
	snapshot := count.Load()
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, snapshot, count.Load(), "count should not increase while paused")

	close(quit)
	<-done
}

func TestPausableTicker_ResumeRestartsTicks(t *testing.T) {
	pauseCh, send := makePauseCh()

	var count atomic.Int32
	pt := NewPausableTicker(PausableTickerConfig{
		Interval: 10 * time.Millisecond,
		OnTick:   func() { count.Add(1) },
	}, pauseCh)

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		pt.Run(quit)
	}()

	send(true) // start paused
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, int32(0), count.Load(), "no ticks while paused")

	send(false) // resume
	require.Eventually(t, func() bool {
		return count.Load() >= 2
	}, 500*time.Millisecond, 5*time.Millisecond, "expected ticks after resume")

	close(quit)
	<-done
}

func TestPausableTicker_QuitStopsGoroutine(t *testing.T) {
	pauseCh, send := makePauseCh()

	pt := NewPausableTicker(PausableTickerConfig{
		Interval: 10 * time.Millisecond,
		OnTick:   func() {},
	}, pauseCh)

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		pt.Run(quit)
	}()

	send(false)
	close(quit)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("goroutine did not stop after quit was closed")
	}
}

func TestPausableTicker_ClosedPauseCh(t *testing.T) {
	pauseCh := make(chan bool)
	close(pauseCh)

	pt := NewPausableTicker(PausableTickerConfig{
		Interval: 10 * time.Millisecond,
		OnTick:   func() {},
	}, pauseCh)

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		pt.Run(quit)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not return when pauseCh was closed")
	}
}

func TestPausableTicker_OnPauseChanged(t *testing.T) {
	pauseCh, send := makePauseCh()

	var states []bool
	pt := NewPausableTicker(PausableTickerConfig{
		Interval: time.Hour, // irrelevant for this test
		OnTick:   func() {},
		OnPauseChanged: func(paused bool) {
			states = append(states, paused)
		},
	}, pauseCh)

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		pt.Run(quit)
	}()

	send(false) // initial: running
	time.Sleep(20 * time.Millisecond)
	send(true)  // pause
	time.Sleep(20 * time.Millisecond)
	send(false) // resume
	time.Sleep(20 * time.Millisecond)

	close(quit)
	<-done

	require.Equal(t, []bool{false, true, false}, states)
}
