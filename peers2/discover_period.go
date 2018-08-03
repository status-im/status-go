package peers2

import (
	"sync"
	"time"
)

const (
	// defaultFastSync is a recommended value for aggressive peers search.
	defaultFastSync = 3 * time.Second
	// defaultSlowSync is a recommended value for slow (background) peers search.
	defaultSlowSync = 30 * time.Second
	// defaultFastSyncTimeout tells how long the fast sync should last.
	defaultFastSyncTimeout = 30 * time.Second
)

type fastSlowDiscoverPeriod struct {
	sync.RWMutex

	fast        time.Duration
	slow        time.Duration
	fastTimeout time.Duration

	currentValue    time.Duration
	cs              []chan time.Duration
	csTimeoutCancel []chan struct{}
}

func newFastSlowDiscoverPeriod(fast, slow, fastTimeout time.Duration) *fastSlowDiscoverPeriod {
	return &fastSlowDiscoverPeriod{
		fast:        fast,
		slow:        slow,
		fastTimeout: fastTimeout,
	}
}

func defaultFastSlowDiscoverPeriod() *fastSlowDiscoverPeriod {
	return newFastSlowDiscoverPeriod(defaultFastSync, defaultSlowSync, defaultFastSyncTimeout)
}

func (p *fastSlowDiscoverPeriod) channel() <-chan time.Duration {
	ch := make(chan time.Duration, 1)
	ch <- p.fast
	cancel := make(chan struct{})

	p.Lock()
	p.cs = append(p.cs, ch)
	p.csTimeoutCancel = append(p.csTimeoutCancel, cancel)
	p.Unlock()

	// timeout fast mode
	go func(val, timeout time.Duration) {
		select {
		case <-cancel:
		case <-time.After(timeout):
			ch <- val
		}
	}(p.slow, p.fastTimeout)

	return ch
}

func (p *fastSlowDiscoverPeriod) transSlow() { p.trans(p.slow) }

func (p *fastSlowDiscoverPeriod) trans(val time.Duration) {
	p.RLock()
	defer p.RUnlock()

	if p.currentValue == val {
		return
	}
	p.currentValue = val

	for _, c := range p.cs {
		c <- val
	}
}

func (p *fastSlowDiscoverPeriod) close() {
	p.Lock()
	defer p.Unlock()

	for _, c := range p.csTimeoutCancel {
		close(c)
	}
	p.csTimeoutCancel = nil

	for _, c := range p.cs {
		close(c)
	}
	p.cs = nil
}
