package mailserver

import "time"

type ticker struct {
	timeTicker *time.Ticker
}

func (t *ticker) run(period time.Duration, fn func()) {
	if t.timeTicker != nil {
		return
	}

	t.timeTicker = time.NewTicker(period)
	go func() {
		for range t.timeTicker.C {
			fn()
		}
	}()
}

func (t *ticker) stop() {
	t.timeTicker.Stop()
}
