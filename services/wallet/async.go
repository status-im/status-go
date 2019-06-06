package wallet

import (
	"context"
	"sync"
	"time"
)

type Command interface {
	Run(context.Context)
}

type FiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

func (c FiniteCommand) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.Runable(ctx)
			if err == nil {
				return
			}
		}
	}
}

type InfiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

func (c InfiniteCommand) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.Runable(ctx)
		}
	}
}

func NewGroup() *Group {
	ctx, cancel := context.WithCancel(context.Background())
	return &Group{
		ctx:    ctx,
		cancel: cancel,
	}
}

type Group struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
}

func (g *Group) Add(cmd Command) {
	g.wg.Add(1)
	go func() {
		cmd.Run(g.ctx)
		g.wg.Done()
	}()
}

func (g *Group) Stop() {
	g.cancel()
	g.wg.Wait()
}
