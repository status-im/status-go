package networksevent

import (
	"context"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
)

type EventCallbacks struct {
	ActiveNetworksChangeCb ActiveNetworksChangeCb
}

type ActiveNetworksChangeCb func()

// Watcher executes a given callback whenever a network event is emitted
type Watcher struct {
	accountFeed *event.Feed
	group       *async.Group
	callbacks   EventCallbacks
}

func NewWatcher(accountFeed *event.Feed, callbacks EventCallbacks) *Watcher {
	return &Watcher{
		accountFeed: accountFeed,
		callbacks:   callbacks,
	}
}

func (w *Watcher) Start() {
	if w.group != nil {
		return
	}

	w.group = async.NewGroup(context.Background())
	w.group.Add(func(ctx context.Context) error {
		return watch(ctx, w.accountFeed, w.callbacks)
	})
}

func (w *Watcher) Stop() {
	if w.group != nil {
		w.group.Stop()
		w.group.Wait()
		w.group = nil
	}
}

func onActiveNetworksChange(callback ActiveNetworksChangeCb) {
	if callback == nil {
		return
	}

	callback()
}

func watch(ctx context.Context, accountFeed *event.Feed, callbacks EventCallbacks) error {
	ch := make(chan Event, 1)
	sub := accountFeed.Subscribe(ch)
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			if err != nil {
				logutils.ZapLogger().Error("network events watcher subscription failed", zap.Error(err))
			}
		case ev := <-ch:
			switch ev.Type {
			case EventTypeActiveNetworksChanged:
				onActiveNetworksChange(callbacks.ActiveNetworksChangeCb)
			}
		}
	}

}
