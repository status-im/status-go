package protocol

import (
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/params"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/backupsync"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/discord"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/wallet"
)

type MessageDeliveredHandler func(string, string)

type MessengerSignalsHandler interface {
	MessageDelivered(chatID string, messageID string)
	CommunityInfoFound(community *communities.Community)
	MessengerResponse(response *MessengerResponse)
	HistoryRequestStarted(numBatches int)
	HistoryRequestCompleted()

	HistoryArchivesProtocolEnabled()
	HistoryArchivesProtocolDisabled()
	CreatingHistoryArchives(communityID string)
	NoHistoryArchivesCreated(communityID string, from int, to int)
	HistoryArchivesCreated(communityID string, from int, to int)
	HistoryArchivesSeeding(communityID string)
	HistoryArchivesUnseeded(communityID string)
	HistoryArchiveDownloaded(communityID string, from int, to int)
	DownloadingHistoryArchivesStarted(communityID string)
	DownloadingHistoryArchivesFinished(communityID string)
	ImportingHistoryArchiveMessages(communityID string)
	StatusUpdatesTimedOut(statusUpdates *[]UserStatus)
	DiscordCategoriesAndChannelsExtracted(categories []*discord.Category, channels []*discord.Channel, oldestMessageTimestamp int64, errors map[string]*discord.ImportError)
	DiscordCommunityImportProgress(importProgress *discord.ImportProgress)
	DiscordCommunityImportFinished(communityID string)
	DiscordCommunityImportCancelled(communityID string)
	DiscordCommunityImportCleanedUp(communityID string)
	DiscordChannelImportProgress(importProgress *discord.ImportProgress)
	DiscordChannelImportFinished(communityID string, channelID string)
	DiscordChannelImportCancelled(channelID string)
	SendBackedUpProfile(response *backupsync.BackedUpDataResponse)
	SendBackedUpSettings(response *backupsync.BackedUpDataResponse)
	SendCuratedCommunitiesUpdate(response *communities.KnownCommunitiesResponse)
}

type config struct {
	// systemMessagesTranslations holds translations for system-messages
	systemMessagesTranslations *systemMessageTranslationsMap
	envelopeEventsConfig       *messagingtypes.EnvelopeEventsConfig

	featureFlags     common.FeatureFlags
	codeControlFlags common.CodeControlFlags

	appDb                  *sql.DB
	walletDb               *sql.DB
	afterDbCreatedHooks    []Option
	multiAccount           *multiaccounts.Database
	mailserversDatabase    *mailservers.Database
	account                *multiaccounts.Account
	clusterConfig          params.ClusterConfig
	browserDatabase        *browsers.Database
	torrentConfig          *params.TorrentConfig
	walletService          *wallet.Service
	communityTokensService communities.CommunityTokensServiceInterface
	httpServer             *server.MediaServer
	rpcClient              *rpc.Client
	tokenManager           communities.TokenManager
	tokenBalanceManager    communities.TokenBalanceManager
	networkManager         communities.NetworkManager
	collectiblesManager    communities.CollectiblesManager
	accountsManager        AccountsManager
	signer                 communities.MessageSigner

	ensVerifier *ens.Verifier

	pushNotificationClientConfig *pushnotificationclient.Config
	pushNotificationServer       PushNotificationServer

	logger *zap.Logger
	tracer trace.Tracer

	outputMessagesCSV bool

	messengerSignalsHandler MessengerSignalsHandler

	messageResendMinDelay time.Duration
	messageResendMaxCount int

	communityManagerOptions []communities.ManagerOption

	accountsPublisher *pubsub.Publisher

	onlineChecker func() bool

	communitiesRekeyInterval time.Duration
}

func messengerDefaultConfig() config {
	c := config{
		messageResendMinDelay: 30 * time.Second,
		messageResendMaxCount: 3,
	}

	c.codeControlFlags.AutoRequestHistoricMessages = true
	c.codeControlFlags.CuratedCommunitiesUpdateLoopEnabled = true

	c.tracer = trace.NewNoopTracer()

	return c
}

type Option func(*config) error

// WithSystemMessagesTranslations is required for Group Chats which are currently disabled.
// nolint: unused
func WithSystemMessagesTranslations(t map[protobuf.MembershipUpdateEvent_EventType]string) Option {
	return func(c *config) error {
		c.systemMessagesTranslations.Init(t)
		return nil
	}
}

func WithCustomLogger(logger *zap.Logger) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

func WithTracer(tracer trace.Tracer) Option {
	return func(c *config) error {
		c.tracer = tracer
		return nil
	}
}

func WithResendParams(minDelay time.Duration, maxCount int) Option {
	return func(c *config) error {
		c.messageResendMinDelay = minDelay
		c.messageResendMaxCount = maxCount
		return nil
	}
}

