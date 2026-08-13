package async

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/logutils"
)

type Command func(context.Context) error

type Commander interface {
	Command(inteval ...time.Duration) Command
}

type Runner interface {
	Run(context.Context) error
}

// SingleShotCommand runs once.
type SingleShotCommand struct {
	Interval time.Duration
	Init     func(context.Context) error
	Runable  func(context.Context) error
}

// FiniteCommand terminates when error is nil.
type FiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

// InfiniteCommand runs until context is closed.
type InfiniteCommand struct {
	Interval time.Duration
	Runable  func(context.Context) error
}

func NewGroup(parent context.Context) *Group {
	ctx, cancel := context.WithCancel(parent)
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
		defer common.LogOnPanic()
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

func (g *Group) WaitAsync() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer common.LogOnPanic()
		g.Wait()
		close(ch)
	}()
	return ch
}

func NewAtomicGroup(parent context.Context) *AtomicGroup {
	ctx, cancel := context.WithCancel(parent)
	ag := &AtomicGroup{ctx: ctx, cancel: cancel}
	ag.done = ag.onFinish
	return ag
}

// AtomicGroup terminates as soon as first goroutine terminates with error.
type AtomicGroup struct {
	ctx    context.Context
	cancel func()
	done   func()
	wg     sync.WaitGroup

	mu    sync.Mutex
	error error
}

type AtomicGroupKey string

func (d *AtomicGroup) Name() string {
	val := d.ctx.Value(AtomicGroupKey("name"))
	if val != nil {
		return val.(string)
	}
	return ""
}

// Go spawns function in a goroutine and stores results or errors.
func (d *AtomicGroup) Add(cmd Command) {
	d.wg.Add(1)
	go func() {
		defer common.LogOnPanic()
		defer d.done()
		err := cmd(d.ctx)
		d.mu.Lock()
		defer d.mu.Unlock()
		if err != nil {
			// do not overwrite original error by context errors
			if d.error != nil {
				logutils.ZapLogger().Info("async.Command failed",
					zap.String("group", d.Name()),
					zap.NamedError("error", err),
					zap.NamedError("d.error", d.error),
				)
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

// Error stores an error that was reported by any of the downloader. Should be called after Wait.
func (d *AtomicGroup) Error() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.error
}

func (d *AtomicGroup) onFinish() {
	d.wg.Done()
}

type QueuedAtomicGroup struct {
	*AtomicGroup
	limit       uint32
	count       uint32
	pendingCmds []Command
	mu          sync.Mutex
}

type ErrorCounter struct {
	cnt       int
	maxErrors int
	err       error
	msg       string
}

type FiniteCommandWithErrorCounter struct {
	FiniteCommand
	*ErrorCounter
}
