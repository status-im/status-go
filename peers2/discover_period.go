package peers2

import (
	"time"
)

type fastSlowDiscoverPeriod struct {
	slow time.Duration
	fast time.Duration

	ch chan time.Duration
}

func newFastSlowDiscoverPeriod(slow, fast time.Duration) *fastSlowDiscoverPeriod {
	ch := make(chan time.Duration, 1)
	ch <- fast

	return &fastSlowDiscoverPeriod{
		slow: slow,
		fast: fast,
		ch:   ch,
	}
}

func (p *fastSlowDiscoverPeriod) channel() chan time.Duration {
	return p.ch
}

func (p *fastSlowDiscoverPeriod) transSlow() {
	p.ch <- p.slow
}

func (p *fastSlowDiscoverPeriod) transFast() {
	p.ch <- p.fast
}
