package node

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/timesource"
	nodeadapters "github.com/status-im/status-go/pkg/backend/node/adapters"
	"github.com/status-im/status-go/pkg/featureflags"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/eth"
	"github.com/status-im/status-go/services/linkpreview"
	"github.com/status-im/status-go/services/newsfeed"
	"github.com/status-im/status-go/services/sharedurls"

	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
	accountssvc "github.com/status-im/status-go/services/accounts"
	appgeneral "github.com/status-im/status-go/services/app-general"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/chat"
	"github.com/status-im/status-go/services/communitytokens"
	"github.com/status-im/status-go/services/connector"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/gif"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/permissions"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcstats"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/updates"
	"github.com/status-im/status-go/services/wakuv2ext"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

var (
	// ErrWakuClearIdentitiesFailure clearing whisper identities has failed.
	ErrWakuClearIdentitiesFailure = errors.New("failed to clear waku identities")
	// ErrRPCClientUnavailable is returned if an RPC client can't be retrieved.
	// This is a normal situation when a node is stopped.
	ErrRPCClientUnavailable = errors.New("JSON-RPC client is unavailable")
)

func (b *StatusNode) ThirdpartyServicesEnabled() (bool, error) {
	accDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		b.logger.Error("failed to create accounts db", zap.Error(err))
		return true, err
	}
	enabled, err := accDB.ThirdpartyServicesEnabled()
	if err != nil {
		return true, err
	}
	return enabled, nil
}

func (b *StatusNode) initServices(config *params.NodeConfig, mediaServer *server.MediaServer) error {
	accDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	services := []common.StatusService{}
	services = append(services, b.rpcStatsService())
	services = append(services, b.appgeneralService())
	services = append(services, b.personalService())
	services = append(services, b.pendingTrackerService(&b.walletFeed))
	services = append(services, b.ensService(b.timeSourceNow()))
	services = append(services, b.CommunityTokensService())
	services = append(services, b.stickersService(accDB))
	services = append(services, b.updatesService())
	services = appendIf(b.appDB != nil && b.multiaccountsDB != nil, services, b.accountsService(accDB, mediaServer))
	services = appendIf(config.BrowsersConfig.Enabled, services, b.browsersService())
	services = appendIf(config.PermissionsConfig.Enabled, services, b.permissionsService())
	services = appendIf(config.ConnectorConfig.Enabled, services, b.connectorService())
	services = append(services, b.gifService(accDB))
	services = append(services, b.ChatService(accDB))
	services = append(services, b.ethService())

	// Wallet Service is used by wakuExtSrvc/wakuV2ExtSrvc
	// Keep this initialization before the other two
	if config.WalletConfig.Enabled {
		err := b.createWalletService(accDB, b.appDB, b.accountsPublisher, &b.walletFeed, config.WalletConfig.StatusProxyStageName)
		if err != nil {
			return err
		}
		services = append(services, b.walletSrvc)
	}

	// CollectiblesManager needs the WakuExt service to get metadata for
	// Community collectibles.
	// Messenger needs the CollectiblesManager to get the list of collectibles owned
	// by a certain account and check community entry permissions.
	// We handle circular dependency between the two by delaying ininitalization of the CommunityCollectibleInfoProvider
	// in the CollectiblesManager.
	if config.WakuV2Config.Enabled {
		wakuext, err := b.wakuV2ExtService(config)
		if err != nil {
			return err
		}

		b.wakuV2ExtSrvc = wakuext

		services = append(services, wakuext)

		b.SetWalletCommunityInfoProvider(wakuext)
	}

	// We ignore for now local notifications flag as users who are upgrading have no mean to enable it
	lns, err := b.localNotificationsService(config.NetworkID)
	if err != nil {
		return err
	}
	services = append(services, lns)

	if b.NewsFeedService() != nil {
		services = append(services, b.NewsFeedService())
	}
	services = append(services, b.sharedUrlsService())
	services = append(services, b.linkPreviewService(accDB))

	b.services = services

	return nil
}

func (b *StatusNode) runServicesMigrations() error {
	err := newsfeed.SQLiteMigrate(b.appDB)
	if err != nil {
		return err
	}

	return nil
}

func (b *StatusNode) registerService(s common.StatusService) error {
	for _, api := range s.APIs() {
		b.logger.Debug("registering service api", zap.String("namespace", api.Namespace))
		err := b.rpcServer.RegisterName(api.Namespace, api.Service)
		if err != nil {
			b.logger.Error("Failed to register API", zap.String("namespace", api.Namespace), zap.Error(err))
			return err
		}
	}

	return nil
}

func (b *StatusNode) wakuV2ExtService(config *params.NodeConfig) (*wakuv2ext.Service, error) {
	if b.wakuV2ExtSrvc == nil {
		b.wakuV2ExtSrvc = wakuv2ext.New(*config, b.rpcClient, b.logger.Named("protocol"))
	}

	return b.wakuV2ExtSrvc, nil
}

func (b *StatusNode) AccountService() *accountssvc.Service {
	return b.accountsSrvc
}

func (b *StatusNode) BrowserService() *browsers.Service {
	return b.browsersSrvc
}