func WithDatabase(db *sql.DB) Option {
	return func(c *config) error {
		c.appDb = db
		return nil
	}
}

func WithWalletDatabase(db *sql.DB) Option {
	return func(c *config) error {
		c.walletDb = db
		return nil
	}
}

func WithMultiAccounts(ma *multiaccounts.Database) Option {
	return func(c *config) error {
		c.multiAccount = ma
		return nil
	}
}

func WithMailserversDatabase(ma *mailservers.Database) Option {
	return func(c *config) error {
		c.mailserversDatabase = ma
		return nil
	}
}

func WithAccount(acc *multiaccounts.Account) Option {
	return func(c *config) error {
		c.account = acc
		return nil
	}
}

func WithBrowserDatabase(bd *browsers.Database) Option {
	return func(c *config) error {
		c.browserDatabase = bd
		return nil
	}
}

func WithPushNotificationClientConfig(pushNotificationClientConfig *pushnotificationclient.Config) Option {
	return func(c *config) error {
		c.pushNotificationClientConfig = pushNotificationClientConfig
		return nil
	}
}

func WithDatasync() func(c *config) error {
	return func(c *config) error {
		c.featureFlags.Datasync = true
		return nil
	}
}

func WithPushNotifications() func(c *config) error {
	return func(c *config) error {
		c.featureFlags.PushNotifications = true
		return nil
	}
}

func WithPushNotificationServer(server PushNotificationServer) func(c *config) error {
	return func(c *config) error {
		c.pushNotificationServer = server
		return nil
	}
}

func WithAutoMessageDisabled() func(c *config) error {
	return func(c *config) error {
		c.featureFlags.DisableAutoMessageLoop = true
		return nil
	}
}

func WithEnvelopeEventsConfig(emc *messagingtypes.EnvelopeEventsConfig) Option {
	return func(c *config) error {
		c.envelopeEventsConfig = emc
		return nil
	}
}

func WithSignalsHandler(h MessengerSignalsHandler) Option {
	return func(c *config) error {
		c.messengerSignalsHandler = h
		return nil
	}
}

func WithClusterConfig(cc params.ClusterConfig) Option {
	return func(c *config) error {
		c.clusterConfig = cc
		return nil
	}
}

func WithTorrentConfig(tc *params.TorrentConfig) Option {
	return func(c *config) error {
		c.torrentConfig = tc
		return nil
	}
}

func WithHTTPServer(s *server.MediaServer) Option {
	return func(c *config) error {
		c.httpServer = s
		return nil
	}
}

func WithRPCClient(r *rpc.Client) Option {
	return func(c *config) error {
		c.rpcClient = r
		return nil
	}
}

func WithMessageCSV(enabled bool) Option {
	return func(c *config) error {
		c.outputMessagesCSV = enabled
		return nil
	}
}

func WithWalletService(s *wallet.Service) Option {
	return func(c *config) error {
		c.walletService = s
		return nil
	}
}

func WithCommunityTokensService(s communities.CommunityTokensServiceInterface) Option {
	return func(c *config) error {
		c.communityTokensService = s
		return nil
	}
}

func WithTokenManager(tokenManager communities.TokenManager) Option {
	return func(c *config) error {
		c.tokenManager = tokenManager
		return nil
	}
}

func WithTokenBalanceManager(tokenBalanceManager communities.TokenBalanceManager) Option {
	return func(c *config) error {
		c.tokenBalanceManager = tokenBalanceManager
		return nil
	}
}

func WithNetworkManager(networkManager communities.NetworkManager) Option {
	return func(c *config) error {
		c.networkManager = networkManager
		return nil
	}
}

func WithCollectiblesManager(collectiblesManager communities.CollectiblesManager) Option {
	return func(c *config) error {
		c.collectiblesManager = collectiblesManager
		return nil
	}
}

func WithAccountsManager(accountsManager AccountsManager) Option {
	return func(c *config) error {
		c.accountsManager = accountsManager
		return nil
	}
}

func WithMessageSigner(signer communities.MessageSigner) Option {
	return func(c *config) error {
		c.signer = signer
		return nil
	}
}

func WithAccountsPublisher(publisher *pubsub.Publisher) Option {
	return func(c *config) error {
		c.accountsPublisher = publisher
		return nil
	}
}

func WithENSVerifier(ensVerifier *ens.Verifier) func(c *config) error {
	return func(c *config) error {
		c.ensVerifier = ensVerifier
		return nil
	}
}

func WithCommunitiesRekeyInterval(interval time.Duration) func(c *config) error {
	return func(c *config) error {
		c.communitiesRekeyInterval = interval
		return nil
	}
}
