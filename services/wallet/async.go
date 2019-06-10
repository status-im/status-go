package wallet

import (
	"context"
	"sync"
	"time"
)

type Command func(context.Context) error

// FiniteCommand terminates when error is nil.
type FiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

func (c FiniteCommand) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.Interval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := c.Runable(ctx)
			if err == nil {
				return nil
			}
		}
	}
}

// InfiniteCommand runs until context is closed.
type InfiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

func (c InfiniteCommand) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.Interval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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
		_ = cmd(g.ctx)
		g.wg.Done()
	}()
}

func (g *Group) Stop() {
	g.cancel()
}

func (g *Group) Wait() {
	g.wg.Wait()
}

func NewAtomicGroup(parent context.Context) *AtomicGroup {
	ctx, cancel := context.WithCancel(parent)
	return &AtomicGroup{ctx: ctx, cancel: cancel}
}

// AtomicGroup terminates as soon as first goroutine terminates..
type AtomicGroup struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup

	mu    sync.Mutex
	error error
}

// Go spawns function in a goroutine and stores results or errors.
func (d *AtomicGroup) Add(cmd Command) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		err := cmd(d.ctx)
		d.mu.Lock()
		defer d.mu.Unlock()
		if err != nil {
			// do not overwrite original error by context errors
			if d.error != nil {
				return
			}
			d.error = err
			d.cancel()
			return
		}
	}()
}

// Wait for all downloaders to finish.
func (d *AtomicGroup) Wait() {
	d.wg.Wait()
	if d.Error() == nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.cancel()
	}
}

func (d *AtomicGroup) WaitAsync() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		d.Wait()
		close(ch)
	}()
	return ch
}

// Error stores an error that was reported by any of the downloader. Should be called after Wait.
func (d *AtomicGroup) Error() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.error
}

func (d *AtomicGroup) Stop() {
	d.cancel()
}
