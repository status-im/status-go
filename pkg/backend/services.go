package backend

import (
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/timesource"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node/adapters"
	"github.com/status-im/status-go/pkg/featureflags"
	accountssvc "github.com/status-im/status-go/services/accounts"
	appgeneral "github.com/status-im/status-go/services/app-general"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/chat"
	"github.com/status-im/status-go/services/communitytokens"
	"github.com/status-im/status-go/services/connector"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/ens/ensresolver"
	"github.com/status-im/status-go/services/eth"
	wakuext "github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/services/gif"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/media"
	"github.com/status-im/status-go/services/newsfeed"
	"github.com/status-im/status-go/services/permissions"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcstats"
	"github.com/status-im/status-go/services/sharedurls"
	"github.com/status-im/status-go/services/status"
	"github.com/status-im/status-go/services/stickers"
	"github.com/status-im/status-go/services/updates"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/tokenbalances"
)

type StatusBackendService interface {
	Start() error
	Stop() error
	//API() interface{}
	APIs() []gethrpc.API // TODO: We don't need gethrpc API
	//Metrics() // TODO: Prometheus metrics
}

// services groups all services in a single place.
// At the moment, the idea is simply not to pollute StatusBackend. For now services have direct access to StatusBackend instance and vice versa.
// FIXME: Ideally, backend pointer should not be needed here. Instead, services should refer to each other with provider interfaces.
type services struct {
	backend *StatusBackend

	// lgoger is used to derive a logger for all services. And for logging.
	logger *zap.Logger

	//backend                *API
	mediaService           *media.Service
	rpcStatsSrvc           *rpcstats.Service
	statusPublicSrvc       *status.Service
	accountsSrvc           *accountssvc.Service
	browsersSrvc           *browsers.Service
	permissionsSrvc        *permissions.Service
	walletSrvc             *wallet.Service
	localNotificationsSrvc *localnotifications.Service
	personalSrvc           *personal.Service
	timeSourceSrvc         timesource.Service
	wakuV2ExtSrvc          *wakuext.Service
	ensSrvc                *ens.Service
	communityTokensSrvc    *communitytokens.Service
	gifSrvc                *gif.Service
	stickersSrvc           *stickers.Service
	chatSrvc               *chat.Service
	updatesSrvc            *updates.Service
	pendingTracker         *pendingtxtracker.PendingTxTracker
	connectorSrvc          *connector.Service
	appGeneralSrvc         *appgeneral.Service
	ethSrvc                *eth.Service
	newsfeedSrvc           *newsfeed.Service
	sharedUrlsSrvc         *sharedurls.Service

	// idleServices is a list of created, but not yet started services
	// After starting, services are removed from this list.
	idleServices []StatusBackendService
}

func newServices(backend *StatusBackend) *services {
	return &services{
		backend: backend,
		logger:  backend.logger.Named("services"),
	}
}

// All returns all created services. The order is fixed, but might change between versions.
func (s *services) All() []StatusBackendService {
	var out []StatusBackendService

	all := []StatusBackendService{
		s.mediaService,
		s.rpcStatsSrvc,
		s.statusPublicSrvc,
		s.accountsSrvc,
		s.browsersSrvc,
		s.permissionsSrvc,
		s.walletSrvc,
		s.localNotificationsSrvc,
		s.personalSrvc,
		s.wakuV2ExtSrvc,
		s.ensSrvc,
		s.communityTokensSrvc,
		s.gifSrvc,
		s.stickersSrvc,
		s.chatSrvc,
		s.updatesSrvc,
		s.pendingTracker,
		s.connectorSrvc,
		s.appGeneralSrvc,
		s.ethSrvc,
		s.newsfeedSrvc,
		s.sharedUrlsSrvc,
	}

	for _, service := range all {
		if !common.IsNil(service) {
			out = append(out, service)
		}
	}

	return out
}

