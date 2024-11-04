package activitycenter

import (
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/services/wallet/walletconnect"
	"github.com/status-im/status-go/services/wallet/walletconnect/walletconnectevent"
)

type Controller struct {
	messenger            *protocol.Messenger
	walletConnectsFeed   *event.Feed
	walletConnectWatcher *walletconnectevent.Watcher
	commandsLock         sync.RWMutex
}

func NewController(walletConnectsFeed *event.Feed) *Controller {
	return &Controller{
		walletConnectsFeed: walletConnectsFeed,
	}
}

func (c *Controller) Start(messenger *protocol.Messenger) {
	c.messenger = messenger
	c.startWalletConnectConnectionWatcher()
}

func (c *Controller) Stop() {
	c.stopWalletConnectConnectionWatcher()
}

func (c *Controller) startWalletConnectConnectionWatcher() {
	logutils.ZapLogger().Debug("Starting wallet connect connection watcher")
	if c.walletConnectWatcher != nil {
		return
	}

	walletConnectChangeCb := func(eventType walletconnectevent.EventType, session walletconnect.Session) {
		c.commandsLock.Lock()
		defer c.commandsLock.Unlock()
		// Whenever an dApp gets added, add it to activity center
		if eventType == walletconnectevent.EventTypeAdded {
			err := c.createNewSessionActivityCenterNotification(&session)
			if err != nil {
				logutils.ZapLogger().Error("Failed to create new session activity center notification", zap.Error(err))
			}

		}
	}

	c.walletConnectWatcher = walletconnectevent.NewWatcher(c.walletConnectsFeed, walletConnectChangeCb)

	c.walletConnectWatcher.Start()
}

func (c *Controller) stopWalletConnectConnectionWatcher() {
	if c.walletConnectWatcher != nil {
		c.walletConnectWatcher.Stop()
		c.walletConnectWatcher = nil
	}
}

func (c *Controller) createNewSessionActivityCenterNotification(session *walletconnect.Session) error {
	now := c.messenger.GetCurrentTimeInMillis()

	logutils.ZapLogger().Info("Creating new session activity center notification", zap.Any("session", session))

	notification := &protocol.ActivityCenterNotification{
		ID:                         types.FromHex(string(session.Topic) + "_dapp_connected"),
		Type:                       protocol.ActivityCenterNotificationTypeDAppConnected,
		DAppURL:                    session.Peer.Metadata.URL,
		DAppName:                   session.Peer.Metadata.Name,
		WalletProviderSessionTopic: string(session.Topic),
		Timestamp:                  now,
		UpdatedAt:                  now,
	}

	if len(session.Peer.Metadata.Icons) > 0 {
		notification.DAppIconURL = session.Peer.Metadata.Icons[0]
	}

	return c.messenger.AddNotificationToActivityCenter(notification)
}
