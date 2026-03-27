package common

import "time"

// PausableTickerConfig configures a PausableTicker.
type PausableTickerConfig struct {
	// Interval is the tick interval.
	Interval time.Duration
	// OnTick is called each time the ticker fires while not paused.
	OnTick func()
	// OnPauseChanged is optional. Called whenever the pause state transitions,
	// including once with the initial state before the first tick. Useful for
	// consumers that need to change their own interval or behaviour (e.g. NTP).
	OnPauseChanged func(paused bool)
	// TickerArmed is optional. When set, called with true after a time.Ticker is
	// created for the running state, and false after it is stopped (pause,
	// replacement, or Run exit). Production code can leave this nil; tests use it
	// to assert no underlying ticker is armed while paused.
	TickerArmed func(armed bool)
}

// PausableTicker runs a periodic tick that is suppressed while paused.
//
// pauseCh is a channel of bool values: true means "now paused", false means
// "now running". The first value sent on the channel sets the initial state
// and must be delivered before Run returns for the first time. This matches
// the contract of PauseBroadcaster.Subscribe().
//
// Usage:
//
//	sub := s.Subscribe()
//	defer sub.Unsubscribe()
//	pt := common.NewPausableTicker(common.PausableTickerConfig{
//	    Interval: 30 * time.Second,
//	    OnTick:   func() { doWork() },
//	}, sub.C())
//	pt.Run(quit)
type PausableTicker struct {
	cfg     PausableTickerConfig
	pauseCh <-chan bool
}

// NewPausableTicker creates a PausableTicker.
func NewPausableTicker(cfg PausableTickerConfig, pauseCh <-chan bool) *PausableTicker {
	return &PausableTicker{cfg: cfg, pauseCh: pauseCh}
}

// Run blocks, firing cfg.OnTick at cfg.Interval while not paused.
// It returns when quit is closed or pauseCh is closed.
// While paused, no time.Ticker is active (see cfg.TickerArmed for tests).
func (pt *PausableTicker) Run(quit <-chan struct{}) {
	if pt.cfg.Interval <= 0 || pt.cfg.OnTick == nil {
		return
	}

	var ticker *time.Ticker
	notifyArmed := func(armed bool) {
		if pt.cfg.TickerArmed != nil {
			pt.cfg.TickerArmed(armed)
		}
	}
	defer func() {
		if ticker != nil {
			ticker.Stop()
			notifyArmed(false)
		}
	}()

	startTicker := func() {
		if ticker != nil {
			ticker.Stop()
			notifyArmed(false)
		}
		ticker = time.NewTicker(pt.cfg.Interval)
		notifyArmed(true)
	}
	stopTicker := func() {
		if ticker != nil {
			ticker.Stop()
			notifyArmed(false)
			ticker = nil
		}
	}

	// Read initial pause state; this unblocks once the lifecycle sends it.
	var paused bool
	select {
	case p, ok := <-pt.pauseCh:
		if !ok {
			return
		}
		paused = p
	case <-quit:
		return
	}

	if pt.cfg.OnPauseChanged != nil {
		pt.cfg.OnPauseChanged(paused)
	}

	var tickerC <-chan time.Time
	if !paused {
		startTicker()
		tickerC = ticker.C
	}

	for {
		select {
		case pausedState, ok := <-pt.pauseCh:
			if !ok {
				return
			}
			prev := paused
			paused = pausedState
			if paused && !prev {
				stopTicker()
				tickerC = nil
			} else if !paused && prev {
				startTicker()
				tickerC = ticker.C
			}
			if pt.cfg.OnPauseChanged != nil {
				pt.cfg.OnPauseChanged(paused)
			}
		case <-tickerC:
			pt.cfg.OnTick()
		case <-quit:
			return
		}
	}
}