func (b *StatusNode) EnsService() *ens.Service {
	return b.ensSrvc
}

func (b *StatusNode) WakuV2ExtService() *wakuv2ext.Service {
	return b.wakuV2ExtSrvc
}

func (b *StatusNode) connectorService() *connector.Service {
	if b.connectorSrvc == nil {
		logger := b.logger.Named("connector")

		b.connectorSrvc = connector.NewService(
			logger,
			b.walletDB,
			b.rpcClient,
			fees.NewFeeManager(b.rpcClient, logger.Named("feeManager")),
			b.rpcClient.GetNetworkManager(),
			&connector.Config{
				WSHost:    b.config.WSHost,
				WSPort:    b.config.WSPort,
				ProjectID: b.config.WalletConnectProjectID,
			},
		)
	}
	return b.connectorSrvc
}

func (b *StatusNode) rpcStatsService() *rpcstats.Service {
	if b.rpcStatsSrvc == nil {
		b.rpcStatsSrvc = rpcstats.New()
	}

	return b.rpcStatsSrvc
}

func (b *StatusNode) accountsService(accDB *accounts.Database, mediaServer *server.MediaServer) *accountssvc.Service {
	if b.accountsSrvc == nil {
		b.accountsSrvc = accountssvc.NewService(
			accDB,
			b.multiaccountsDB,
			b.gethAccountsManager,
			b.config,
			b.accountsPublisher,
			mediaServer,
			b.logger.Named("AccountsService"),
		)
	}

	return b.accountsSrvc
}

func (b *StatusNode) browsersService() *browsers.Service {
	if b.browsersSrvc == nil {
		b.browsersSrvc = browsers.NewService(browsers.NewDB(b.appDB))
	}
	return b.browsersSrvc
}

func (b *StatusNode) ensService(timesource func() time.Time) *ens.Service {
	if b.ensSrvc == nil {
		b.ensSrvc = ens.NewService(b.rpcClient, b.gethAccountsManager, b.pendingTracker, b.config, b.appDB, timesource)
	}
	return b.ensSrvc
}

func (b *StatusNode) pendingTrackerService(walletFeed *event.Feed) *pendingtxtracker.PendingTxTracker {
	if b.pendingTracker == nil {
		b.pendingTracker = pendingtxtracker.NewPendingTxTracker(b.walletDB, pendingtxtracker.NewBatchTxStatusFetcher(b.rpcClient, b.logger.Named("PendingTxTracker")), walletFeed, pendingtxtracker.PendingCheckInterval)
		if b.transactor != nil {
			b.transactor.SetPendingTracker(b.pendingTracker)
		}
	}
	return b.pendingTracker
}

func (b *StatusNode) CommunityTokensService() *communitytokens.Service {
	if b.communityTokensSrvc == nil {
		b.communityTokensSrvc = communitytokens.NewService(b.rpcClient, b.gethAccountsManager, b.config, b.appDB, &b.walletFeed, b.transactor)
	}
	return b.communityTokensSrvc
}

func (b *StatusNode) stickersService(accountDB *accounts.Database) *stickers.Service {
	if b.stickersSrvc == nil {
		b.stickersSrvc = stickers.NewService(accountDB, b.rpcClient, b.gethAccountsManager, b.config, b.downloader, b.mediaServer, b.pendingTracker)
	}
	return b.stickersSrvc
}

func (b *StatusNode) updatesService() *updates.Service {
	if b.updatesSrvc == nil {
		b.updatesSrvc = updates.NewService(b.ensService(b.timeSourceNow()))
	}

	return b.updatesSrvc
}

func (b *StatusNode) gifService(accountsDB *accounts.Database) *gif.Service {
	if b.gifSrvc == nil {
		b.gifSrvc = gif.NewService(accountsDB)
	}
	return b.gifSrvc
}

func (b *StatusNode) ChatService(accountsDB *accounts.Database) *chat.Service {
	if b.chatSrvc == nil {
		b.chatSrvc = chat.NewService(accountsDB)
	}
	return b.chatSrvc
}

func (b *StatusNode) permissionsService() *permissions.Service {
	if b.permissionsSrvc == nil {
		b.permissionsSrvc = permissions.NewService(permissions.NewDB(b.appDB))
	}
	return b.permissionsSrvc
}

func (b *StatusNode) appgeneralService() *appgeneral.Service {
	if b.appGeneralSrvc == nil {
		b.appGeneralSrvc = appgeneral.New()
	}
	return b.appGeneralSrvc
}

func (b *StatusNode) WalletService() *wallet.Service {
	return b.walletSrvc
}

func (b *StatusNode) AccountsPublisher() *pubsub.Publisher {
	return b.accountsPublisher
}

func (b *StatusNode) SetWalletCommunityInfoProvider(provider thirdparty.CommunityInfoProvider) {
	if b.walletSrvc != nil {
		b.walletSrvc.SetWalletCommunityInfoProvider(provider)
	}
}

