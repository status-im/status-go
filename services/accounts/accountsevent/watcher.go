package accountsevent

import (
	"context"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/async"
)

type AccountsChangeCb func(changedAddresses []common.Address, eventType EventType, currentAddresses []common.Address)

// Watcher executes a given callback whenever an account gets added/removed
type Watcher struct {
	accountsDB  *accounts.Database
	accountFeed *event.Feed
	group       *async.Group
	callback    AccountsChangeCb
}

func NewWatcher(accountsDB *accounts.Database, accountFeed *event.Feed, callback AccountsChangeCb) *Watcher {
	return &Watcher{
		accountsDB:  accountsDB,
		accountFeed: accountFeed,
		callback:    callback,
	}
}

func (w *Watcher) Start() {
	if w.group != nil {
		return
	}

	w.group = async.NewGroup(context.Background())
	w.group.Add(func(ctx context.Context) error {
		return watch(ctx, w.accountsDB, w.accountFeed, w.callback)
	})
}

func (w *Watcher) Stop() {
	if w.group != nil {
		w.group.Stop()
		w.group.Wait()
		w.group = nil
	}
}

func onAccountsChange(accountsDB *accounts.Database, callback AccountsChangeCb, changedAddresses []common.Address, eventType EventType) {
	currentEthAddresses, err := accountsDB.GetWalletAddresses()

	if err != nil {
		logutils.ZapLogger().Error("failed getting wallet addresses", zap.Error(err))
		return
	}

	currentAddresses := make([]common.Address, 0, len(currentEthAddresses))
	for _, ethAddress := range currentEthAddresses {
		currentAddresses = append(currentAddresses, common.Address(ethAddress))
	}

	if callback != nil {
		callback(changedAddresses, eventType, currentAddresses)
	}
}

func watch(ctx context.Context, accountsDB *accounts.Database, accountFeed *event.Feed, callback AccountsChangeCb) error {
	ch := make(chan Event, 1)
	sub := accountFeed.Subscribe(ch)
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
			onAccountsChange(accountsDB, callback, ev.Accounts, ev.Type)
		}
	}
}
