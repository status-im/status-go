package walletconnectevent

import (
	"context"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/walletconnect"
)

type WalletConnectsChangeCb func(eventType EventType, Session walletconnect.Session)

// Watcher executes a given callback whenever an walletConnect gets added/removed
type Watcher struct {
	walletConnectFeed *event.Feed
	group             *async.Group
	callback          WalletConnectsChangeCb
}

func NewWatcher(walletConnectFeed *event.Feed, callback WalletConnectsChangeCb) *Watcher {
	return &Watcher{
		walletConnectFeed: walletConnectFeed,
		callback:          callback,
	}
}

func (w *Watcher) Start() {
	logutils.ZapLogger().Debug("Starting wallet connect connection watcher")
	if w.group != nil {
		return
	}

	w.group = async.NewGroup(context.Background())
	w.group.Add(func(ctx context.Context) error {
		return watch(ctx, w.walletConnectFeed, w.callback)
	})
}

func (w *Watcher) Stop() {
	if w.group != nil {
		w.group.Stop()
		w.group.Wait()
		w.group = nil
	}
}

func onWalletConnectSessionChange(callback WalletConnectsChangeCb, session walletconnect.Session, eventType EventType) {
	if callback != nil {
		callback(eventType, session)
	}
}

func watch(ctx context.Context, walletConnectFeed *event.Feed, callback WalletConnectsChangeCb) error {
	ch := make(chan Event, 1)
	sub := walletConnectFeed.Subscribe(ch)
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			if err != nil {
				logutils.ZapLogger().Error("accounts watcher subscription failed", zap.Error(err))
			}
		case ev := <-ch:
			onWalletConnectSessionChange(callback, ev.Session, ev.Type)
		}
	}
}