func (b *StatusNode) createWalletService(accountsDB *accounts.Database, appDB *sql.DB, accountsPublisher *pubsub.Publisher, walletFeed *event.Feed, statusProxyStageName string) (err error) {
	if b.walletSrvc == nil {
		b.walletSrvc, err = wallet.NewService(
			b.walletDB, accountsDB, appDB, b.rpcClient, accountsPublisher, b.gethAccountsManager, b.transactor, b.config,
			b.ensService(b.timeSourceNow()).API().EnsResolver(),
			b.pendingTracker,
			walletFeed,
			b.mediaServer,
			b.tokenManager,
			statusProxyStageName,
		)
	}
	return
}

func (b *StatusNode) ethService() *eth.Service {
	if b.ethSrvc == nil {
		b.ethSrvc = eth.NewService(b.rpcClient, b.gethAccountsManager)
	}
	return b.ethSrvc
}

func (b *StatusNode) sharedUrlsService() *sharedurls.Service {
	if b.sharedUrlsSrvc == nil {
		b.sharedUrlsSrvc = sharedurls.NewService(nil)
		if extService := b.WakuV2ExtService(); extService != nil {
			provider := nodeadapters.NewSharedUrlsMessengerAdapter(extService.Messenger())
			b.sharedUrlsSrvc.SetDataProvider(provider)
		}
	}
	return b.sharedUrlsSrvc
}

func (b *StatusNode) SharedUrlsService() *sharedurls.Service {
	return b.sharedUrlsSrvc
}

func (b *StatusNode) linkPreviewService(accDB *accounts.Database) *linkpreview.Service {
	if b.linkPreviewSrvc == nil {
		settingsProvider := nodeadapters.NewLinkPreviewSettingsAdapter(accDB)
		b.linkPreviewSrvc = linkpreview.NewService(b.logger.Named("linkpreview"), settingsProvider, nil)
	}
	return b.linkPreviewSrvc
}

func (b *StatusNode) LinkPreviewService() *linkpreview.Service {
	return b.linkPreviewSrvc
}

func (b *StatusNode) localNotificationsService(network uint64) (*localnotifications.Service, error) {
	var err error
	if b.localNotificationsSrvc == nil {
		b.localNotificationsSrvc, err = localnotifications.NewService(b.appDB)
		if err != nil {
			return nil, err
		}
	}
	return b.localNotificationsSrvc, nil
}

func appendIf(condition bool, services []common.StatusService, service common.StatusService) []common.StatusService {
	if !condition {
		return services
	}
	return append(services, service)
}

func (b *StatusNode) PendingTracker() *pendingtxtracker.PendingTxTracker {
	return b.pendingTracker
}

func (b *StatusNode) StopLocalNotifications() error {
	if b.localNotificationsSrvc == nil {
		return nil
	}

	if b.localNotificationsSrvc.IsStarted() {
		err := b.localNotificationsSrvc.Stop()
		if err != nil {
			b.logger.Error("LocalNotifications service stop failed on StopLocalNotifications", zap.Error(err))
			return nil
		}
	}

	return nil
}

func (b *StatusNode) StartLocalNotifications() error {
	if b.localNotificationsSrvc == nil {
		return nil
	}

	if b.walletSrvc == nil {
		return nil
	}

	if !b.localNotificationsSrvc.IsStarted() {
		err := b.localNotificationsSrvc.Start()

		if err != nil {
			b.logger.Error("LocalNotifications service start failed on StartLocalNotifications", zap.Error(err))
			return nil
		}
	}

	return nil
}

func (b *StatusNode) personalService() *personal.Service {
	if b.personalSrvc == nil {
		b.personalSrvc = personal.New()
	}
	return b.personalSrvc
}

func (b *StatusNode) TimeSource() timesource.Provider {
	if b.timeSourceSrvc == nil {
		thirdpartyServicesEnabled, err := b.ThirdpartyServicesEnabled()
		if err != nil {
			b.logger.Error("failed to get if thirdparty services enabled", zap.Error(err))
		}

		if thirdpartyServicesEnabled {
			b.timeSourceSrvc = timesource.DefaultService()
		} else {
			b.timeSourceSrvc = timesource.LocalService()
		}
	}
	return b.timeSourceSrvc
}

func (b *StatusNode) timeSourceNow() func() time.Time {
	return b.TimeSource().Now
}

func (b *StatusNode) NewsFeedService() *newsfeed.Service {
	thirdpartyServicesEnabled, err := b.ThirdpartyServicesEnabled()
	if err != nil {
		b.logger.Error("failed to get if thirdparty services are enabled", zap.Error(err))
		return nil
	}

	if !featureflags.EnableNewsFeed || !thirdpartyServicesEnabled {
		return nil
	}

	if b.newsfeedSrvc == nil {
		persistence := newsfeed.NewSQLitePersistence(b.appDB)

		b.newsfeedSrvc = newsfeed.NewService(
			b.logger.Named("newsfeed"),
			persistence,
			nil,
		)

		if wakuext := b.WakuV2ExtService(); wakuext != nil && wakuext.Messenger() != nil {
			ac := nodeadapters.NewNewsFeedActivityCenterAdapter(wakuext.Messenger())
			b.newsfeedSrvc.SetActivityCenter(ac)
		}
	}
	return b.newsfeedSrvc
}
