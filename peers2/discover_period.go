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

	fast time.Duration
	slow time.Duration

	cs []chan time.Duration
}

func newFastSlowDiscoverPeriod(fast, slow time.Duration) *fastSlowDiscoverPeriod {
	return &fastSlowDiscoverPeriod{
		fast: fast,
		slow: slow,
	}
}

func defaultFastSlowDiscoverPeriod() *fastSlowDiscoverPeriod {
	return newFastSlowDiscoverPeriod(defaultFastSync, defaultSlowSync)
}

func (p *fastSlowDiscoverPeriod) channel() <-chan time.Duration {
	// we don't expect the period to be changed more than 10x
	ch := make(chan time.Duration, 10)
	ch <- p.fast

	p.Lock()
	p.cs = append(p.cs, ch)
	p.Unlock()

	return ch
}

func (p *fastSlowDiscoverPeriod) transSlow() { p.trans(p.slow) }

func (p *fastSlowDiscoverPeriod) transFast() { p.trans(p.fast) }

func (p *fastSlowDiscoverPeriod) close() {
	p.RLock()
	for _, c := range p.cs {
		close(c)
	}
	p.RUnlock()
}

func (p *fastSlowDiscoverPeriod) trans(d time.Duration) {
	p.RLock()
	for _, c := range p.cs {
		c <- d
	}
	p.RUnlock()
}