func (s *services) Register() error {
	for _, service := range s.idleServices {
		for _, api := range service.APIs() {
			err := s.backend.rpcServer.RegisterName(api.Namespace, api.Service)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Start starts all created, but still idle (not started) services.
// If any of the services failed to start, operation is interrupted and an error is returned.
func (s *services) Start() error {
	for _, service := range s.idleServices {
		err := service.Start()
		if err != nil {
			return errors.Wrap(err, "failed to start service")
		}
	}

	s.idleServices = []StatusBackendService{}

	return nil
}

func (s *services) Stop() {
	for _, service := range s.idleServices {
		err := service.Stop()
		if err != nil {
			s.logger.Warn("failed to stop service", zap.Error(err))
		}
	}

	// All services can be destroyed now
	s.mediaService = nil
	s.rpcStatsSrvc = nil
	s.statusPublicSrvc = nil
	s.accountsSrvc = nil
	s.browsersSrvc = nil
	s.permissionsSrvc = nil
	s.walletSrvc = nil
	s.localNotificationsSrvc = nil
	s.personalSrvc = nil
	s.timeSourceSrvc = nil
	s.wakuV2ExtSrvc = nil
	s.ensSrvc = nil
	s.communityTokensSrvc = nil
	s.gifSrvc = nil
	s.stickersSrvc = nil
	s.chatSrvc = nil
	s.updatesSrvc = nil
	s.pendingTracker = nil
	s.connectorSrvc = nil
	s.appGeneralSrvc = nil
	s.ethSrvc = nil
	s.newsfeedSrvc = nil
	s.sharedUrlsSrvc = nil
}

func (s *services) requireWakuextService() {
	if s.wakuV2ExtSrvc == nil {
		panic("attempted to create a service that requires WakuV2Ext")
	}
	if s.wakuV2ExtSrvc.Messenger() == nil {
		panic("wakuV2Ext service initialized, but messenger is nil")
	}
}

func (s *services) requireWalletService() {
	if s.walletSrvc == nil {
		panic("attempted to create a service that requires wallet service")
	}
}

func (s *services) createMedia(address, advertizeHost string, advertizePort int) error {
	b := s.backend

	if s.mediaService != nil {
		// Media service should only be spawned once
		return errors.New("media service is already created")
	}

	opts := []media.Option{
		media.WithLogger(b.logger.Named("media-server")),
		media.WithDisableTLS(false),
		media.WithServerAddress(address),
		media.WithServerAdvertizeAddress(advertizeHost, advertizePort),
	}

	mediaService, err := media.NewService(nil, b.ipfs, b.multiaccountsDB, nil, opts...)
	if err != nil {
		return err
	}
	//mediaService.SetDataProviders(b.ActiveAccount.appDB, b.ActiveAccount.walletDB, b.ipfs)

	s.mediaService = mediaService
	s.addService(s.mediaService)

	return nil
}

func (s *services) addService(service StatusBackendService) {
	s.idleServices = append(s.idleServices, service)
}

func (s *services) media() *media.Service {
	return s.mediaService
}

func (s *services) createRPCStats() {
	s.rpcStatsSrvc = rpcstats.New()
	s.addService(s.rpcStatsSrvc)
}

func (s *services) createStatusPublicService() {
	s.requireWakuextService()

	s.statusPublicSrvc = status.New()
	s.statusPublicSrvc.Init(s.wakuV2ExtSrvc.Messenger())
	s.addService(s.statusPublicSrvc)
}

func (s *services) createAccountsService() {
	s.requireWakuextService()

	if s.mediaService == nil {
		s.logger.Warn("creating accounts service without media service")
	}

	s.accountsSrvc = accountssvc.NewService(
		s.backend.activeAccount.accountsDB,
		s.backend.multiaccountsDB,
		s.backend.activeAccount.accsManager,
		s.mediaService,
		s.logger.Named("accounts"),
	)
	s.accountsSrvc.Init(s.wakuV2ExtSrvc.Messenger(), s.backend.activeAccount.account)
	s.addService(s.accountsSrvc)
}

func (s *services) createBrowsersService() {
	db := browsers.NewDB(s.backend.activeAccount.appDB)
	s.browsersSrvc = browsers.NewService(db)
	s.addService(s.browsersSrvc)
}

func (s *services) createPermissionsService() {
	db := permissions.NewDB(s.backend.activeAccount.appDB)
	s.permissionsSrvc = permissions.NewService(db)
}

func (s *services) createWalletService() {
	if s.mediaService == nil {
		s.logger.Warn("creating wallet service without media service")
	}

	var ensResolver *ensresolver.EnsResolver
	if s.ensSrvc != nil {
		ensResolver = s.ensSrvc.API().EnsResolver()
	}

	s.walletSrvc = wallet.NewService(
		s.backend.activeAccount.walletDB,
		s.backend.activeAccount.accountsDB,
		s.backend.rpcClient,
		s.accountsSrvc.Publisher(),
		s.backend.activeAccount.accsManager,
		s.backend.transactor,
		s.backend.activeAccount.nodeConfig,
		ensResolver,
		&s.backend.walletFeed,
		s.mediaService,
		s.backend.tokenManager,
		s.backend.activeAccount.nodeConfig.WalletConfig.StatusProxyStageName,
	)
	s.addService(s.walletSrvc)

	if s.wakuV2ExtSrvc != nil {
		s.walletSrvc.SetWalletCommunityInfoProvider(s.wakuV2ExtSrvc)
	}
}

func (s *services) createLocalNotificationsService() {
	db := s.backend.activeAccount.appDB
	s.localNotificationsSrvc = localnotifications.NewService(db)
	s.addService(s.localNotificationsSrvc)
}

func (s *services) createPersonalService() {
	s.personalSrvc = personal.New()
	s.addService(s.personalSrvc)
}

func (s *services) createTimeSource() {
	const privateMode = false // FIXME: This is temporal and will be replaced with the actual setting
	if privateMode {
		s.timeSourceSrvc = timesource.LocalService()
	} else {
		s.timeSourceSrvc = timesource.DefaultService()
	}
}

func (s *services) createWakuExtService() error {
	s.wakuV2ExtSrvc = wakuext.New(*s.backend.activeAccount.nodeConfig, s.backend.rpcClient, s.logger.Named("protocol"))
	s.addService(s.wakuV2ExtSrvc)

	activeAccount := s.backend.activeAccount
	chatAccount, err := activeAccount.accsManager.SelectedChatAccount()
	if err != nil {
		return errors.Wrap(err, "failed to get selected chat account")
	}

	var tokenBalancesFetcher *tokenbalances.Fetcher
	var tokenBalancesStorage tokenbalances.Storage

	if s.walletSrvc != nil {
		s.walletSrvc.SetWalletCommunityInfoProvider(s.wakuV2ExtSrvc)
		tokenBalancesFetcher = s.walletSrvc.GetTokenBalancesFetcher().(*tokenbalances.Fetcher)
		tokenBalancesStorage = s.walletSrvc.GetTokenBalancesStorage()
	}

	params := wakuext.InitProtocolParams{
		Identity:               chatAccount.PrivateKey(),
		AppDB:                  activeAccount.appDB,
		WalletDB:               activeAccount.walletDB,
		HTTPServer:             s.mediaService,
		MultiAccountDB:         s.backend.multiaccountsDB,
		Account:                activeAccount.account,
		AccountsManager:        activeAccount.accsManager,
		RPCClient:              s.backend.rpcClient,
		WalletService:          s.walletSrvc,
		CommunityTokensService: s.communityTokensSrvc,
		AccountsPublisher:      s.accountsSrvc.Publisher(),
		TimeSource:             s.timeSourceSrvc,
		MetricsEnabled:         s.backend.prometheusMetrics != nil,
		TokenManager:           adapters.NewCommunitiesTokenManager(s.backend.tokenManager),
		TokenBalanceManager:    adapters.NewCommunitiesTokenBalanceManager(tokenBalancesFetcher, tokenBalancesStorage),
		NetworkManager:         adapters.NewCommunitiesNetworkManager(s.backend.rpcClient.GetNetworkManager()),
	}

	err = s.wakuV2ExtSrvc.InitProtocol(params)
	if err != nil {
		return errors.Wrap(err, "failed to initialize protocol")
	}

	return nil
}

func (s *services) createEnsService() {
	s.requireWakuextService()

	timeSourceCb := s.timeSourceSrvc.Now // TODO: Replace callback with proper interface
	s.ensSrvc = ens.NewService(
		s.backend.rpcClient,
		s.backend.activeAccount.appDB,
		timeSourceCb,
	)
	s.ensSrvc.Init(s.wakuV2ExtSrvc.Messenger().SyncEnsNamesWithDispatchMessage)
	s.addService(s.ensSrvc)
}

func (s *services) createCommunityTokensService() {
	s.requireWakuextService()
	s.requireWalletService()

	s.communityTokensSrvc = communitytokens.NewService(
		s.backend.rpcClient,
		s.backend.activeAccount.accsManager,
		s.backend.activeAccount.appDB,
		&s.backend.walletFeed,
	)
	s.communityTokensSrvc.Init(s.wakuV2ExtSrvc.Messenger())
	s.addService(s.communityTokensSrvc)
}

func (s *services) createGifService(accountsDB *accounts.Database) {
	s.gifSrvc = gif.NewService(accountsDB)
	s.addService(s.gifSrvc)
}

func (s *services) createStickersService() {
	if s.mediaService == nil {
		s.logger.Warn("creating stickers service without media service")
	}
	s.stickersSrvc = stickers.NewService(
		s.backend.activeAccount.accountsDB,
		s.backend.rpcClient,
		s.backend.activeAccount.accsManager,
		s.backend.ipfs,
		s.mediaService,
	)
	s.addService(s.stickersSrvc)
}

func (s *services) createChatService(accountsDB *accounts.Database) {
	s.requireWakuextService()
	s.chatSrvc = chat.NewService(accountsDB)
	s.chatSrvc.Init(s.wakuV2ExtSrvc.Messenger())
	s.addService(s.chatSrvc)
}

func (s *services) createUpdatesService() {
	ensService := s.ensSrvc
	s.updatesSrvc = updates.NewService(ensService)
	s.addService(s.updatesSrvc)
}

func (s *services) createPendingTrackerService() {
	s.requireWalletService()
	s.pendingTracker = pendingtxtracker.NewPendingTxTracker(
		s.backend.activeAccount.walletDB,
		pendingtxtracker.NewBatchTxStatusFetcher(
			s.backend.rpcClient,
			s.logger.Named("PendingTxTracker"),
		),
		&s.backend.walletFeed,
		pendingtxtracker.PendingCheckInterval,
	)
	s.backend.transactor.SetPendingTracker(s.pendingTracker)
	s.addService(s.pendingTracker)
}

func (s *services) createConnectorService() {
	logger := s.logger.Named("connector")
	s.connectorSrvc = connector.NewService(
		logger,
		s.backend.activeAccount.walletDB,
		s.backend.rpcClient,
		fees.NewFeeManager(s.backend.rpcClient, logger.Named("feeManager")),
		s.backend.rpcClient.GetNetworkManager(),
		&connector.Config{
			WSHost: s.backend.activeAccount.nodeConfig.WSHost,
			WSPort: s.backend.activeAccount.nodeConfig.WSPort,
		},
	)
	s.addService(s.connectorSrvc)
}

func (s *services) createAppgeneralService() {
	s.appGeneralSrvc = appgeneral.New()
	s.addService(s.appGeneralSrvc)
}

func (s *services) createEthService() {
	s.ethSrvc = eth.NewService(s.backend.rpcClient, s.backend.activeAccount.accsManager)
	s.addService(s.ethSrvc)
}

func (s *services) createNewsFeedService() {
	if !featureflags.EnableNewsFeed {
		return
	}

	s.requireWakuextService()

	persistence := newsfeed.NewSQLitePersistence(s.backend.activeAccount.appDB)

	s.newsfeedSrvc = newsfeed.NewService(
		s.logger.Named("newsfeed"),
		persistence,
		nil,
	)

	activityCenter := adapters.NewNewsFeedActivityCenterAdapter(s.wakuV2ExtSrvc.Messenger())
	s.newsfeedSrvc.SetActivityCenter(activityCenter)

	s.addService(s.newsfeedSrvc)
}

func (s *services) createSharedUrlsService() {
	s.sharedUrlsSrvc = sharedurls.NewService(nil)

	provider := adapters.NewSharedUrlsMessengerAdapter(s.wakuV2ExtSrvc.Messenger())
	s.sharedUrlsSrvc.SetDataProvider(provider)

	s.addService(s.sharedUrlsSrvc)
}
