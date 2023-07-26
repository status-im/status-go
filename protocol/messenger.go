package protocol

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/appmetrics"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	sociallinkssettings "github.com/status-im/status-go/multiaccounts/settings_social_links"
	"github.com/status-im/status-go/protocol/anonmetrics"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/linkpreview"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/protocol/verification"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/browsers"
	ensservice "github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/services/ext/mailservers"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/telemetry"
)

// todo: kozieiev: get rid of wakutransp word
type chatContext string

const (
	PubKeyStringLength = 132

	transactionSentTxt = "Transaction sent"

	publicChat  chatContext = "public-chat"
	privateChat chatContext = "private-chat"
)

const messageResendMinDelay = 30
const messageResendMaxCount = 3

var communityAdvertiseIntervalSecond int64 = 60 * 60

// messageCacheIntervalMs is how long we should keep processed messages in the cache, in ms
var messageCacheIntervalMs uint64 = 1000 * 60 * 60 * 48

// Messenger is a entity managing chats and messages.
// It acts as a bridge between the application and encryption
// layers.
// It needs to expose an interface to manage installations
// because installations are managed by the user.
// Similarly, it needs to expose an interface to manage
// mailservers because they can also be managed by the user.
type Messenger struct {
	node                   types.Node
	server                 *p2p.Server
	peerStore              *mailservers.PeerStore
	config                 *config
	identity               *ecdsa.PrivateKey
	persistence            *sqlitePersistence
	transport              *transport.Transport
	encryptor              *encryption.Protocol
	sender                 *common.MessageSender
	ensVerifier            *ens.Verifier
	anonMetricsClient      *anonmetrics.Client
	anonMetricsServer      *anonmetrics.Server
	pushNotificationClient *pushnotificationclient.Client
	pushNotificationServer *pushnotificationserver.Server
	communitiesManager     *communities.Manager
	accountsManager        account.Manager
	mentionsManager        *MentionManager
	logger                 *zap.Logger

	outputCSV bool
	csvFile   *os.File

	verifyTransactionClient    EthClient
	featureFlags               common.FeatureFlags
	shutdownTasks              []func() error
	shouldPublishContactCode   bool
	systemMessagesTranslations *systemMessageTranslationsMap
	allChats                   *chatMap
	allContacts                *contactMap
	allInstallations           *installationMap
	modifiedInstallations      *stringBoolMap
	installationID             string
	mailserverCycle            mailserverCycle
	database                   *sql.DB
	multiAccounts              *multiaccounts.Database
	mailservers                *mailserversDB.Database
	settings                   *accounts.Database
	account                    *multiaccounts.Account
	mailserversDatabase        *mailserversDB.Database
	browserDatabase            *browsers.Database
	httpServer                 *server.MediaServer

	quit   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc

	importingCommunities map[string]bool
	importRateLimiter    *rate.Limiter
	importDelayer        struct {
		wait chan struct{}
		once sync.Once
	}

	requestedCommunitiesLock sync.RWMutex
	requestedCommunities     map[string]*transport.Filter

	requestedContactsLock sync.RWMutex
	requestedContacts     map[string]*transport.Filter

	connectionState                      connection.State
	telemetryClient                      *telemetry.Client
	contractMaker                        *contracts.ContractMaker
	downloadHistoryArchiveTasksWaitGroup sync.WaitGroup
	verificationDatabase                 *verification.Persistence
	savedAddressesManager                *wallet.SavedAddressesManager
	walletAPI                            *wallet.API

	// TODO(samyoul) Determine if/how the remaining usage of this mutex can be removed
	mutex                     sync.Mutex
	mailPeersMutex            sync.Mutex
	handleMessagesMutex       sync.Mutex
	handleImportMessagesMutex sync.Mutex

	// flag to disable checking #hasPairedDevices
	localPairing bool
	// flag to enable backedup messages processing, false by default
	processBackedupMessages bool
}

type connStatus int

const (
	disconnected connStatus = iota + 1
	connecting
	connected
)

type peerStatus struct {
	status                connStatus
	canConnectAfter       time.Time
	lastConnectionAttempt time.Time
	mailserver            mailserversDB.Mailserver
}
type mailserverCycle struct {
	sync.RWMutex
	activeMailserver          *mailserversDB.Mailserver
	peers                     map[string]peerStatus
	events                    chan *p2p.PeerEvent
	subscription              event.Subscription
	availabilitySubscriptions []chan struct{}
}

type dbConfig struct {
	dbPath          string
	dbKey           string
	dbKDFIterations int
}

type EnvelopeEventsInterceptor struct {
	EnvelopeEventsHandler transport.EnvelopeEventsHandler
	Messenger             *Messenger
}

// EnvelopeSent triggered when envelope delivered at least to 1 peer.
func (interceptor EnvelopeEventsInterceptor) EnvelopeSent(identifiers [][]byte) {
	if interceptor.Messenger != nil {
		var ids []string
		for _, identifierBytes := range identifiers {
			ids = append(ids, types.EncodeHex(identifierBytes))
		}

		err := interceptor.Messenger.processSentMessages(ids)
		if err != nil {
			interceptor.Messenger.logger.Info("messenger failed to process sent messages", zap.Error(err))
		}

		// We notify the client, regardless whether we were able to mark them as sent
		interceptor.EnvelopeEventsHandler.EnvelopeSent(identifiers)
	} else {
		// NOTE(rasom): In case if interceptor.Messenger is not nil and
		// some error occurred on processing sent message we don't want
		// to send envelop.sent signal to the client, thus `else` cause
		// is necessary.
		interceptor.EnvelopeEventsHandler.EnvelopeSent(identifiers)
	}
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (interceptor EnvelopeEventsInterceptor) EnvelopeExpired(identifiers [][]byte, err error) {
	//we don't track expired events in Messenger, so just redirect to handler
	interceptor.EnvelopeEventsHandler.EnvelopeExpired(identifiers, err)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (interceptor EnvelopeEventsInterceptor) MailServerRequestCompleted(requestID types.Hash, lastEnvelopeHash types.Hash, cursor []byte, err error) {
	//we don't track mailserver requests in Messenger, so just redirect to handler
	interceptor.EnvelopeEventsHandler.MailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (interceptor EnvelopeEventsInterceptor) MailServerRequestExpired(hash types.Hash) {
	//we don't track mailserver requests in Messenger, so just redirect to handler
	interceptor.EnvelopeEventsHandler.MailServerRequestExpired(hash)
}

func NewMessenger(
	nodeName string,
	identity *ecdsa.PrivateKey,
	node types.Node,
	installationID string,
	peerStore *mailservers.PeerStore,
	accountsManager account.Manager,
	opts ...Option,
) (*Messenger, error) {
	var messenger *Messenger

	c := config{}

	for _, opt := range opts {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	logger := c.logger
	if c.logger == nil {
		var err error
		if logger, err = zap.NewDevelopment(); err != nil {
			return nil, errors.Wrap(err, "failed to create a logger")
		}
	}

	if c.systemMessagesTranslations == nil {
		c.systemMessagesTranslations = defaultSystemMessagesTranslations
	}

	// Configure the database.
	database := c.db
	if c.db == nil && c.dbConfig == (dbConfig{}) {
		return nil, errors.New("database instance or database path needs to be provided")
	}
	if c.db == nil {
		logger.Info("opening a database", zap.String("dbPath", c.dbConfig.dbPath), zap.Int("KDFIterations", c.dbConfig.dbKDFIterations))
		var err error
		database, err = appdatabase.InitializeDB(c.dbConfig.dbPath, c.dbConfig.dbKey, c.dbConfig.dbKDFIterations)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize database from the db config")
		}
	}

	// Apply any post database creation changes to the database
	c.db = database
	for _, opt := range c.afterDbCreatedHooks {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	// Apply migrations for all components.
	err := sqlite.Migrate(database)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply migrations")
	}

	// Initialize transport layer.
	var transp *transport.Transport

	if waku, err := node.GetWaku(nil); err == nil && waku != nil {
		transp, err = transport.NewTransport(
			waku,
			identity,
			database,
			"waku_keys",
			nil,
			c.envelopesMonitorConfig,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create  Transport")
		}
	} else {
		logger.Info("failed to find Waku service; trying WakuV2", zap.Error(err))
		wakuV2, err := node.GetWakuV2(nil)
		if err != nil || wakuV2 == nil {
			return nil, errors.Wrap(err, "failed to find Whisper and Waku V1/V2 services")
		}
		transp, err = transport.NewTransport(
			wakuV2,
			identity,
			database,
			"wakuv2_keys",
			nil,
			c.envelopesMonitorConfig,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create  Transport")
		}
	}

	// Initialize encryption layer.
	encryptionProtocol := encryption.New(
		database,
		installationID,
		logger,
	)

	sender, err := common.NewMessageSender(
		identity,
		database,
		encryptionProtocol,
		transp,
		logger,
		c.featureFlags,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create messageSender")
	}

	// Initialise anon metrics client
	var anonMetricsClient *anonmetrics.Client
	if c.anonMetricsClientConfig != nil &&
		c.anonMetricsClientConfig.ShouldSend &&
		c.anonMetricsClientConfig.Active == anonmetrics.ActiveClientPhrase {

		anonMetricsClient = anonmetrics.NewClient(sender)
		anonMetricsClient.Config = c.anonMetricsClientConfig
		anonMetricsClient.Identity = identity
		anonMetricsClient.DB = appmetrics.NewDB(database)
		anonMetricsClient.Logger = logger
	}

	// Initialise anon metrics server
	var anonMetricsServer *anonmetrics.Server
	if c.anonMetricsServerConfig != nil &&
		c.anonMetricsServerConfig.Enabled &&
		c.anonMetricsServerConfig.Active == anonmetrics.ActiveServerPhrase {

		server, err := anonmetrics.NewServer(c.anonMetricsServerConfig.PostgresURI)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create anonmetrics.Server")
		}

		anonMetricsServer = server
		anonMetricsServer.Config = c.anonMetricsServerConfig
		anonMetricsServer.Logger = logger
	}

	var telemetryClient *telemetry.Client
	if c.telemetryServerURL != "" {
		telemetryClient = telemetry.NewClient(logger, c.telemetryServerURL, c.account.KeyUID, nodeName)
	}

	// Initialize push notification server
	var pushNotificationServer *pushnotificationserver.Server
	if c.pushNotificationServerConfig != nil && c.pushNotificationServerConfig.Enabled {
		c.pushNotificationServerConfig.Identity = identity
		pushNotificationServerPersistence := pushnotificationserver.NewSQLitePersistence(database)
		pushNotificationServer = pushnotificationserver.New(c.pushNotificationServerConfig, pushNotificationServerPersistence, sender)
	}

	// Initialize push notification client
	pushNotificationClientPersistence := pushnotificationclient.NewPersistence(database)
	pushNotificationClientConfig := c.pushNotificationClientConfig
	if pushNotificationClientConfig == nil {
		pushNotificationClientConfig = &pushnotificationclient.Config{}
	}

	sqlitePersistence := newSQLitePersistence(database)
	// Overriding until we handle different identities
	pushNotificationClientConfig.Identity = identity
	pushNotificationClientConfig.Logger = logger
	pushNotificationClientConfig.InstallationID = installationID

	pushNotificationClient := pushnotificationclient.New(pushNotificationClientPersistence, pushNotificationClientConfig, sender, sqlitePersistence)

	ensVerifier := ens.New(node, logger, transp, database, c.verifyENSURL, c.verifyENSContractAddress)

	var walletAPI *wallet.API
	if c.walletService != nil {
		walletAPI = wallet.NewAPI(c.walletService)
	}

	managerOptions := []communities.ManagerOption{
		communities.WithAccountManager(accountsManager),
	}

	if walletAPI != nil {
		managerOptions = append(managerOptions, communities.WithCollectiblesManager(walletAPI))
	}

	if c.tokenManager != nil {
		managerOptions = append(managerOptions, communities.WithTokenManager(c.tokenManager))
	} else if c.rpcClient != nil {
		tokenManager := token.NewTokenManager(database, c.rpcClient, c.rpcClient.NetworkManager)
		managerOptions = append(managerOptions, communities.WithTokenManager(communities.NewDefaultTokenManager(tokenManager)))
	}

	if c.walletConfig != nil {
		managerOptions = append(managerOptions, communities.WithWalletConfig(c.walletConfig))
	}

	communitiesManager, err := communities.NewManager(identity, database, encryptionProtocol, logger, ensVerifier, transp, c.torrentConfig, managerOptions...)
	if err != nil {
		return nil, err
	}

	settings, err := accounts.NewDB(database)
	if err != nil {
		return nil, err
	}

	mailservers := mailserversDB.NewDB(database)

	savedAddressesManager := wallet.NewSavedAddressesManager(c.db)

	myPublicKeyString := types.EncodeHex(crypto.FromECDSAPub(&identity.PublicKey))
	myContact, err := buildContact(myPublicKeyString, &identity.PublicKey)
	if err != nil {
		return nil, errors.New("failed to build contact of ourself: " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	messenger = &Messenger{
		config:                     &c,
		node:                       node,
		identity:                   identity,
		persistence:                sqlitePersistence,
		transport:                  transp,
		encryptor:                  encryptionProtocol,
		sender:                     sender,
		anonMetricsClient:          anonMetricsClient,
		anonMetricsServer:          anonMetricsServer,
		telemetryClient:            telemetryClient,
		pushNotificationClient:     pushNotificationClient,
		pushNotificationServer:     pushNotificationServer,
		communitiesManager:         communitiesManager,
		accountsManager:            accountsManager,
		ensVerifier:                ensVerifier,
		featureFlags:               c.featureFlags,
		systemMessagesTranslations: c.systemMessagesTranslations,
		allChats:                   new(chatMap),
		allContacts: &contactMap{
			logger: logger,
			me:     myContact,
		},
		allInstallations:        new(installationMap),
		installationID:          installationID,
		modifiedInstallations:   new(stringBoolMap),
		verifyTransactionClient: c.verifyTransactionClient,
		database:                database,
		multiAccounts:           c.multiAccount,
		settings:                settings,
		peerStore:               peerStore,
		verificationDatabase:    verification.NewPersistence(database),
		mailservers:             mailservers,
		mailserverCycle: mailserverCycle{
			peers:                     make(map[string]peerStatus),
			availabilitySubscriptions: make([]chan struct{}, 0),
		},
		mailserversDatabase:      c.mailserversDatabase,
		account:                  c.account,
		quit:                     make(chan struct{}),
		ctx:                      ctx,
		cancel:                   cancel,
		requestedCommunitiesLock: sync.RWMutex{},
		requestedCommunities:     make(map[string]*transport.Filter),
		requestedContactsLock:    sync.RWMutex{},
		requestedContacts:        make(map[string]*transport.Filter),
		importingCommunities:     make(map[string]bool),
		importRateLimiter:        rate.NewLimiter(rate.Every(importSlowRate), 1),
		importDelayer: struct {
			wait chan struct{}
			once sync.Once
		}{wait: make(chan struct{})},
		browserDatabase: c.browserDatabase,
		httpServer:      c.httpServer,
		contractMaker: &contracts.ContractMaker{
			RPCClient: c.rpcClient,
		},
		shutdownTasks: []func() error{
			ensVerifier.Stop,
			pushNotificationClient.Stop,
			communitiesManager.Stop,
			encryptionProtocol.Stop,
			func() error {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := transp.ResetFilters(ctx)
				if err != nil {
					logger.Warn("could not reset filters", zap.Error(err))
				}
				// We don't want to thrown an error in this case, this is a soft
				// fail
				return nil
			},
			transp.Stop,
			func() error { sender.Stop(); return nil },
			// Currently this often fails, seems like it's safe to ignore them
			// https://github.com/uber-go/zap/issues/328
			func() error { _ = logger.Sync; return nil },
			database.Close,
		},
		logger:                logger,
		savedAddressesManager: savedAddressesManager,
	}
	messenger.mentionsManager = NewMentionManager(messenger)

	if c.walletService != nil {
		messenger.walletAPI = walletAPI
	}

	if c.outputMessagesCSV {
		messenger.outputCSV = c.outputMessagesCSV
		csvFile, err := os.Create("messages-" + fmt.Sprint(time.Now().Unix()) + ".csv")
		if err != nil {
			return nil, err
		}

		_, err = csvFile.Write([]byte("timestamp\tmessageID\tfrom\ttopic\tchatID\tmessageType\tmessage\n"))
		if err != nil {
			return nil, err
		}

		messenger.csvFile = csvFile
		messenger.shutdownTasks = append(messenger.shutdownTasks, csvFile.Close)
	}

	if anonMetricsClient != nil {
		messenger.shutdownTasks = append(messenger.shutdownTasks, anonMetricsClient.Stop)
	}
	if anonMetricsServer != nil {
		messenger.shutdownTasks = append(messenger.shutdownTasks, anonMetricsServer.Stop)
	}

	if c.envelopesMonitorConfig != nil {
		interceptor := EnvelopeEventsInterceptor{c.envelopesMonitorConfig.EnvelopeEventsHandler, messenger}
		err := messenger.transport.SetEnvelopeEventsHandler(interceptor)
		if err != nil {
			logger.Info("Unable to set envelopes event handler", zap.Error(err))
		}
	}

	return messenger, nil
}

func (m *Messenger) SetP2PServer(server *p2p.Server) {
	m.server = server
}

func (m *Messenger) EnableBackedupMessagesProcessing() {
	m.processBackedupMessages = true
}

func (m *Messenger) processSentMessages(ids []string) error {
	if m.connectionState.Offline {
		return errors.New("Can't mark message as sent while offline")
	}

	for _, id := range ids {
		rawMessage, err := m.persistence.RawMessageByID(id)
		// If we have no raw message, we create a temporary one, so that
		// the sent status is preserved
		if err == sql.ErrNoRows || rawMessage == nil {
			rawMessage = &common.RawMessage{
				ID:          id,
				MessageType: protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
			}
		} else if err != nil {
			return errors.Wrapf(err, "Can't get raw message with id %v", id)
		}

		rawMessage.Sent = true

		err = m.persistence.SaveRawMessage(rawMessage)
		if err != nil {
			return errors.Wrapf(err, "Can't save raw message marked as sent")
		}

		err = m.UpdateMessageOutgoingStatus(id, common.OutgoingStatusSent)
		if err != nil {
			return err
		}
	}

	return nil
}

func shouldResendMessage(message *common.RawMessage, t common.TimeSource) (bool, error) {
	if !(message.MessageType == protobuf.ApplicationMetadataMessage_EMOJI_REACTION ||
		message.MessageType == protobuf.ApplicationMetadataMessage_CHAT_MESSAGE) {
		return false, errors.Errorf("Should resend only specific types of messages, can't resend %v", message.MessageType)
	}

	if message.Sent {
		return false, errors.New("Should resend only non-sent messages")
	}

	if message.SendCount > messageResendMaxCount {
		return false, nil
	}

	//exponential backoff depends on how many attempts to send message already made
	backoff := uint64(math.Pow(2, float64(message.SendCount-1))) * messageResendMinDelay * uint64(time.Second.Milliseconds())
	backoffElapsed := t.GetCurrentTime() > (message.LastSent + backoff)
	return backoffElapsed, nil
}

func (m *Messenger) resendExpiredMessages() error {
	if m.connectionState.Offline {
		return errors.New("offline")
	}

	ids, err := m.persistence.ExpiredMessagesIDs(messageResendMaxCount)
	if err != nil {
		return errors.Wrapf(err, "Can't get expired reactions from db")
	}

	for _, id := range ids {
		rawMessage, err := m.persistence.RawMessageByID(id)
		if err != nil {
			return errors.Wrapf(err, "Can't get raw message with id %v", id)
		}

		chat, ok := m.allChats.Load(rawMessage.LocalChatID)
		if !ok {
			return ErrChatNotFound
		}

		if !(chat.Public() || chat.CommunityChat()) {
			return errors.New("Only public chats and community chats messages are resent")
		}

		ok, err = shouldResendMessage(rawMessage, m.getTimesource())
		if err != nil {
			return err
		}

		if ok {
			err = m.persistence.SaveRawMessage(rawMessage)
			if err != nil {
				return errors.Wrapf(err, "Can't save raw message marked as non-expired")
			}

			err = m.reSendRawMessage(context.Background(), rawMessage.ID)
			if err != nil {
				return errors.Wrapf(err, "Can't resend expired message with id %v", rawMessage.ID)
			}
		}
	}
	return nil
}

func (m *Messenger) ToForeground() {
	if m.httpServer != nil {
		m.httpServer.ToForeground()
	}
}

func (m *Messenger) ToBackground() {
	if m.httpServer != nil {
		m.httpServer.ToBackground()
	}
}

func (m *Messenger) Start() (*MessengerResponse, error) {
	now := time.Now().UnixMilli()
	if err := m.settings.CheckAndDeleteExpiredKeypairsAndAccounts(uint64(now)); err != nil {
		return nil, err
	}

	m.logger.Info("starting messenger", zap.String("identity", types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))))
	// Start push notification server
	if m.pushNotificationServer != nil {
		if err := m.pushNotificationServer.Start(); err != nil {
			return nil, err
		}
	}

	// Start push notification client
	if m.pushNotificationClient != nil {
		m.handlePushNotificationClientRegistrations(m.pushNotificationClient.SubscribeToRegistrations())

		if err := m.pushNotificationClient.Start(); err != nil {
			return nil, err
		}
	}

	// Start anonymous metrics client
	if m.anonMetricsClient != nil {
		if err := m.anonMetricsClient.Start(); err != nil {
			return nil, err
		}
	}

	ensSubscription := m.ensVerifier.Subscribe()

	// Subscrbe
	if err := m.ensVerifier.Start(); err != nil {
		return nil, err
	}

	if err := m.communitiesManager.Start(); err != nil {
		return nil, err
	}

	// set shared secret handles
	m.sender.SetHandleSharedSecrets(m.handleSharedSecrets)

	subscriptions, err := m.encryptor.Start(m.identity)
	if err != nil {
		return nil, err
	}

	// handle stored shared secrets
	err = m.handleSharedSecrets(subscriptions.SharedSecrets)
	if err != nil {
		return nil, err
	}

	m.handleEncryptionLayerSubscriptions(subscriptions)
	m.handleCommunitiesSubscription(m.communitiesManager.Subscribe())
	m.handleCommunitiesHistoryArchivesSubscription(m.communitiesManager.Subscribe())
	m.updateCommunitiesActiveMembersPeriodically()
	m.handleConnectionChange(m.online())
	m.handleENSVerificationSubscription(ensSubscription)
	m.watchConnectionChange()
	m.watchChatsAndCommunitiesToUnmute()
	m.watchCommunitiesToUnmute()
	m.watchExpiredMessages()
	m.watchIdentityImageChanges()
	m.watchWalletBalances()
	m.watchPendingCommunityRequestToJoin()
	m.broadcastLatestUserStatus()
	m.timeoutAutomaticStatusUpdates()
	m.startBackupLoop()
	err = m.startAutoMessageLoop()
	if err != nil {
		return nil, err
	}
	m.startSyncSettingsLoop()
	m.startCommunityRekeyLoop()

	if err := m.cleanTopics(); err != nil {
		return nil, err
	}
	response := &MessengerResponse{}

	mailservers, err := m.allMailservers()
	if err != nil {
		return nil, err
	}

	response.Mailservers = mailservers
	err = m.StartMailserverCycle()
	if err != nil {
		return nil, err
	}

	if m.torrentClientReady() {
		adminCommunities, err := m.communitiesManager.Created()
		if err == nil && len(adminCommunities) > 0 {
			available := m.SubscribeMailserverAvailable()
			go func() {
				<-available
				m.InitHistoryArchiveTasks(adminCommunities)
			}()

			for _, c := range adminCommunities {
				if c.Joined() && c.HasTokenPermissions() {
					go m.communitiesManager.CheckMemberPermissionsPeriodically(c.ID())
				}
			}
		}
	}

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		return nil, err
	}

	for _, joinedCommunity := range joinedCommunities {
		// resume importing message history archives in case
		// imports have been interrupted previously
		err := m.resumeHistoryArchivesImport(joinedCommunity.ID())
		if err != nil {
			return nil, err
		}
	}
	m.enableHistoryArchivesImportAfterDelay()

	if m.httpServer != nil {
		err = m.httpServer.Start()
		if err != nil {
			return nil, err
		}
	}

	err = m.GarbageCollectRemovedBookmarks()
	if err != nil {
		return nil, err
	}

	err = m.garbageCollectRemovedSavedAddresses()
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) IdentityPublicKey() *ecdsa.PublicKey {
	return &m.identity.PublicKey
}

func (m *Messenger) IdentityPublicKeyCompressed() []byte {
	return crypto.CompressPubkey(m.IdentityPublicKey())
}

func (m *Messenger) IdentityPublicKeyString() string {
	return types.EncodeHex(crypto.FromECDSAPub(m.IdentityPublicKey()))
}

// cleanTopics remove any topic that does not have a Listen flag set
func (m *Messenger) cleanTopics() error {
	if m.mailserversDatabase == nil {
		return nil
	}
	var filters []*transport.Filter
	for _, f := range m.transport.Filters() {
		if f.Listen && !f.Ephemeral {
			filters = append(filters, f)
		}
	}

	m.logger.Debug("keeping topics", zap.Any("filters", filters))

	return m.mailserversDatabase.SetTopics(filters)
}

// handle connection change is called each time we go from offline/online or viceversa
func (m *Messenger) handleConnectionChange(online bool) {
	if online {
		if m.pushNotificationClient != nil {
			m.pushNotificationClient.Online()
		}

		if m.shouldPublishContactCode {
			if err := m.publishContactCode(); err != nil {
				m.logger.Error("could not publish on contact code", zap.Error(err))
				return
			}
			m.shouldPublishContactCode = false
		}
		go func() {
			_, err := m.RequestAllHistoricMessagesWithRetries(false)
			if err != nil {
				m.logger.Warn("failed to fetch historic messages", zap.Error(err))
			}
		}()

	} else {
		if m.pushNotificationClient != nil {
			m.pushNotificationClient.Offline()
		}

	}

	m.ensVerifier.SetOnline(online)
}

func (m *Messenger) online() bool {
	switch m.transport.WakuVersion() {
	case 2:
		return m.transport.PeerCount() > 0
	default:
		return m.node.PeersCount() > 0
	}
}

func (m *Messenger) buildContactCodeAdvertisement() (*protobuf.ContactCodeAdvertisement, error) {
	if m.pushNotificationClient == nil || !m.pushNotificationClient.Enabled() {
		return nil, nil
	}
	m.logger.Debug("adding push notification info to contact code bundle")
	info, err := m.pushNotificationClient.MyPushNotificationQueryInfo()
	if err != nil {
		return nil, err
	}
	if len(info) == 0 {
		return nil, nil
	}
	return &protobuf.ContactCodeAdvertisement{
		PushNotificationInfo: info,
	}, nil
}

// publishContactCode sends a public message wrapped in the encryption
// layer, which will propagate our bundle
func (m *Messenger) publishContactCode() error {
	var payload []byte
	m.logger.Debug("sending contact code")
	contactCodeAdvertisement, err := m.buildContactCodeAdvertisement()
	if err != nil {
		m.logger.Error("could not build contact code advertisement", zap.Error(err))
	}

	if contactCodeAdvertisement == nil {
		contactCodeAdvertisement = &protobuf.ContactCodeAdvertisement{}
	}

	err = m.attachChatIdentity(contactCodeAdvertisement)
	if err != nil {
		return err
	}

	if contactCodeAdvertisement.ChatIdentity != nil {
		m.logger.Debug("attached chat identity", zap.Int("images len", len(contactCodeAdvertisement.ChatIdentity.Images)))
	} else {
		m.logger.Debug("no attached chat identity")
	}

	payload, err = proto.Marshal(contactCodeAdvertisement)
	if err != nil {
		return err
	}

	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	rawMessage := common.RawMessage{
		LocalChatID: contactCodeTopic,
		MessageType: protobuf.ApplicationMetadataMessage_CONTACT_CODE_ADVERTISEMENT,
		Payload:     payload,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = m.sender.SendPublic(ctx, contactCodeTopic, rawMessage)
	if err != nil {
		m.logger.Warn("failed to send a contact code", zap.Error(err))
	}

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		return err
	}
	for _, community := range joinedCommunities {
		rawMessage.LocalChatID = community.MemberUpdateChannelID()
		_, err = m.sender.SendPublic(ctx, rawMessage.LocalChatID, rawMessage)
		if err != nil {
			return err
		}
	}

	m.logger.Debug("contact code sent")
	return err
}

// contactCodeAdvertisement attaches a protobuf.ChatIdentity to the given protobuf.ContactCodeAdvertisement,
// if the `shouldPublish` conditions are met
func (m *Messenger) attachChatIdentity(cca *protobuf.ContactCodeAdvertisement) error {
	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	shouldPublish, err := m.shouldPublishChatIdentity(contactCodeTopic)
	if err != nil {
		return err
	}

	if !shouldPublish {
		return nil
	}

	cca.ChatIdentity, err = m.createChatIdentity(privateChat)
	if err != nil {
		return err
	}

	img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, images.SmallDimName)
	if err != nil {
		return err
	}

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	bio, err := m.settings.Bio()
	if err != nil {
		return err
	}

	socialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return err
	}

	identityHash, err := m.getIdentityHash(displayName, bio, img, socialLinks)
	if err != nil {
		return err
	}

	err = m.persistence.SaveWhenChatIdentityLastPublished(contactCodeTopic, identityHash)
	if err != nil {
		return err
	}

	return nil
}

// handleStandaloneChatIdentity sends a standalone ChatIdentity message to a public or private channel if the publish criteria is met
func (m *Messenger) handleStandaloneChatIdentity(chat *Chat) error {
	if chat.ChatType != ChatTypePublic && chat.ChatType != ChatTypeOneToOne {
		return nil
	}
	shouldPublishChatIdentity, err := m.shouldPublishChatIdentity(chat.ID)
	if err != nil {
		return err
	}
	if !shouldPublishChatIdentity {
		return nil
	}

	ci, err := m.createChatIdentity(publicChat)
	if err != nil {
		return err
	}

	payload, err := proto.Marshal(ci)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID: chat.ID,
		MessageType: protobuf.ApplicationMetadataMessage_CHAT_IDENTITY,
		Payload:     payload,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if chat.ChatType == ChatTypePublic {
		_, err = m.sender.SendPublic(ctx, chat.ID, rawMessage)
		if err != nil {
			return err
		}
	} else {
		pk, err := chat.PublicKey()
		if err != nil {
			return err
		}
		_, err = m.sender.SendPrivate(ctx, pk, &rawMessage)
		if err != nil {
			return err
		}

	}

	img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, images.SmallDimName)
	if err != nil {
		return err
	}

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	bio, err := m.settings.Bio()
	if err != nil {
		return err
	}

	socialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return err
	}

	identityHash, err := m.getIdentityHash(displayName, bio, img, socialLinks)
	if err != nil {
		return err
	}

	err = m.persistence.SaveWhenChatIdentityLastPublished(chat.ID, identityHash)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) getIdentityHash(displayName, bio string, img *images.IdentityImage, socialLinks identity.SocialLinks) ([]byte, error) {
	socialLinksData, err := socialLinks.Serialize()
	if err != nil {
		return []byte{}, err
	}
	if img == nil {
		return crypto.Keccak256([]byte(displayName), []byte(bio), socialLinksData), nil
	}
	return crypto.Keccak256(img.Payload, []byte(displayName), []byte(bio), socialLinksData), nil
}

// shouldPublishChatIdentity returns true if the last time the ChatIdentity was attached was more than 24 hours ago
func (m *Messenger) shouldPublishChatIdentity(chatID string) (bool, error) {
	if m.account == nil {
		return false, nil
	}

	// Check we have at least one image or a display name
	img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, images.SmallDimName)
	if err != nil {
		return false, err
	}

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return false, err
	}

	if img == nil && displayName == "" {
		return false, nil
	}

	lp, hash, err := m.persistence.GetWhenChatIdentityLastPublished(chatID)
	if err != nil {
		return false, err
	}

	bio, err := m.settings.Bio()
	if err != nil {
		return false, err
	}

	socialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return false, err
	}

	identityHash, err := m.getIdentityHash(displayName, bio, img, socialLinks)
	if err != nil {
		return false, err
	}

	if !bytes.Equal(hash, identityHash) {
		return true, nil
	}

	// Note: If Alice does not add bob as a contact she will not update her contact code with images
	return lp == 0 || time.Now().Unix()-lp > 24*60*60, nil
}

// createChatIdentity creates a context based protobuf.ChatIdentity.
// context 'public-chat' will attach only the 'thumbnail' IdentityImage
// context 'private-chat' will attach all IdentityImage
func (m *Messenger) createChatIdentity(context chatContext) (*protobuf.ChatIdentity, error) {
	m.logger.Info(fmt.Sprintf("account keyUID '%s'", m.account.KeyUID))
	m.logger.Info(fmt.Sprintf("context '%s'", context))

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	bio, err := m.settings.Bio()
	if err != nil {
		return nil, err
	}

	socialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return nil, err
	}

	ci := &protobuf.ChatIdentity{
		Clock:       m.transport.GetCurrentTime(),
		EnsName:     "", // TODO add ENS name handling to dedicate PR
		DisplayName: displayName,
		Description: bio,
		SocialLinks: socialLinks.ToProtobuf(),
	}

	err = m.attachIdentityImagesToChatIdentity(context, ci)
	if err != nil {
		return nil, err
	}

	return ci, nil
}

// adaptIdentityImageToProtobuf Adapts a images.IdentityImage to protobuf.IdentityImage
func (m *Messenger) adaptIdentityImageToProtobuf(img *images.IdentityImage) *protobuf.IdentityImage {
	return &protobuf.IdentityImage{
		Payload:    img.Payload,
		SourceType: protobuf.IdentityImage_RAW_PAYLOAD, // TODO add ENS avatar handling to dedicated PR
		ImageType:  images.GetProtobufImageType(img.Payload),
	}
}

func (m *Messenger) attachIdentityImagesToChatIdentity(context chatContext, ci *protobuf.ChatIdentity) error {
	s, err := m.getSettings()
	if err != nil {
		return err
	}

	if s.ProfilePicturesShowTo == settings.ProfilePicturesShowToNone {
		m.logger.Info(fmt.Sprintf("settings.ProfilePicturesShowTo is set to '%d', skipping attaching IdentityImages", s.ProfilePicturesShowTo))
		return nil
	}

	ciis := make(map[string]*protobuf.IdentityImage)

	switch context {
	case publicChat:
		m.logger.Info(fmt.Sprintf("handling %s ChatIdentity", context))

		img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, images.SmallDimName)
		if err != nil {
			return err
		}

		if img == nil {
			return nil
		}

		m.logger.Debug(fmt.Sprintf("%s images.IdentityImage '%s'", context, spew.Sdump(img)))

		ciis[images.SmallDimName] = m.adaptIdentityImageToProtobuf(img)
		m.logger.Debug(fmt.Sprintf("%s protobuf.IdentityImage '%s'", context, spew.Sdump(ciis)))
		ci.Images = ciis

	case privateChat:
		m.logger.Info(fmt.Sprintf("handling %s ChatIdentity", context))

		imgs, err := m.multiAccounts.GetIdentityImages(m.account.KeyUID)
		if err != nil {
			return err
		}

		m.logger.Debug(fmt.Sprintf("%s images.IdentityImage '%s'", context, spew.Sdump(imgs)))

		for _, img := range imgs {
			ciis[img.Name] = m.adaptIdentityImageToProtobuf(img)
		}
		m.logger.Debug(fmt.Sprintf("%s protobuf.IdentityImage '%s'", context, spew.Sdump(ciis)))
		ci.Images = ciis

	default:
		return fmt.Errorf("unknown ChatIdentity context '%s'", context)
	}

	if s.ProfilePicturesShowTo == settings.ProfilePicturesShowToContactsOnly {
		err := EncryptIdentityImagesWithContactPubKeys(ci.Images, m)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleSharedSecrets process the negotiated secrets received from the encryption layer
func (m *Messenger) handleSharedSecrets(secrets []*sharedsecret.Secret) error {
	for _, secret := range secrets {
		fSecret := types.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		}
		_, err := m.transport.ProcessNegotiatedSecret(fSecret)
		if err != nil {
			return err
		}
	}
	return nil
}

// handleInstallations adds the installations in the installations map
func (m *Messenger) handleInstallations(installations []*multidevice.Installation) {
	for _, installation := range installations {
		if installation.Identity == contactIDFromPublicKey(&m.identity.PublicKey) {
			if _, ok := m.allInstallations.Load(installation.ID); !ok {
				m.allInstallations.Store(installation.ID, installation)
				m.modifiedInstallations.Store(installation.ID, true)
			}
		}
	}
}

// handleEncryptionLayerSubscriptions handles events from the encryption layer
func (m *Messenger) handleEncryptionLayerSubscriptions(subscriptions *encryption.Subscriptions) {
	go func() {
		for {
			select {
			case <-subscriptions.SendContactCode:
				if err := m.publishContactCode(); err != nil {
					m.logger.Error("failed to publish contact code", zap.Error(err))
				}
				// we also piggy-back to clean up cached messages
				if err := m.transport.CleanMessagesProcessed(m.getTimesource().GetCurrentTime() - messageCacheIntervalMs); err != nil {
					m.logger.Error("failed to clean processed messages", zap.Error(err))
				}

			case <-subscriptions.Quit:
				m.logger.Debug("quitting encryption subscription loop")
				return
			}
		}
	}()
}

func (m *Messenger) handleENSVerified(records []*ens.VerificationRecord) {
	var contacts []*Contact
	for _, record := range records {
		m.logger.Info("handling record", zap.Any("record", record))
		contact, ok := m.allContacts.Load(record.PublicKey)
		if !ok {
			m.logger.Info("contact not found")
			continue
		}

		contact.ENSVerified = record.Verified
		contact.EnsName = record.Name
		contacts = append(contacts, contact)
	}

	m.logger.Info("handled records", zap.Any("contacts", contacts))
	if len(contacts) != 0 {
		if err := m.persistence.SaveContacts(contacts); err != nil {
			m.logger.Error("failed to save contacts", zap.Error(err))
			return
		}
	}

	m.logger.Info("calling on contacts")
	if m.config.onContactENSVerified != nil {
		m.logger.Info("called on contacts")
		response := &MessengerResponse{Contacts: contacts}
		m.config.onContactENSVerified(response)
	}

}

func (m *Messenger) handleENSVerificationSubscription(c chan []*ens.VerificationRecord) {
	go func() {
		for {
			select {
			case records, more := <-c:
				if !more {
					m.logger.Info("No more records, quitting")
					return
				}
				if len(records) != 0 {
					m.logger.Info("handling records", zap.Any("records", records))
					m.handleENSVerified(records)
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchConnectionChange checks the connection status and call handleConnectionChange when this changes
func (m *Messenger) watchConnectionChange() {
	m.logger.Debug("watching connection changes")
	state := m.online()
	go func() {
		for {
			select {
			case <-time.After(200 * time.Millisecond):
				newState := m.online()
				if state != newState {
					state = newState
					m.logger.Debug("connection changed", zap.Bool("online", state))
					m.handleConnectionChange(state)
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchChatsAndCommunitiesToUnmute regularly checks for chats and communities that should be unmuted
func (m *Messenger) watchChatsAndCommunitiesToUnmute() {
	m.logger.Debug("watching unmuted chats")
	go func() {
		for {
			select {
			case <-time.After(3 * time.Second): // Poll every 3 seconds
				response := &MessengerResponse{}
				m.allChats.Range(func(chatID string, c *Chat) bool {
					chatMuteTill, _ := time.Parse(time.RFC3339, c.MuteTill.Format(time.RFC3339))
					currTime, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

					if currTime.After(chatMuteTill) && !chatMuteTill.Equal(time.Time{}) && c.Muted {
						err := m.persistence.UnmuteChat(c.ID)
						if err != nil {
							m.logger.Info("err", zap.Any("Couldn't unmute chat", err))
							return false
						}
						c.Muted = false
						c.MuteTill = time.Time{}
						response.AddChat(c)
					}
					return true
				})

				if !response.IsEmpty() {
					signal.SendNewMessages(response)
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchCommunitiesToUnmute regularly checks for communities that should be unmuted
func (m *Messenger) watchCommunitiesToUnmute() {
	m.logger.Debug("watching unmuted communities")
	go func() {
		for {
			select {
			case <-time.After(3 * time.Second): // Poll every 3 seconds
				response, err := m.CheckCommunitiesToUnmute()
				if err != nil {
					return
				}

				if !response.IsEmpty() {
					signal.SendNewMessages(response)
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchExpiredMessages regularly checks for expired emojis and invoke their resending
func (m *Messenger) watchExpiredMessages() {
	m.logger.Debug("watching expired messages")
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if m.online() {
					err := m.resendExpiredMessages()
					if err != nil {
						m.logger.Debug("Error when resending expired emoji reactions", zap.Error(err))
					}
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchIdentityImageChanges checks for identity images changes and publishes to the contact code when it happens
func (m *Messenger) watchIdentityImageChanges() {
	m.logger.Debug("watching identity image changes")
	if m.multiAccounts == nil {
		return
	}

	channel := m.multiAccounts.SubscribeToIdentityImageChanges()

	go func() {
		for {
			select {
			case <-channel:
				err := m.syncProfilePictures(m.dispatchMessage)
				if err != nil {
					m.logger.Error("failed to sync profile pictures to paired devices", zap.Error(err))
				}
				err = m.PublishIdentityImage()
				if err != nil {
					m.logger.Error("failed to publish identity image", zap.Error(err))
				}
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) watchPendingCommunityRequestToJoin() {
	m.logger.Debug("watching community request to join")

	go func() {
		for {
			select {
			case <-time.After(time.Minute * 10):
				_, err := m.CheckAndDeletePendingRequestToJoinCommunity(false)
				if err != nil {
					m.logger.Error("failed to check and delete pending request to join community", zap.Error(err))
				}
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) PublishIdentityImage() error {
	// Reset last published time for ChatIdentity so new contact can receive data
	err := m.resetLastPublishedTimeForChatIdentity()
	if err != nil {
		m.logger.Error("failed to reset publish time", zap.Error(err))
		return err
	}

	// If not online, we schedule it
	if !m.online() {
		m.shouldPublishContactCode = true
		return nil
	}

	return m.publishContactCode()
}

// handlePushNotificationClientRegistration handles registration events
func (m *Messenger) handlePushNotificationClientRegistrations(c chan struct{}) {
	go func() {
		for {
			_, more := <-c
			if !more {
				return
			}
			if err := m.publishContactCode(); err != nil {
				m.logger.Error("failed to publish contact code", zap.Error(err))
			}

		}
	}()
}

// Init analyzes chats and contacts in order to setup filters
// which are responsible for retrieving messages.
func (m *Messenger) Init() error {

	// Seed the for color generation
	rand.Seed(time.Now().Unix())

	logger := m.logger.With(zap.String("site", "Init"))

	var (
		publicChatIDs []string
		publicKeys    []*ecdsa.PublicKey
	)

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		return err
	}
	for _, org := range joinedCommunities {
		// the org advertise on the public topic derived by the pk
		publicChatIDs = append(publicChatIDs, org.DefaultFilters()...)

		// This is for status-go versions that didn't have `CommunitySettings`
		// We need to ensure communities that existed before community settings
		// were introduced will have community settings as well
		exists, err := m.communitiesManager.CommunitySettingsExist(org.ID())
		if err != nil {
			logger.Warn("failed to check if community settings exist", zap.Error(err))
			continue
		}

		if !exists {
			communitySettings := communities.CommunitySettings{
				CommunityID:                  org.IDString(),
				HistoryArchiveSupportEnabled: true,
			}

			err = m.communitiesManager.SaveCommunitySettings(communitySettings)
			if err != nil {
				logger.Warn("failed to save community settings", zap.Error(err))
			}
			continue
		}

		// In case we do have settings, but the history archive support is disabled
		// for this community, we enable it, as this should be the default for all
		// non-admin communities
		communitySettings, err := m.communitiesManager.GetCommunitySettingsByID(org.ID())
		if err != nil {
			logger.Warn("failed to fetch community settings", zap.Error(err))
			continue
		}

		if !org.IsControlNode() && !communitySettings.HistoryArchiveSupportEnabled {
			communitySettings.HistoryArchiveSupportEnabled = true
			err = m.communitiesManager.UpdateCommunitySettings(*communitySettings)
			if err != nil {
				logger.Warn("failed to update community settings", zap.Error(err))
			}
		}
	}

	spectatedCommunities, err := m.communitiesManager.Spectated()
	if err != nil {
		return err
	}

	for _, org := range spectatedCommunities {
		publicChatIDs = append(publicChatIDs, org.DefaultFilters()...)
	}

	// Init filters for the communities we are an admin of
	var adminCommunitiesPks []*ecdsa.PrivateKey
	adminCommunities, err := m.communitiesManager.Created()
	if err != nil {
		return err
	}

	for _, c := range adminCommunities {
		adminCommunitiesPks = append(adminCommunitiesPks, c.PrivateKey())
	}

	_, err = m.transport.InitCommunityFilters(adminCommunitiesPks)
	if err != nil {
		return err
	}

	// Get chat IDs and public keys from the existing chats.
	// TODO: Get only active chats by the query.
	chats, err := m.persistence.Chats()
	if err != nil {
		return err
	}
	for _, chat := range chats {
		if err := chat.Validate(); err != nil {
			logger.Warn("failed to validate chat", zap.Error(err))
			continue
		}

		if err = m.initChatFirstMessageTimestamp(chat); err != nil {
			logger.Warn("failed to init first message timestamp", zap.Error(err))
			continue
		}

		m.allChats.Store(chat.ID, chat)

		if !chat.Active || chat.Timeline() {
			continue
		}

		switch chat.ChatType {
		case ChatTypePublic, ChatTypeProfile:
			publicChatIDs = append(publicChatIDs, chat.ID)
		case ChatTypeCommunityChat:
			// TODO not public chat now
			publicChatIDs = append(publicChatIDs, chat.ID)
		case ChatTypeOneToOne:
			pk, err := chat.PublicKey()
			if err != nil {
				return err
			}
			publicKeys = append(publicKeys, pk)
		case ChatTypePrivateGroupChat:
			for _, member := range chat.Members {
				publicKey, err := member.PublicKey()
				if err != nil {
					return errors.Wrapf(err, "invalid public key for member %s in chat %s", member.ID, chat.Name)
				}
				publicKeys = append(publicKeys, publicKey)
			}
		default:
			return errors.New("invalid chat type")
		}
	}

	// Timeline and profile chats are deprecated.
	// This code can be removed after some reasonable time.

	// upsert timeline chat
	//err = m.ensureTimelineChat()
	//if err != nil {
	//	return err
	//}

	// upsert profile chat
	//err = m.ensureMyOwnProfileChat()
	//if err != nil {
	//	return err
	//}

	// Get chat IDs and public keys from the contacts.
	contacts, err := m.persistence.Contacts()
	if err != nil {
		return err
	}
	for idx, contact := range contacts {
		if err = m.updateContactImagesURL(contact); err != nil {
			return err
		}
		m.allContacts.Store(contact.ID, contacts[idx])
		// We only need filters for contacts added by us and not blocked.
		if !contact.added() || contact.Blocked {
			continue
		}
		publicKey, err := contact.PublicKey()
		if err != nil {
			logger.Error("failed to get contact's public key", zap.Error(err))
			continue
		}
		publicKeys = append(publicKeys, publicKey)
	}

	installations, err := m.encryptor.GetOurInstallations(&m.identity.PublicKey)
	if err != nil {
		return err
	}

	for _, installation := range installations {
		m.allInstallations.Store(installation.ID, installation)
	}

	err = m.setInstallationHostname()
	if err != nil {
		return err
	}

	_, err = m.transport.InitFilters(publicChatIDs, publicKeys)
	return err
}

// Shutdown takes care of ensuring a clean shutdown of Messenger
func (m *Messenger) Shutdown() (err error) {
	close(m.quit)
	m.cancel()
	m.downloadHistoryArchiveTasksWaitGroup.Wait()
	for i, task := range m.shutdownTasks {
		m.logger.Debug("running shutdown task", zap.Int("n", i))
		if tErr := task(); tErr != nil {
			m.logger.Info("shutdown task failed", zap.Error(tErr))
			if err == nil {
				// First error appeared.
				err = tErr
			} else {
				// We return all errors. They will be concatenated in the order of occurrence,
				// however, they will also be returned as a single error.
				err = errors.Wrap(err, tErr.Error())
			}
		}
	}
	return
}

func (m *Messenger) EnableInstallation(id string) error {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return errors.New("no installation found")
	}

	err := m.encryptor.EnableInstallation(&m.identity.PublicKey, id)
	if err != nil {
		return err
	}
	installation.Enabled = true
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allInstallations.Store(id, installation)
	return nil
}

func (m *Messenger) DisableInstallation(id string) error {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return errors.New("no installation found")
	}

	err := m.encryptor.DisableInstallation(&m.identity.PublicKey, id)
	if err != nil {
		return err
	}
	installation.Enabled = false
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allInstallations.Store(id, installation)
	return nil
}

func (m *Messenger) Installations() []*multidevice.Installation {
	installations := make([]*multidevice.Installation, m.allInstallations.Len())

	var i = 0
	m.allInstallations.Range(func(installationID string, installation *multidevice.Installation) (shouldContinue bool) {
		installations[i] = installation
		i++
		return true
	})
	return installations
}

func (m *Messenger) setInstallationMetadata(id string, data *multidevice.InstallationMetadata) error {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return errors.New("no installation found")
	}

	installation.InstallationMetadata = data
	return m.encryptor.SetInstallationMetadata(m.IdentityPublicKey(), id, data)
}

func (m *Messenger) SetInstallationMetadata(id string, data *multidevice.InstallationMetadata) error {
	return m.setInstallationMetadata(id, data)
}

func (m *Messenger) SetInstallationName(id string, name string) error {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return errors.New("no installation found")
	}

	installation.InstallationMetadata.Name = name
	return m.encryptor.SetInstallationName(m.IdentityPublicKey(), id, name)
}

// NOT IMPLEMENTED
func (m *Messenger) SelectMailserver(id string) error {
	return ErrNotImplemented
}

// NOT IMPLEMENTED
func (m *Messenger) AddMailserver(enode string) error {
	return ErrNotImplemented
}

// NOT IMPLEMENTED
func (m *Messenger) RemoveMailserver(id string) error {
	return ErrNotImplemented
}

// NOT IMPLEMENTED
func (m *Messenger) Mailservers() ([]string, error) {
	return nil, ErrNotImplemented
}

func (m *Messenger) initChatFirstMessageTimestamp(chat *Chat) error {
	if !chat.CommunityChat() || chat.FirstMessageTimestamp != FirstMessageTimestampUndefined {
		return nil
	}

	oldestMessageTimestamp, hasAnyMessage, err := m.persistence.OldestMessageWhisperTimestampByChatID(chat.ID)
	if err != nil {
		return err
	}

	if hasAnyMessage {
		if oldestMessageTimestamp == FirstMessageTimestampUndefined {
			return nil
		}
		return m.updateChatFirstMessageTimestamp(chat, whisperToUnixTimestamp(oldestMessageTimestamp), &MessengerResponse{})
	}

	return m.updateChatFirstMessageTimestamp(chat, FirstMessageTimestampNoMessage, &MessengerResponse{})
}

func (m *Messenger) addMessagesAndChat(chat *Chat, messages []*common.Message, response *MessengerResponse) (*MessengerResponse, error) {
	response.AddChat(chat)
	response.AddMessages(messages)
	err := m.persistence.SaveMessages(response.Messages())
	if err != nil {
		return nil, err
	}

	return response, m.saveChat(chat)
}

func (m *Messenger) reregisterForPushNotifications() error {
	m.logger.Info("contact state changed, re-registering for push notification")
	if m.pushNotificationClient == nil {
		return nil
	}

	return m.pushNotificationClient.Reregister(m.pushNotificationOptions())
}

// pull a message from the database and send it again
func (m *Messenger) reSendRawMessage(ctx context.Context, messageID string) error {
	message, err := m.persistence.RawMessageByID(messageID)
	if err != nil {
		return err
	}

	chat, ok := m.allChats.Load(message.LocalChatID)
	if !ok {
		return errors.New("chat not found")
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             message.Payload,
		MessageType:         message.MessageType,
		Recipients:          message.Recipients,
		ResendAutomatically: message.ResendAutomatically,
		SendCount:           message.SendCount,
	})
	return err
}

// ReSendChatMessage pulls a message from the database and sends it again
func (m *Messenger) ReSendChatMessage(ctx context.Context, messageID string) error {
	return m.reSendRawMessage(ctx, messageID)
}

func (m *Messenger) SetLocalPairing(localPairing bool) {
	m.localPairing = localPairing
}
func (m *Messenger) hasPairedDevices() bool {
	logger := m.logger.Named("hasPairedDevices")

	if m.localPairing {
		return true
	}

	var count int
	m.allInstallations.Range(func(installationID string, installation *multidevice.Installation) (shouldContinue bool) {
		if installation.Enabled {
			count++
		}
		return true
	})
	logger.Debug("installations info",
		zap.Int("Number of installations", m.allInstallations.Len()),
		zap.Int("Number of enabled installations", count))
	return count > 1
}

func (m *Messenger) HasPairedDevices() bool {
	return m.hasPairedDevices()
}

// sendToPairedDevices will check if we have any paired devices and send to them if necessary
func (m *Messenger) sendToPairedDevices(ctx context.Context, spec common.RawMessage) error {
	hasPairedDevices := m.hasPairedDevices()
	// We send a message to any paired device
	if hasPairedDevices {
		_, err := m.sender.SendPrivate(ctx, &m.identity.PublicKey, &spec)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) dispatchPairInstallationMessage(ctx context.Context, spec common.RawMessage) (common.RawMessage, error) {
	var err error
	var id []byte

	id, err = m.sender.SendPairInstallation(ctx, &m.identity.PublicKey, spec)

	if err != nil {
		return spec, err
	}
	spec.ID = types.EncodeHex(id)
	spec.SendCount++
	err = m.persistence.SaveRawMessage(&spec)
	if err != nil {
		return spec, err
	}

	return spec, nil
}

func (m *Messenger) dispatchMessage(ctx context.Context, rawMessage common.RawMessage) (common.RawMessage, error) {
	var err error
	var id []byte
	logger := m.logger.With(zap.String("site", "dispatchMessage"), zap.String("chatID", rawMessage.LocalChatID))
	chat, ok := m.allChats.Load(rawMessage.LocalChatID)
	if !ok {
		return rawMessage, errors.New("no chat found")
	}

	switch chat.ChatType {
	case ChatTypeOneToOne:
		publicKey, err := chat.PublicKey()
		if err != nil {
			return rawMessage, err
		}

		//SendPrivate will alter message identity and possibly datasyncid, so we save an unchanged
		//message for sending to paired devices later
		specCopyForPairedDevices := rawMessage
		if !common.IsPubKeyEqual(publicKey, &m.identity.PublicKey) || rawMessage.SkipProtocolLayer {
			id, err = m.sender.SendPrivate(ctx, publicKey, &rawMessage)

			if err != nil {
				return rawMessage, err
			}
		}

		err = m.sendToPairedDevices(ctx, specCopyForPairedDevices)

		if err != nil {
			return rawMessage, err
		}

	case ChatTypePublic, ChatTypeProfile:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		id, err = m.sender.SendPublic(ctx, chat.ID, rawMessage)
		if err != nil {
			return rawMessage, err
		}
	case ChatTypeCommunityChat:
		// TODO: add grant
		canPost, err := m.communitiesManager.CanPost(&m.identity.PublicKey, chat.CommunityID, chat.CommunityChatID(), nil)
		if err != nil {
			return rawMessage, err
		}

		// We allow emoji reactions by anyone
		if rawMessage.MessageType != protobuf.ApplicationMetadataMessage_EMOJI_REACTION && !canPost {
			m.logger.Error("can't post on chat", zap.String("chat-id", chat.ID), zap.String("chat-name", chat.Name))

			return rawMessage, errors.New("can't post on chat")
		}

		logger.Debug("sending community chat message", zap.String("chatName", chat.Name))
		isEncrypted, err := m.communitiesManager.IsEncrypted(chat.CommunityID)
		if err != nil {
			return rawMessage, err
		}
		if !isEncrypted {
			id, err = m.sender.SendPublic(ctx, chat.ID, rawMessage)
		} else {
			rawMessage.CommunityID, err = types.DecodeHex(chat.CommunityID)

			if err == nil {
				id, err = m.sender.SendCommunityMessage(ctx, rawMessage)
			}
		}
		if err != nil {
			return rawMessage, err
		}
	case ChatTypePrivateGroupChat:
		logger.Debug("sending group message", zap.String("chatName", chat.Name))
		if rawMessage.Recipients == nil {
			rawMessage.Recipients, err = chat.MembersAsPublicKeys()
			if err != nil {
				return rawMessage, err
			}
		}

		hasPairedDevices := m.hasPairedDevices()

		if !hasPairedDevices {

			// Filter out my key from the recipients
			n := 0
			for _, recipient := range rawMessage.Recipients {
				if !common.IsPubKeyEqual(recipient, &m.identity.PublicKey) {
					rawMessage.Recipients[n] = recipient
					n++
				}
			}
			rawMessage.Recipients = rawMessage.Recipients[:n]
		}

		// We won't really send the message out if there's no recipients
		if len(rawMessage.Recipients) == 0 {
			rawMessage.Sent = true
		}

		// We skip wrapping in some cases (emoji reactions for example)
		if !rawMessage.SkipGroupMessageWrap {
			rawMessage.MessageType = protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE
		}

		id, err = m.sender.SendGroup(ctx, rawMessage.Recipients, rawMessage)
		if err != nil {
			return rawMessage, err
		}

	default:
		return rawMessage, errors.New("chat type not supported")
	}
	rawMessage.ID = types.EncodeHex(id)
	rawMessage.SendCount++
	rawMessage.LastSent = m.getTimesource().GetCurrentTime()
	err = m.persistence.SaveRawMessage(&rawMessage)
	if err != nil {
		return rawMessage, err
	}

	return rawMessage, nil
}

// SendChatMessage takes a minimal message and sends it based on the corresponding chat
func (m *Messenger) SendChatMessage(ctx context.Context, message *common.Message) (*MessengerResponse, error) {
	return m.sendChatMessage(ctx, message)
}

// SendChatMessages takes a array of messages and sends it based on the corresponding chats
func (m *Messenger) SendChatMessages(ctx context.Context, messages []*common.Message) (*MessengerResponse, error) {
	var response MessengerResponse

	generatedAlbumID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	imagesCount := uint32(0)
	for _, message := range messages {
		if message.ContentType == protobuf.ChatMessage_IMAGE {
			imagesCount++
		}

	}

	for _, message := range messages {
		if message.ContentType == protobuf.ChatMessage_IMAGE && len(messages) > 1 {
			err = message.SetAlbumIDAndImagesCount(generatedAlbumID.String(), imagesCount)
			if err != nil {
				return nil, err
			}
		}
		messageResponse, err := m.SendChatMessage(ctx, message)
		if err != nil {
			return nil, err
		}
		err = response.Merge(messageResponse)
		if err != nil {
			return nil, err
		}
	}

	return &response, nil
}

// sendChatMessage takes a minimal message and sends it based on the corresponding chat
func (m *Messenger) sendChatMessage(ctx context.Context, message *common.Message) (*MessengerResponse, error) {
	displayName, err := m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	message.DisplayName = displayName
	if len(message.ImagePath) != 0 {

		err := message.LoadImage()
		if err != nil {
			return nil, err
		}

	} else if len(message.CommunityID) != 0 {
		community, err := m.communitiesManager.GetByIDString(message.CommunityID)
		if err != nil {
			return nil, err
		}

		if community == nil {
			return nil, errors.New("community not found")
		}

		wrappedCommunity, err := community.ToBytes()
		if err != nil {
			return nil, err
		}
		message.Payload = &protobuf.ChatMessage_Community{Community: wrappedCommunity}

		message.ContentType = protobuf.ChatMessage_COMMUNITY
	} else if len(message.AudioPath) != 0 {
		err := message.LoadAudio()
		if err != nil {
			return nil, err
		}
	}

	unfurledLinks, err := message.ConvertLinkPreviewsToProto()
	// We consider link previews non-critical data, so we do not want to block
	// messages from being sent.
	if err != nil {
		m.logger.Error("failed to convert link previews", zap.Error(err))
	} else {
		message.UnfurledLinks = unfurledLinks
	}

	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats.Load(message.ChatId)
	if !ok {
		return nil, errors.New("Chat not found")
	}

	err = m.handleStandaloneChatIdentity(chat)
	if err != nil {
		return nil, err
	}

	err = extendMessageFromChat(message, chat, &m.identity.PublicKey, m.getTimesource())
	if err != nil {
		return nil, err
	}

	err = m.addContactRequestPropagatedState(message)
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.encodeChatEntity(chat, message)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		LocalChatID:          chat.ID,
		SendPushNotification: m.featureFlags.PushNotifications,
		Payload:              encodedMessage,
		MessageType:          protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		ResendAutomatically:  true,
	}

	// We want to save the raw message before dispatching it, to avoid race conditions
	// since it might get dispatched and confirmed before it's saved.
	// This is not the best solution, probably it would be better to split
	// the sent status in a different table and join on query for messages,
	// but that's a much larger change and it would require an expensive migration of clients
	rawMessage.BeforeDispatch = func(rawMessage *common.RawMessage) error {
		if rawMessage.Sent {
			message.OutgoingStatus = common.OutgoingStatusSent
		}
		message.ID = rawMessage.ID
		err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
		if err != nil {
			return err
		}

		err = chat.UpdateFromMessage(message, m.getTimesource())
		if err != nil {
			return err
		}

		return m.persistence.SaveMessages([]*common.Message{message})
	}

	rawMessage, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return nil, err
	}

	msg, err := m.pullMessagesAndResponsesFromDB([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	if err := m.updateChatFirstMessageTimestamp(chat, whisperToUnixTimestamp(message.WhisperTimestamp), &response); err != nil {
		return nil, err
	}

	response.SetMessages(msg)
	response.AddChat(chat)

	m.logger.Debug("sent message", zap.String("id", message.ID))
	m.prepareMessages(response.messages)

	return &response, m.saveChat(chat)
}

func whisperToUnixTimestamp(whisperTimestamp uint64) uint32 {
	return uint32(whisperTimestamp / 1000)
}

func (m *Messenger) updateChatFirstMessageTimestamp(chat *Chat, timestamp uint32, response *MessengerResponse) error {
	// Currently supported only for communities
	if !chat.CommunityChat() {
		return nil
	}

	community, err := m.communitiesManager.GetByIDString(chat.CommunityID)
	if err != nil {
		return err
	}

	if community.IsControlNode() && chat.UpdateFirstMessageTimestamp(timestamp) {
		community, changes, err := m.communitiesManager.EditChatFirstMessageTimestamp(community.ID(), chat.ID, chat.FirstMessageTimestamp)
		if err != nil {
			return err
		}

		response.AddCommunity(community)
		response.CommunityChanges = append(response.CommunityChanges, changes)
	}

	return nil
}

func (m *Messenger) ShareImageMessage(request *requests.ShareImageMessage) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	response := &MessengerResponse{}

	msg, err := m.persistence.MessageByID(request.MessageID)
	if err != nil {
		return nil, err
	}

	var messages []*common.Message
	for _, pk := range request.Users {
		message := &common.Message{}
		message.ChatId = pk.String()
		message.Payload = msg.Payload
		message.Text = "This message has been shared with you"
		message.ContentType = protobuf.ChatMessage_IMAGE
		messages = append(messages, message)

		r, err := m.CreateOneToOneChat(&requests.CreateOneToOneChat{ID: pk})
		if err != nil {
			return nil, err
		}

		if err := response.Merge(r); err != nil {
			return nil, err
		}
	}

	sendMessagesResponse, err := m.SendChatMessages(context.Background(), messages)
	if err != nil {
		return nil, err
	}

	if err := response.Merge(sendMessagesResponse); err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) syncProfilePictures(rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	keyUID := m.account.KeyUID
	images, err := m.multiAccounts.GetIdentityImages(keyUID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pictures := make([]*protobuf.SyncProfilePicture, len(images))
	clock, chat := m.getLastClockWithRelatedChat()
	for i, image := range images {
		p := &protobuf.SyncProfilePicture{}
		p.Name = image.Name
		p.Payload = image.Payload
		p.Width = uint32(image.Width)
		p.Height = uint32(image.Height)
		p.FileSize = uint32(image.FileSize)
		p.ResizeTarget = uint32(image.ResizeTarget)
		if image.Clock == 0 {
			p.Clock = clock
		} else {
			p.Clock = image.Clock
		}
		pictures[i] = p
	}

	message := &protobuf.SyncProfilePictures{}
	message.KeyUid = keyUID
	message.Pictures = pictures

	encodedMessage, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_PROFILE_PICTURE,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

// SyncDevices sends all public chats and contacts to paired devices
// TODO remove use of photoPath in contacts
func (m *Messenger) SyncDevices(ctx context.Context, ensName, photoPath string, rawMessageHandler RawMessageHandler) (err error) {
	syncedFromLocalPairing := rawMessageHandler != nil
	if rawMessageHandler == nil {
		rawMessageHandler = m.dispatchMessage
	}

	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	if _, err = m.sendContactUpdate(ctx, myID, displayName, ensName, photoPath, rawMessageHandler); err != nil {
		return err
	}

	m.allChats.Range(func(chatID string, chat *Chat) (shouldContinue bool) {
		isPublicChat := !chat.Timeline() && !chat.ProfileUpdates() && chat.Public()
		if isPublicChat && chat.Active {
			err = m.syncPublicChat(ctx, chat, rawMessageHandler)
			if err != nil {
				return false
			}
		}

		if (isPublicChat || chat.OneToOne() || chat.PrivateGroupChat()) && !chat.Active && chat.DeletedAtClockValue > 0 {
			pending, err := m.persistence.HasPendingNotificationsForChat(chat.ID)
			if err != nil {
				return false
			}

			if !pending {
				err = m.syncChatRemoving(ctx, chatID, rawMessageHandler)
				if err != nil {
					return false
				}
			}
		}

		if (isPublicChat || chat.OneToOne() || chat.PrivateGroupChat() || chat.CommunityChat()) && chat.Active {
			err := m.syncChatMessagesRead(ctx, chatID, chat.ReadMessagesAtClockValue, rawMessageHandler)
			if err != nil {
				return false
			}
		}

		if isPublicChat && chat.Active && chat.DeletedAtClockValue > 0 {
			err = m.syncClearHistory(ctx, chat, rawMessageHandler)
			if err != nil {
				return false
			}
		}

		return true
	})
	if err != nil {
		return err
	}

	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.ID != myID &&
			(contact.LocalNickname != "" || contact.added() || contact.Blocked) {
			if err = m.syncContact(ctx, contact, rawMessageHandler); err != nil {
				return false
			}
		}
		return true
	})

	cs, err := m.communitiesManager.JoinedAndPendingCommunitiesWithRequests()
	if err != nil {
		return err
	}
	for _, c := range cs {
		if err = m.syncCommunity(ctx, c, rawMessageHandler); err != nil {
			return err
		}
	}

	bookmarks, err := m.browserDatabase.GetBookmarks()
	if err != nil {
		return err
	}
	for _, b := range bookmarks {
		if err = m.SyncBookmark(ctx, b, rawMessageHandler); err != nil {
			return err
		}
	}

	trustedUsers, err := m.verificationDatabase.GetAllTrustStatus()
	if err != nil {
		return err
	}
	for id, ts := range trustedUsers {
		if err = m.SyncTrustedUser(ctx, id, ts, rawMessageHandler); err != nil {
			return err
		}
	}

	verificationRequests, err := m.verificationDatabase.GetVerificationRequests()
	if err != nil {
		return err
	}
	for i := range verificationRequests {
		if err = m.SyncVerificationRequest(ctx, &verificationRequests[i], rawMessageHandler); err != nil {
			return err
		}
	}

	err = m.syncSettings(rawMessageHandler)
	if err != nil {
		return err
	}

	err = m.syncProfilePictures(rawMessageHandler)
	if err != nil {
		return err
	}

	ids, err := m.persistence.LatestContactRequestIDs()

	if err != nil {
		return err
	}

	for id, state := range ids {
		if state == common.ContactRequestStateAccepted || state == common.ContactRequestStateDismissed {
			accepted := state == common.ContactRequestStateAccepted
			err := m.syncContactRequestDecision(ctx, id, accepted, rawMessageHandler)
			if err != nil {
				return err
			}
		}
	}

	// we have to sync deleted keypairs as well
	keypairs, err := m.settings.GetAllKeypairs()
	if err != nil {
		return err
	}

	for _, kp := range keypairs {
		if syncedFromLocalPairing {
			kp.SyncedFrom = accounts.SyncedFromLocalPairing
		}
		err = m.syncKeypair(kp, rawMessageHandler)
		if err != nil {
			return err
		}
	}

	// we have to sync deleted watch only accounts as well
	woAccounts, err := m.settings.GetAllWatchOnlyAccounts()
	if err != nil {
		return err
	}

	for _, woAcc := range woAccounts {
		err = m.syncWalletAccount(woAcc, rawMessageHandler)
		if err != nil {
			return err
		}
	}

	savedAddresses, err := m.savedAddressesManager.GetRawSavedAddresses()
	if err != nil {
		return err
	}

	for i := range savedAddresses {
		sa := savedAddresses[i]

		err = m.syncSavedAddress(ctx, sa, rawMessageHandler)
		if err != nil {
			return err
		}
	}

	if err = m.syncEnsUsernameDetails(ctx, rawMessageHandler); err != nil {
		return err
	}

	if err = m.syncDeleteForMeMessage(ctx, rawMessageHandler); err != nil {
		return err
	}

	err = m.syncAccountsPositions(rawMessageHandler)
	if err != nil {
		return err
	}

	return m.syncSocialLinks(context.Background(), rawMessageHandler)
}

func (m *Messenger) syncContactRequestDecision(ctx context.Context, requestID string, accepted bool, rawMessageHandler RawMessageHandler) error {
	m.logger.Info("syncContactRequestDecision", zap.Any("from", requestID))
	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	var status protobuf.SyncContactRequestDecision_DecisionStatus
	if accepted {
		status = protobuf.SyncContactRequestDecision_ACCEPTED
	} else {
		status = protobuf.SyncContactRequestDecision_DECLINED
	}

	message := &protobuf.SyncContactRequestDecision{
		RequestId:      requestID,
		Clock:          clock,
		DecisionStatus: status,
	}

	encodedMessage, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_CONTACT_REQUEST_DECISION,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) getLastClockWithRelatedChat() (uint64, *Chat) {
	chatID := contactIDFromPublicKey(&m.identity.PublicKey)

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		chat = OneToOneFromPublicKey(&m.identity.PublicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats.Store(chat.ID, chat)
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	return clock, chat
}

// SendPairInstallation sends a pair installation message
func (m *Messenger) SendPairInstallation(ctx context.Context, rawMessageHandler RawMessageHandler) (*MessengerResponse, error) {
	var err error
	var response MessengerResponse

	installation, ok := m.allInstallations.Load(m.installationID)
	if !ok {
		return nil, errors.New("no installation found")
	}

	if installation.InstallationMetadata == nil {
		return nil, errors.New("no installation metadata")
	}

	clock, chat := m.getLastClockWithRelatedChat()

	pairMessage := &protobuf.PairInstallation{
		Clock:          clock,
		Name:           installation.InstallationMetadata.Name,
		InstallationId: installation.ID,
		DeviceType:     installation.InstallationMetadata.DeviceType,
		Version:        installation.Version}
	encodedMessage, err := proto.Marshal(pairMessage)
	if err != nil {
		return nil, err
	}

	if rawMessageHandler == nil {
		rawMessageHandler = m.dispatchPairInstallationMessage
	}
	_, err = rawMessageHandler(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_PAIR_INSTALLATION,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)

	chat.LastClockValue = clock
	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// syncPublicChat sync a public chat with paired devices
func (m *Messenger) syncPublicChat(ctx context.Context, publicChat *Chat, rawMessageHandler RawMessageHandler) error {
	var err error
	if !m.hasPairedDevices() {
		return nil
	}
	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncInstallationPublicChat{
		Clock: clock,
		Id:    publicChat.ID,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_PUBLIC_CHAT,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncClearHistory(ctx context.Context, publicChat *Chat, rawMessageHandler RawMessageHandler) error {
	var err error
	if !m.hasPairedDevices() {
		return nil
	}
	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncClearHistory{
		ChatId:    publicChat.ID,
		ClearedAt: publicChat.DeletedAtClockValue,
	}

	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_CLEAR_HISTORY,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncChatRemoving(ctx context.Context, id string, rawMessageHandler RawMessageHandler) error {
	var err error
	if !m.hasPairedDevices() {
		return nil
	}
	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncChatRemoved{
		Clock: clock,
		Id:    id,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_CHAT_REMOVED,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

// syncContact sync as contact with paired devices
func (m *Messenger) syncContact(ctx context.Context, contact *Contact, rawMessageHandler RawMessageHandler) error {
	var err error
	if contact.IsSyncing {
		return nil
	}
	if !m.hasPairedDevices() {
		return nil
	}
	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := m.buildSyncContactMessage(contact)

	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_CONTACT,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncCommunity(ctx context.Context, community *communities.Community, rawMessageHandler RawMessageHandler) error {
	logger := m.logger.Named("syncCommunity")
	if !m.hasPairedDevices() {
		logger.Debug("device has no paired devices")
		return nil
	}
	logger.Debug("device has paired device(s)")

	clock, chat := m.getLastClockWithRelatedChat()

	communitySettings, err := m.communitiesManager.GetCommunitySettingsByID(community.ID())
	if err != nil {
		return err
	}

	syncMessage, err := community.ToSyncCommunityProtobuf(clock, communitySettings)
	if err != nil {
		return err
	}

	encodedKeys, err := m.encryptor.GetAllHREncodedKeys(community.ID())
	if err != nil {
		return err
	}
	syncMessage.EncryptionKeys = encodedKeys

	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_COMMUNITY,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}
	logger.Debug("message dispatched")

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) SyncBookmark(ctx context.Context, bookmark *browsers.Bookmark, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncBookmark{
		Clock:     clock,
		Url:       bookmark.URL,
		Name:      bookmark.Name,
		ImageUrl:  bookmark.ImageURL,
		Removed:   bookmark.Removed,
		DeletedAt: bookmark.DeletedAt,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_BOOKMARK,
		ResendAutomatically: true,
	}
	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) SyncEnsNamesWithDispatchMessage(ctx context.Context, usernameDetail *ensservice.UsernameDetail) error {
	return m.syncEnsUsernameDetail(ctx, usernameDetail, m.dispatchMessage)
}

func (m *Messenger) syncEnsUsernameDetails(ctx context.Context, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	ensNameDetails, err := m.getEnsUsernameDetails()
	if err != nil {
		return err
	}
	for _, d := range ensNameDetails {
		if err = m.syncEnsUsernameDetail(ctx, d, rawMessageHandler); err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) saveEnsUsernameDetailProto(syncMessage protobuf.SyncEnsUsernameDetail) (*ensservice.UsernameDetail, error) {
	ud := &ensservice.UsernameDetail{
		Username: syncMessage.Username,
		Clock:    syncMessage.Clock,
		ChainID:  syncMessage.ChainId,
		Removed:  syncMessage.Removed,
	}
	db := ensservice.NewEnsDatabase(m.database)
	err := db.SaveOrUpdateEnsUsername(ud)
	if err != nil {
		return nil, err
	}
	return ud, nil
}

func (m *Messenger) handleSyncEnsUsernameDetail(state *ReceivedMessageState, syncMessage protobuf.SyncEnsUsernameDetail) error {
	ud, err := m.saveEnsUsernameDetailProto(syncMessage)
	if err != nil {
		return err
	}
	state.Response.AddEnsUsernameDetail(ud)
	return nil
}

func (m *Messenger) syncEnsUsernameDetail(ctx context.Context, usernameDetail *ensservice.UsernameDetail, rawMessageHandler RawMessageHandler) error {
	syncMessage := &protobuf.SyncEnsUsernameDetail{
		Clock:    usernameDetail.Clock,
		Username: usernameDetail.Username,
		ChainId:  usernameDetail.ChainID,
		Removed:  usernameDetail.Removed,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         contactIDFromPublicKey(&m.identity.PublicKey),
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ENS_USERNAME_DETAIL,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	return err
}

func (m *Messenger) syncAccountCustomizationColor(ctx context.Context, acc *multiaccounts.Account) error {
	if !m.hasPairedDevices() {
		return nil
	}

	_, chat := m.getLastClockWithRelatedChat()

	message := &protobuf.SyncAccountCustomizationColor{
		KeyUid:             acc.KeyUID,
		CustomizationColor: string(acc.CustomizationColor),
		UpdatedAt:          acc.CustomizationColorClock,
	}

	encodedMessage, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_ACCOUNT_CUSTOMIZATION_COLOR,
		ResendAutomatically: true,
	}

	_, err = m.dispatchMessage(ctx, rawMessage)
	return err
}

func (m *Messenger) SyncTrustedUser(ctx context.Context, publicKey string, ts verification.TrustStatus, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncTrustedUser{
		Clock:  clock,
		Id:     publicKey,
		Status: protobuf.SyncTrustedUser_TrustStatus(ts),
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_TRUSTED_USER,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) SyncVerificationRequest(ctx context.Context, vr *verification.Request, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncVerificationRequest{
		Id:                 vr.ID,
		Clock:              clock,
		From:               vr.From,
		To:                 vr.To,
		Challenge:          vr.Challenge,
		Response:           vr.Response,
		RequestedAt:        vr.RequestedAt,
		RepliedAt:          vr.RepliedAt,
		VerificationStatus: protobuf.SyncVerificationRequest_VerificationStatus(vr.RequestStatus),
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_VERIFICATION_REQUEST,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

// RetrieveAll retrieves messages from all filters, processes them and returns a
// MessengerResponse to the client
func (m *Messenger) RetrieveAll() (*MessengerResponse, error) {
	chatWithMessages, err := m.transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}

	return m.handleRetrievedMessages(chatWithMessages, true)
}

func (m *Messenger) GetStats() types.StatsSummary {
	return m.transport.GetStats()
}

type CurrentMessageState struct {
	// Message is the protobuf message received
	Message protobuf.ChatMessage
	// MessageID is the ID of the message
	MessageID string
	// WhisperTimestamp is the whisper timestamp of the message
	WhisperTimestamp uint64
	// Contact is the contact associated with the author of the message
	Contact *Contact
	// PublicKey is the public key of the author of the message
	PublicKey *ecdsa.PublicKey
}

type ReceivedMessageState struct {
	// State on the message being processed
	CurrentMessageState *CurrentMessageState
	// AllChats in memory
	AllChats *chatMap
	// All contacts in memory
	AllContacts *contactMap
	// List of contacts modified
	ModifiedContacts *stringBoolMap
	// All installations in memory
	AllInstallations *installationMap
	// List of communities modified
	ModifiedInstallations *stringBoolMap
	// Map of existing messages
	ExistingMessagesMap map[string]bool
	// EmojiReactions is a list of emoji reactions for the current batch
	// indexed by from-message-id-emoji-type
	EmojiReactions map[string]*EmojiReaction
	// GroupChatInvitations is a list of invitation requests or rejections
	GroupChatInvitations map[string]*GroupChatInvitation
	// Response to the client
	Response *MessengerResponse
	// Timesource is a time source for clock values/timestamps.
	Timesource              common.TimeSource
	AllBookmarks            map[string]*browsers.Bookmark
	AllVerificationRequests []*verification.Request
	AllTrustStatus          map[string]verification.TrustStatus
}

func (m *Messenger) markDeliveredMessages(acks [][]byte) {
	for _, ack := range acks {
		//get message ID from database by datasync ID, with at-least-one
		// semantic
		messageIDBytes, err := m.persistence.MarkAsConfirmed(ack, true)
		if err != nil {
			m.logger.Info("got datasync acknowledge for message we don't have in db", zap.String("ack", hex.EncodeToString(ack)))
			continue
		}

		messageID := messageIDBytes.String()
		//mark messages as delivered

		err = m.UpdateMessageOutgoingStatus(messageID, common.OutgoingStatusDelivered)
		if err != nil {
			m.logger.Debug("Can't set message status as delivered", zap.Error(err))
		}

		//send signal to client that message status updated
		if m.config.messengerSignalsHandler != nil {
			message, err := m.persistence.MessageByID(messageID)
			if err != nil {
				m.logger.Debug("Can't get message from database", zap.Error(err))
				continue
			}
			m.config.messengerSignalsHandler.MessageDelivered(message.LocalChatID, messageID)
		}
	}
}

// addNewMessageNotification takes a common.Message and generates a new NotificationBody and appends it to the
// []Response.Notifications if the message is m.New
func (r *ReceivedMessageState) addNewMessageNotification(publicKey ecdsa.PublicKey, m *common.Message, responseTo *common.Message, profilePicturesVisibility int) error {
	if !m.New {
		return nil
	}

	pubKey, err := m.GetSenderPubKey()
	if err != nil {
		return err
	}
	contactID := contactIDFromPublicKey(pubKey)

	chat, ok := r.AllChats.Load(m.LocalChatID)
	if !ok {
		return fmt.Errorf("chat ID '%s' not present", m.LocalChatID)
	}

	contact, ok := r.AllContacts.Load(contactID)
	if !ok {
		return fmt.Errorf("contact ID '%s' not present", contactID)
	}

	if !chat.Muted {
		if showMessageNotification(publicKey, m, chat, responseTo) {
			notification, err := NewMessageNotification(m.ID, m, chat, contact, r.AllContacts, profilePicturesVisibility)
			if err != nil {
				return err
			}
			r.Response.AddNotification(notification)
		}
	}

	return nil
}

// updateExistingActivityCenterNotification updates AC notification if it exists and hasn't been read yet
func (r *ReceivedMessageState) updateExistingActivityCenterNotification(publicKey ecdsa.PublicKey, m *Messenger, message *common.Message, responseTo *common.Message) error {
	notification, err := m.persistence.GetActivityCenterNotificationByID(types.FromHex(message.ID))
	if err != nil {
		return err
	}

	if notification == nil || notification.Read {
		return nil
	}

	notification.Message = message
	notification.ReplyMessage = responseTo
	notification.UpdatedAt = m.getCurrentTimeInMillis()

	err = m.addActivityCenterNotification(r.Response, notification)
	if err != nil {
		return err
	}

	return nil
}

// addNewActivityCenterNotification takes a common.Message and generates a new ActivityCenterNotification and appends it to the
// []Response.ActivityCenterNotifications if the message is m.New
func (r *ReceivedMessageState) addNewActivityCenterNotification(publicKey ecdsa.PublicKey, m *Messenger, message *common.Message, responseTo *common.Message) error {
	if !message.New {
		return nil
	}

	chat, ok := r.AllChats.Load(message.LocalChatID)
	if !ok {
		return fmt.Errorf("chat ID '%s' not present", message.LocalChatID)
	}

	// Use albumId as notificationId to prevent multiple notifications
	// for same message with multiple images
	var idToUse string

	if message.GetImage() != nil {
		idToUse = message.GetImage().GetAlbumId()
	} else {
		idToUse = message.ID
	}

	isNotification, notificationType := showMentionOrReplyActivityCenterNotification(publicKey, message, chat, responseTo)
	if isNotification {
		notification := &ActivityCenterNotification{
			ID:           types.FromHex(idToUse),
			Name:         chat.Name,
			Message:      message,
			ReplyMessage: responseTo,
			Type:         notificationType,
			Timestamp:    message.WhisperTimestamp,
			ChatID:       chat.ID,
			CommunityID:  chat.CommunityID,
			Author:       message.From,
			UpdatedAt:    m.getCurrentTimeInMillis(),
		}

		return m.addActivityCenterNotification(r.Response, notification)
	}

	return nil
}

func (m *Messenger) buildMessageState() *ReceivedMessageState {
	return &ReceivedMessageState{
		AllChats:              m.allChats,
		AllContacts:           m.allContacts,
		ModifiedContacts:      new(stringBoolMap),
		AllInstallations:      m.allInstallations,
		ModifiedInstallations: m.modifiedInstallations,
		ExistingMessagesMap:   make(map[string]bool),
		EmojiReactions:        make(map[string]*EmojiReaction),
		GroupChatInvitations:  make(map[string]*GroupChatInvitation),
		Response:              &MessengerResponse{},
		Timesource:            m.getTimesource(),
		AllBookmarks:          make(map[string]*browsers.Bookmark),
		AllTrustStatus:        make(map[string]verification.TrustStatus),
	}
}

func (m *Messenger) outputToCSV(timestamp uint32, messageID types.HexBytes, from string, topic types.TopicType, chatID string, msgType protobuf.ApplicationMetadataMessage_Type, parsedMessage interface{}) {
	if !m.outputCSV {
		return
	}

	msgJSON, err := json.Marshal(parsedMessage)
	if err != nil {
		m.logger.Error("could not marshall message", zap.Error(err))
		return
	}

	line := fmt.Sprintf("%d\t%s\t%s\t%s\t%s\t%s\t%s\n", timestamp, messageID.String(), from, topic.String(), chatID, msgType, msgJSON)
	_, err = m.csvFile.Write([]byte(line))
	if err != nil {
		m.logger.Error("could not write to csv", zap.Error(err))
		return
	}
}

func (m *Messenger) handleImportedMessages(messagesToHandle map[transport.Filter][]*types.Message) error {

	messageState := m.buildMessageState()

	logger := m.logger.With(zap.String("site", "handleImportedMessages"))

	for filter, messages := range messagesToHandle {
		for _, shhMessage := range messages {

			statusMessages, _, err := m.sender.HandleMessages(shhMessage)
			if err != nil {
				logger.Info("failed to decode messages", zap.Error(err))
				continue
			}

			for _, msg := range statusMessages {
				logger := logger.With(zap.String("message-id", msg.TransportMessage.ThirdPartyID))
				logger.Debug("processing message")

				publicKey := msg.SigPubKey()
				senderID := contactIDFromPublicKey(publicKey)

				// Don't process duplicates
				messageID := msg.TransportMessage.ThirdPartyID
				exists, err := m.messageExists(messageID, messageState.ExistingMessagesMap)
				if err != nil {
					logger.Warn("failed to check message exists", zap.Error(err))
				}
				if exists {
					logger.Debug("messageExists", zap.String("messageID", messageID))
					continue
				}

				var contact *Contact
				if c, ok := messageState.AllContacts.Load(senderID); ok {
					contact = c
				} else {
					c, err := buildContact(senderID, publicKey)
					if err != nil {
						logger.Info("failed to build contact", zap.Error(err))
						continue
					}
					contact = c
					messageState.AllContacts.Store(senderID, contact)
				}
				messageState.CurrentMessageState = &CurrentMessageState{
					MessageID:        messageID,
					WhisperTimestamp: uint64(msg.TransportMessage.Timestamp) * 1000,
					Contact:          contact,
					PublicKey:        publicKey,
				}

				if msg.ParsedMessage != nil {

					logger.Debug("Handling parsed message")

					switch msg.ParsedMessage.Interface().(type) {

					case protobuf.ChatMessage:
						logger.Debug("Handling ChatMessage")
						messageState.CurrentMessageState.Message = msg.ParsedMessage.Interface().(protobuf.ChatMessage)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, messageState.CurrentMessageState.Message)
						err = m.HandleImportedChatMessage(messageState)
						if err != nil {
							logger.Warn("failed to handle ChatMessage", zap.Error(err))
							continue
						}
					}
				}
			}
		}
	}

	importMessageAuthors := messageState.Response.DiscordMessageAuthors()
	if len(importMessageAuthors) > 0 {
		err := m.persistence.SaveDiscordMessageAuthors(importMessageAuthors)
		if err != nil {
			return err
		}
	}

	importMessagesToSave := messageState.Response.DiscordMessages()
	if len(importMessagesToSave) > 0 {
		m.communitiesManager.LogStdout(fmt.Sprintf("saving %d discord messages", len(importMessagesToSave)))
		m.handleImportMessagesMutex.Lock()
		err := m.persistence.SaveDiscordMessages(importMessagesToSave)
		if err != nil {
			m.communitiesManager.LogStdout("failed to save discord messages", zap.Error(err))
			m.handleImportMessagesMutex.Unlock()
			return err
		}
		m.handleImportMessagesMutex.Unlock()
	}

	messageAttachmentsToSave := messageState.Response.DiscordMessageAttachments()
	if len(messageAttachmentsToSave) > 0 {
		m.communitiesManager.LogStdout(fmt.Sprintf("saving %d discord message attachments", len(messageAttachmentsToSave)))
		m.handleImportMessagesMutex.Lock()
		err := m.persistence.SaveDiscordMessageAttachments(messageAttachmentsToSave)
		if err != nil {
			m.communitiesManager.LogStdout("failed to save discord message attachments", zap.Error(err))
			m.handleImportMessagesMutex.Unlock()
			return err
		}
		m.handleImportMessagesMutex.Unlock()
	}

	messagesToSave := messageState.Response.Messages()
	if len(messagesToSave) > 0 {
		m.communitiesManager.LogStdout(fmt.Sprintf("saving %d app messages", len(messagesToSave)))
		m.handleMessagesMutex.Lock()
		err := m.SaveMessages(messagesToSave)
		if err != nil {
			m.handleMessagesMutex.Unlock()
			return err
		}
		m.handleMessagesMutex.Unlock()
	}

	// Save chats if they were modified
	if len(messageState.Response.chats) > 0 {
		err := m.saveChats(messageState.Response.Chats())
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) handleRetrievedMessages(chatWithMessages map[transport.Filter][]*types.Message, storeWakuMessages bool) (*MessengerResponse, error) {

	m.handleMessagesMutex.Lock()
	defer m.handleMessagesMutex.Unlock()

	messageState := m.buildMessageState()

	logger := m.logger.With(zap.String("site", "RetrieveAll"))

	adminCommunitiesChatIDs, err := m.communitiesManager.GetAdminCommunitiesChatIDs()
	if err != nil {
		logger.Info("failed to retrieve admin communities", zap.Error(err))
	}

	for filter, messages := range chatWithMessages {
		var processedMessages []string
		for _, shhMessage := range messages {
			logger := logger.With(zap.String("hash", types.EncodeHex(shhMessage.Hash)))
			// Indicates tha all messages in the batch have been processed correctly
			allMessagesProcessed := true

			if adminCommunitiesChatIDs[filter.ChatID] && storeWakuMessages {
				logger.Debug("storing waku message")
				err := m.communitiesManager.StoreWakuMessage(shhMessage)
				if err != nil {
					logger.Warn("failed to store waku message", zap.Error(err))
				}
			}

			statusMessages, acks, err := m.sender.HandleMessages(shhMessage)
			if err != nil {
				logger.Info("failed to decode messages", zap.Error(err))
				continue
			}

			if m.telemetryClient != nil {
				go m.telemetryClient.PushReceivedMessages(filter, shhMessage, statusMessages)
			}
			m.markDeliveredMessages(acks)

			logger.Debug("processing messages further", zap.Int("count", len(statusMessages)))

			for _, msg := range statusMessages {
				logger := logger.With(zap.String("message-id", msg.ID.String()))
				logger.Info("processing message")
				publicKey := msg.SigPubKey()

				m.handleInstallations(msg.Installations)
				err := m.handleSharedSecrets(msg.SharedSecrets)
				if err != nil {
					// log and continue, non-critical error
					logger.Warn("failed to handle shared secrets")
				}

				senderID := contactIDFromPublicKey(publicKey)
				m.logger.Info("processing message", zap.Any("type", msg.Type), zap.String("senderID", senderID))

				contact, contactFound := messageState.AllContacts.Load(senderID)

				if _, ok := m.requestedContacts[senderID]; !ok {
					// Check for messages from blocked users
					if contactFound && contact.Blocked {
						continue
					}
				}

				// Don't process duplicates
				messageID := types.EncodeHex(msg.ID)
				exists, err := m.messageExists(messageID, messageState.ExistingMessagesMap)
				if err != nil {
					logger.Warn("failed to check message exists", zap.Error(err))
				}
				if exists {
					logger.Debug("messageExists", zap.String("messageID", messageID))
					continue
				}

				if !contactFound {
					c, err := buildContact(senderID, publicKey)
					if err != nil {
						logger.Info("failed to build contact", zap.Error(err))
						allMessagesProcessed = false
						continue
					}
					contact = c
					messageState.AllContacts.Store(senderID, contact)
					m.forgetContactInfoRequest(senderID)
				}
				messageState.CurrentMessageState = &CurrentMessageState{
					MessageID:        messageID,
					WhisperTimestamp: uint64(msg.TransportMessage.Timestamp) * 1000,
					Contact:          contact,
					PublicKey:        publicKey,
				}

				if msg.ParsedMessage != nil {

					logger.Debug("Handling parsed message")

					switch msg.ParsedMessage.Interface().(type) {
					case protobuf.MembershipUpdateMessage:
						logger.Debug("Handling MembershipUpdateMessage")
						rawMembershipUpdate := msg.ParsedMessage.Interface().(protobuf.MembershipUpdateMessage)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, rawMembershipUpdate)

						chat, _ := messageState.AllChats.Load(rawMembershipUpdate.ChatId)
						err = m.HandleMembershipUpdate(messageState, chat, rawMembershipUpdate, m.systemMessagesTranslations)
						if err != nil {
							logger.Warn("failed to handle MembershipUpdate", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.ChatMessage:
						logger.Debug("Handling ChatMessage")
						messageState.CurrentMessageState.Message = msg.ParsedMessage.Interface().(protobuf.ChatMessage)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, messageState.CurrentMessageState.Message)
						err = m.HandleChatMessage(messageState)
						if err != nil {
							logger.Warn("failed to handle ChatMessage", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.EditMessage:
						logger.Debug("Handling EditMessage")
						editProto := msg.ParsedMessage.Interface().(protobuf.EditMessage)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, editProto)
						editMessage := EditMessage{
							EditMessage: editProto,
							From:        contact.ID,
							ID:          messageID,
							SigPubKey:   publicKey,
						}
						err = m.HandleEditMessage(messageState, editMessage)
						if err != nil {
							logger.Warn("failed to handle EditMessage", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.DeleteMessage:
						logger.Debug("Handling DeleteMessage")
						deleteProto := msg.ParsedMessage.Interface().(protobuf.DeleteMessage)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, deleteProto)
						deleteMessage := DeleteMessage{
							DeleteMessage: deleteProto,
							From:          contact.ID,
							ID:            messageID,
							SigPubKey:     publicKey,
						}

						err = m.HandleDeleteMessage(messageState, deleteMessage)
						if err != nil {
							logger.Warn("failed to handle DeleteMessage", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.DeleteForMeMessage:
						logger.Debug("Handling DeleteForMeMessage")
						deleteForMeProto := msg.ParsedMessage.Interface().(protobuf.DeleteForMeMessage)
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, deleteForMeProto)

						err = m.HandleDeleteForMeMessage(messageState, deleteForMeProto)
						if err != nil {
							logger.Warn("failed to handle DeleteForMeMessage", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.PinMessage:
						pinMessage := msg.ParsedMessage.Interface().(protobuf.PinMessage)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, pinMessage)
						err = m.HandlePinMessage(messageState, pinMessage)
						if err != nil {
							logger.Warn("failed to handle PinMessage", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.PairInstallation:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}
						p := msg.ParsedMessage.Interface().(protobuf.PairInstallation)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling PairInstallation", zap.Any("message", p))
						err = m.HandlePairInstallation(messageState, p)
						if err != nil {
							logger.Warn("failed to handle PairInstallation", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.StatusUpdate:
						p := msg.ParsedMessage.Interface().(protobuf.StatusUpdate)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling StatusUpdate", zap.Any("message", p))
						err = m.HandleStatusUpdate(messageState, p)
						if err != nil {
							logger.Warn("failed to handle StatusMessage", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncInstallationContact:
						logger.Warn("SyncInstallationContact is not supported")
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, msg.ParsedMessage.Interface().(protobuf.SyncInstallationContact))
						continue

					case protobuf.SyncInstallationContactV2:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncInstallationContactV2)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncInstallationContact", zap.Any("message", p))
						err = m.HandleSyncInstallationContact(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncInstallationContact", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncProfilePictures:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncProfilePictures)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncProfilePicture", zap.Any("message", p))
						err = m.HandleSyncProfilePictures(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncProfilePicture", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncBookmark:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncBookmark)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncBookmark", zap.Any("message", p))
						err = m.handleSyncBookmark(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncBookmark", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncClearHistory:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncClearHistory)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncClearHistory", zap.Any("message", p))
						err = m.handleSyncClearHistory(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncClearHistory", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.SyncCommunitySettings:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}
						p := msg.ParsedMessage.Interface().(protobuf.SyncCommunitySettings)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncCommunitySettings", zap.Any("message", p))
						err = m.handleSyncCommunitySettings(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncCommunitySettings", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncTrustedUser:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncTrustedUser)
						logger.Debug("Handling SyncTrustedUser", zap.Any("message", p))
						err = m.handleSyncTrustedUser(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncTrustedUser", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncVerificationRequest:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncVerificationRequest)
						logger.Debug("Handling SyncVerificationRequest", zap.Any("message", p))
						err = m.handleSyncVerificationRequest(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncClearHistory", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.Backup:
						if !m.processBackedupMessages {
							continue
						}
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.Backup)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling Backup", zap.Any("message", p))
						errors := m.HandleBackup(messageState, p)
						if len(errors) > 0 {
							for _, err := range errors {
								logger.Warn("failed to handle Backup", zap.Error(err))
							}
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncInstallationPublicChat:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncInstallationPublicChat)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncInstallationPublicChat", zap.Any("message", p))
						addedChat := m.HandleSyncInstallationPublicChat(messageState, p)

						// We join and re-register as we want to receive mentions from the newly joined public chat
						if addedChat != nil {
							_, err = m.createPublicChat(addedChat.ID, messageState.Response)
							if err != nil {
								allMessagesProcessed = false
								logger.Error("error joining chat", zap.Error(err))
								continue
							}
						}

					case protobuf.SyncChatRemoved:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncChatRemoved)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncChatRemoved", zap.Any("message", p))
						err := m.HandleSyncChatRemoved(messageState, p)
						if err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle sync removing chat", zap.Error(err))
							continue
						}

					case protobuf.SyncChatMessagesRead:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncChatMessagesRead)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncChatMessagesRead", zap.Any("message", p))
						err := m.HandleSyncChatMessagesRead(messageState, p)
						if err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle sync chat message read", zap.Error(err))
							continue
						}

					case protobuf.SyncCommunity:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						community := msg.ParsedMessage.Interface().(protobuf.SyncCommunity)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, community)
						logger.Debug("Handling SyncCommunity", zap.Any("message", community))

						err = m.handleSyncCommunity(messageState, community)
						if err != nil {
							logger.Warn("failed to handle SyncCommunity", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncActivityCenterRead:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						a := msg.ParsedMessage.Interface().(protobuf.SyncActivityCenterRead)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, a)
						logger.Debug("Handling SyncActivityCenterRead", zap.Any("message", a))

						err = m.handleActivityCenterRead(messageState, a)
						if err != nil {
							logger.Warn("failed to handle SyncActivityCenterRead", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncActivityCenterAccepted:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						a := msg.ParsedMessage.Interface().(protobuf.SyncActivityCenterAccepted)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, a)
						logger.Debug("Handling SyncActivityCenterAccepted", zap.Any("message", a))

						err = m.handleActivityCenterAccepted(messageState, a)
						if err != nil {
							logger.Warn("failed to handle SyncActivityCenterAccepted", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncActivityCenterDismissed:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						a := msg.ParsedMessage.Interface().(protobuf.SyncActivityCenterDismissed)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, a)
						logger.Debug("Handling SyncActivityCenterDismissed", zap.Any("message", a))

						err = m.handleActivityCenterDismissed(messageState, a)
						if err != nil {
							logger.Warn("failed to handle SyncActivityCenterDismissed", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncActivityCenterNotifications:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						a := msg.ParsedMessage.Interface().(protobuf.SyncActivityCenterNotifications)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, a)
						logger.Debug("Handling SyncActivityCenterNotification", zap.Any("message", a))

						err = m.handleSyncActivityCenterNotifications(messageState, &a)
						if err != nil {
							logger.Warn("failed to handle SyncActivityCenterNotification", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncActivityCenterNotificationState:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						a := msg.ParsedMessage.Interface().(protobuf.SyncActivityCenterNotificationState)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, a)
						logger.Debug("Handling SyncActivityCenterNotificationState", zap.Any("message", a))

						err = m.handleSyncActivityCenterNotificationState(messageState, &a)
						if err != nil {
							logger.Warn("failed to handle SyncActivityCenterNotificationState", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncSetting:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						ss := msg.ParsedMessage.Interface().(protobuf.SyncSetting)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, ss)
						logger.Debug("Handling SyncSetting", zap.Any("message", ss))

						err := m.handleSyncSetting(messageState, &ss)
						if err != nil {
							logger.Warn("failed to handle SyncSetting", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncAccountCustomizationColor:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}
						sac := msg.ParsedMessage.Interface().(protobuf.SyncAccountCustomizationColor)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, sac)
						logger.Debug("Handling SyncAccountCustomizationColor", zap.Any("message", sac))

						err := m.handleSyncAccountCustomizationColor(messageState, sac)
						if err != nil {
							logger.Warn("failed to handle SyncAccountCustomizationColor", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.RequestAddressForTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.RequestAddressForTransaction)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, command)
						logger.Debug("Handling RequestAddressForTransaction", zap.Any("message", command))
						err = m.HandleRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle RequestAddressForTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SendTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.SendTransaction)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, command)
						logger.Debug("Handling SendTransaction", zap.Any("message", command))
						err = m.HandleSendTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle SendTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.AcceptRequestAddressForTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.AcceptRequestAddressForTransaction)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, command)
						logger.Debug("Handling AcceptRequestAddressForTransaction")
						err = m.HandleAcceptRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle AcceptRequestAddressForTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.DeclineRequestAddressForTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.DeclineRequestAddressForTransaction)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, command)
						logger.Debug("Handling DeclineRequestAddressForTransaction")
						err = m.HandleDeclineRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle DeclineRequestAddressForTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.DeclineRequestTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.DeclineRequestTransaction)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, command)
						logger.Debug("Handling DeclineRequestTransaction")
						err = m.HandleDeclineRequestTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle DeclineRequestTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.RequestTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.RequestTransaction)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, command)
						logger.Debug("Handling RequestTransaction")
						err = m.HandleRequestTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle RequestTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.ContactUpdate:
						if common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("coming from us, ignoring")
							continue
						}

						contactUpdate := msg.ParsedMessage.Interface().(protobuf.ContactUpdate)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, contactUpdate)
						err = m.HandleContactUpdate(messageState, contactUpdate)
						if err != nil {
							logger.Warn("failed to handle ContactUpdate", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
						m.forgetContactInfoRequest(senderID)

					case protobuf.AcceptContactRequest:
						logger.Debug("Handling AcceptContactRequest")
						message := msg.ParsedMessage.Interface().(protobuf.AcceptContactRequest)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						err = m.HandleAcceptContactRequest(messageState, message, senderID)
						if err != nil {
							logger.Warn("failed to handle AcceptContactRequest", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.RetractContactRequest:
						logger.Debug("Handling RetractContactRequest")
						message := msg.ParsedMessage.Interface().(protobuf.RetractContactRequest)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						err = m.HandleRetractContactRequest(messageState, message)
						if err != nil {
							logger.Warn("failed to handle RetractContactRequest", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.PushNotificationQuery:
						logger.Debug("Received PushNotificationQuery")
						if m.pushNotificationServer == nil {
							continue
						}
						message := msg.ParsedMessage.Interface().(protobuf.PushNotificationQuery)
						logger.Debug("Handling PushNotificationQuery")
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						if err := m.pushNotificationServer.HandlePushNotificationQuery(publicKey, msg.ID, message); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationQuery", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue
					case protobuf.PushNotificationRegistrationResponse:
						logger.Debug("Received PushNotificationRegistrationResponse")
						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling PushNotificationRegistrationResponse")
						message := msg.ParsedMessage.Interface().(protobuf.PushNotificationRegistrationResponse)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						if err := m.pushNotificationClient.HandlePushNotificationRegistrationResponse(publicKey, message); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationRegistrationResponse", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue
					case protobuf.ContactCodeAdvertisement:
						logger.Debug("Received ContactCodeAdvertisement")

						cca := msg.ParsedMessage.Interface().(protobuf.ContactCodeAdvertisement)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, cca)
						logger.Debug("protobuf.ContactCodeAdvertisement received", zap.Any("cca", cca))
						if cca.ChatIdentity != nil {

							logger.Debug("Received ContactCodeAdvertisement ChatIdentity")
							err = m.HandleChatIdentity(messageState, *cca.ChatIdentity)
							if err != nil {
								allMessagesProcessed = false
								logger.Warn("failed to handle ContactCodeAdvertisement ChatIdentity", zap.Error(err))
								// No continue as Chat Identity may fail but the rest of the cca may process fine.
							}
						}

						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling ContactCodeAdvertisement")
						if err := m.pushNotificationClient.HandleContactCodeAdvertisement(publicKey, cca); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle ContactCodeAdvertisement", zap.Error(err))
						}

						// We continue in any case, no changes to messenger
						continue

					case protobuf.PushNotificationResponse:
						logger.Debug("Received PushNotificationResponse")
						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling PushNotificationResponse")
						message := msg.ParsedMessage.Interface().(protobuf.PushNotificationResponse)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						if err := m.pushNotificationClient.HandlePushNotificationResponse(publicKey, message); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationResponse", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue

					case protobuf.PushNotificationQueryResponse:
						logger.Debug("Received PushNotificationQueryResponse")
						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling PushNotificationQueryResponse")
						message := msg.ParsedMessage.Interface().(protobuf.PushNotificationQueryResponse)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						if err := m.pushNotificationClient.HandlePushNotificationQueryResponse(publicKey, message); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationQueryResponse", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue

					case protobuf.PushNotificationRequest:
						logger.Debug("Received PushNotificationRequest")
						if m.pushNotificationServer == nil {
							continue
						}
						logger.Debug("Handling PushNotificationRequest")
						message := msg.ParsedMessage.Interface().(protobuf.PushNotificationRequest)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						if err := m.pushNotificationServer.HandlePushNotificationRequest(publicKey, msg.ID, message); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationRequest", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue
					case protobuf.EmojiReaction:
						logger.Debug("Handling EmojiReaction")
						message := msg.ParsedMessage.Interface().(protobuf.EmojiReaction)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						err = m.HandleEmojiReaction(messageState, message)
						if err != nil {
							logger.Warn("failed to handle EmojiReaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.GroupChatInvitation:
						logger.Debug("Handling GroupChatInvitation")
						message := msg.ParsedMessage.Interface().(protobuf.GroupChatInvitation)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						err = m.HandleGroupChatInvitation(messageState, message)
						if err != nil {
							logger.Warn("failed to handle GroupChatInvitation", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.ChatIdentity:
						message := msg.ParsedMessage.Interface().(protobuf.ChatIdentity)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						err = m.HandleChatIdentity(messageState, message)
						if err != nil {
							logger.Warn("failed to handle ChatIdentity", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.CommunityDescription:
						logger.Debug("Handling CommunityDescription")
						message := msg.ParsedMessage.Interface().(protobuf.CommunityDescription)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						err = m.handleCommunityDescription(messageState, publicKey, message, msg.DecryptedPayload)
						if err != nil {
							logger.Warn("failed to handle CommunityDescription", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

						//if community was among requested ones, send its info and remove filter
						for communityID := range m.requestedCommunities {
							if _, ok := messageState.Response.communities[communityID]; ok {
								m.passStoredCommunityInfoToSignalHandler(communityID)
							}
						}

					case protobuf.RequestContactVerification:
						logger.Debug("Handling RequestContactVerification")
						err = m.HandleRequestContactVerification(messageState, msg.ParsedMessage.Interface().(protobuf.RequestContactVerification))
						if err != nil {
							logger.Warn("failed to handle RequestContactVerification", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.AcceptContactVerification:
						logger.Debug("Handling AcceptContactVerification")
						err = m.HandleAcceptContactVerification(messageState, msg.ParsedMessage.Interface().(protobuf.AcceptContactVerification))
						if err != nil {
							logger.Warn("failed to handle AcceptContactVerification", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.DeclineContactVerification:
						logger.Debug("Handling DeclineContactVerification")
						err = m.HandleDeclineContactVerification(messageState, msg.ParsedMessage.Interface().(protobuf.DeclineContactVerification))
						if err != nil {
							logger.Warn("failed to handle DeclineContactVerification", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.CancelContactVerification:
						logger.Debug("Handling CancelContactVerification")
						err = m.HandleCancelContactVerification(messageState, msg.ParsedMessage.Interface().(protobuf.CancelContactVerification))
						if err != nil {
							logger.Warn("failed to handle CancelContactVerification", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.CommunityInvitation:
						logger.Debug("Handling CommunityInvitation")
						invitation := msg.ParsedMessage.Interface().(protobuf.CommunityInvitation)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, invitation)
						err = m.HandleCommunityInvitation(messageState, publicKey, invitation, invitation.CommunityDescription)
						if err != nil {
							logger.Warn("failed to handle CommunityInvitation", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.CommunityRequestToJoin:
						logger.Debug("Handling CommunityRequestToJoin")
						request := msg.ParsedMessage.Interface().(protobuf.CommunityRequestToJoin)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, request)
						err = m.HandleCommunityRequestToJoin(messageState, publicKey, request)
						if err != nil {
							logger.Warn("failed to handle CommunityRequestToJoin", zap.Error(err))
							continue
						}
					case protobuf.CommunityEditRevealedAccounts:
						logger.Debug("Handling CommunityEditRevealedAccounts")
						request := msg.ParsedMessage.Interface().(protobuf.CommunityEditRevealedAccounts)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, request)
						err = m.HandleCommunityEditSharedAddresses(messageState, publicKey, request)
						if err != nil {
							logger.Warn("failed to handle CommunityEditRevealedAccounts", zap.Error(err))
							continue
						}
					case protobuf.CommunityCancelRequestToJoin:
						logger.Debug("Handling CommunityCancelRequestToJoin")
						request := msg.ParsedMessage.Interface().(protobuf.CommunityCancelRequestToJoin)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, request)
						err = m.HandleCommunityCancelRequestToJoin(messageState, publicKey, request)
						if err != nil {
							logger.Warn("failed to handle CommunityCancelRequestToJoin", zap.Error(err))
							continue
						}
					case protobuf.CommunityRequestToJoinResponse:
						logger.Debug("Handling CommunityRequestToJoinResponse")
						requestToJoinResponse := msg.ParsedMessage.Interface().(protobuf.CommunityRequestToJoinResponse)
						err = m.HandleCommunityRequestToJoinResponse(messageState, publicKey, requestToJoinResponse)
						if err != nil {
							logger.Warn("failed to handle CommunityRequestToJoinResponse", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.CommunityRequestToLeave:
						logger.Debug("Handling CommunityRequestToLeave")
						request := msg.ParsedMessage.Interface().(protobuf.CommunityRequestToLeave)
						err = m.HandleCommunityRequestToLeave(messageState, publicKey, request)
						if err != nil {
							logger.Warn("failed to handle CommunityRequestToLeave", zap.Error(err))
							continue
						}

					case protobuf.CommunityMessageArchiveMagnetlink:
						logger.Debug("Handling CommunityMessageArchiveMagnetlink")
						magnetlinkMessage := msg.ParsedMessage.Interface().(protobuf.CommunityMessageArchiveMagnetlink)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, magnetlinkMessage)
						err = m.HandleHistoryArchiveMagnetlinkMessage(messageState, publicKey, magnetlinkMessage.MagnetUri, magnetlinkMessage.Clock)
						if err != nil {
							logger.Warn("failed to handle CommunityMessageArchiveMagnetlink", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.CommunityEventsMessage:
						logger.Debug("Handling CommunityEventsMessage")
						message := msg.ParsedMessage.Interface().(protobuf.CommunityEventsMessage)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						err = m.handleCommunityEventsMessage(messageState, publicKey, message)
						if err != nil {
							logger.Warn("failed to handle CommunityEvent", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.AnonymousMetricBatch:
						logger.Debug("Handling AnonymousMetricBatch")
						if m.anonMetricsServer == nil {
							logger.Warn("unable to handle AnonymousMetricBatch, anonMetricsServer is nil")
							continue
						}
						message := msg.ParsedMessage.Interface().(protobuf.AnonymousMetricBatch)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, message)
						ams, err := m.anonMetricsServer.StoreMetrics(message)
						if err != nil {
							logger.Warn("failed to store AnonymousMetricBatch", zap.Error(err))
							continue
						}
						messageState.Response.AnonymousMetrics = append(messageState.Response.AnonymousMetrics, ams...)

					case protobuf.SyncKeypair:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncKeypair)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncKeypair", zap.Any("message", p))
						err = m.HandleSyncKeypair(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncKeypair", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.SyncAccount:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncAccount)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncAccount", zap.Any("message", p))
						err = m.HandleSyncWatchOnlyAccount(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncAccount", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.SyncAccountsPositions:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncAccountsPositions)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						logger.Debug("Handling SyncAccountsPositions", zap.Any("message", p))
						err = m.HandleSyncAccountsPositions(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncAccountsPositions", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.SyncContactRequestDecision:
						logger.Info("SyncContactRequestDecision")
						p := msg.ParsedMessage.Interface().(protobuf.SyncContactRequestDecision)
						err := m.HandleSyncContactRequestDecision(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncContactRequestDecision", zap.Error(err))
							continue
						}
					case protobuf.SyncSavedAddress:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncSavedAddress)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						err = m.handleSyncSavedAddress(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncSavedAddress", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.SyncSocialLinks:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncSocialLinks)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						err = m.HandleSyncSocialLinks(messageState, p)
						if err != nil {
							logger.Warn("failed to handle HandleSyncSocialLinks", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.SyncEnsUsernameDetail:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}
						p := msg.ParsedMessage.Interface().(protobuf.SyncEnsUsernameDetail)
						m.outputToCSV(msg.TransportMessage.Timestamp, msg.ID, senderID, filter.Topic, filter.ChatID, msg.Type, p)
						err = m.handleSyncEnsUsernameDetail(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncEnsName", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					default:
						// Check if is an encrypted PushNotificationRegistration
						if msg.Type == protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION {
							logger.Debug("Received PushNotificationRegistration")
							if m.pushNotificationServer == nil {
								continue
							}
							logger.Debug("Handling PushNotificationRegistration")
							if err := m.pushNotificationServer.HandlePushNotificationRegistration(publicKey, msg.ParsedMessage.Interface().([]byte)); err != nil {
								allMessagesProcessed = false
								logger.Warn("failed to handle PushNotificationRegistration", zap.Error(err))
							}
							// We continue in any case, no changes to messenger
							continue
						}

						logger.Debug("message not handled", zap.Any("messageType", reflect.TypeOf(msg.ParsedMessage.Interface())))

					}
				} else {
					logger.Debug("parsed message is nil")
				}
			}

			// Process any community changes
			for _, changes := range messageState.Response.CommunityChanges {
				if changes.ShouldMemberJoin {
					response, err := m.joinCommunity(context.TODO(), changes.Community.ID(), false)
					if err != nil {
						logger.Error("cannot join community", zap.Error(err))
						continue
					}

					if err := messageState.Response.Merge(response); err != nil {
						logger.Error("cannot merge join community response", zap.Error(err))
						continue
					}

				} else if changes.ShouldMemberLeave {
					// this means we've been kicked by the community owner/admin,
					// in this case we don't want to unsubscribe from community updates
					// so we still get notified accordingly when something changes,
					// hence, we're setting `unsubscribeFromCommunity` to `false` here
					response, err := m.leaveCommunity(changes.Community.ID(), false)
					if err != nil {
						logger.Error("cannot leave community", zap.Error(err))
						continue
					}

					if err := messageState.Response.Merge(response); err != nil {
						logger.Error("cannot merge join community response", zap.Error(err))
						continue
					}

					// Activity Center notification
					now := m.getCurrentTimeInMillis()
					notification := &ActivityCenterNotification{
						ID:          types.FromHex(uuid.New().String()),
						Type:        ActivityCenterNotificationTypeCommunityKicked,
						Timestamp:   now,
						CommunityID: changes.Community.IDString(),
						Read:        false,
						UpdatedAt:   now,
					}

					err = m.addActivityCenterNotification(response, notification)
					if err != nil {
						logger.Error("failed to save notification", zap.Error(err))
						continue
					}

					if err := messageState.Response.Merge(response); err != nil {
						logger.Error("cannot merge notification response", zap.Error(err))
						continue
					}
				}
			}

			// Clean up as not used by clients currently
			messageState.Response.CommunityChanges = nil

			// NOTE: for now we confirm messages as processed regardless whether we
			// actually processed them, this is because we need to differentiate
			// from messages that we want to retry to process and messages that
			// are never going to be processed
			m.transport.MarkP2PMessageAsProcessed(gethcommon.BytesToHash(shhMessage.Hash))

			if allMessagesProcessed {
				processedMessages = append(processedMessages, types.EncodeHex(shhMessage.Hash))
			}
		}

		if len(processedMessages) != 0 {
			if err := m.transport.ConfirmMessagesProcessed(processedMessages, m.getTimesource().GetCurrentTime()); err != nil {
				logger.Warn("failed to confirm processed messages", zap.Error(err))
			}
		}
	}

	return m.saveDataAndPrepareResponse(messageState)
}

func (m *Messenger) saveDataAndPrepareResponse(messageState *ReceivedMessageState) (*MessengerResponse, error) {
	var err error
	var contactsToSave []*Contact
	messageState.ModifiedContacts.Range(func(id string, value bool) (shouldContinue bool) {
		contact, ok := messageState.AllContacts.Load(id)
		if ok {
			contactsToSave = append(contactsToSave, contact)
			messageState.Response.AddContact(contact)
		}
		return true
	})

	// Hydrate chat alias and identicon
	for id := range messageState.Response.chats {
		chat, _ := messageState.AllChats.Load(id)
		if chat == nil {
			continue
		}
		if chat.OneToOne() {
			contact, ok := m.allContacts.Load(chat.ID)
			if ok {
				chat.Alias = contact.Alias
				chat.Identicon = contact.Identicon
			}
		}

		messageState.Response.AddChat(chat)
	}

	messageState.ModifiedInstallations.Range(func(id string, value bool) (shouldContinue bool) {
		installation, _ := messageState.AllInstallations.Load(id)
		messageState.Response.Installations = append(messageState.Response.Installations, installation)
		if installation.InstallationMetadata != nil {
			err = m.setInstallationMetadata(id, installation.InstallationMetadata)
			if err != nil {
				return false
			}
		}

		return true
	})
	if err != nil {
		return nil, err
	}

	if len(messageState.Response.chats) > 0 {
		err = m.saveChats(messageState.Response.Chats())
		if err != nil {
			return nil, err
		}
	}

	messagesToSave := messageState.Response.Messages()
	if len(messagesToSave) > 0 {
		err = m.SaveMessages(messagesToSave)
		if err != nil {
			return nil, err
		}
	}

	for _, emojiReaction := range messageState.EmojiReactions {
		messageState.Response.AddEmojiReaction(emojiReaction)
	}

	for _, groupChatInvitation := range messageState.GroupChatInvitations {
		messageState.Response.Invitations = append(messageState.Response.Invitations, groupChatInvitation)
	}

	if len(contactsToSave) > 0 {
		err = m.persistence.SaveContacts(contactsToSave)
		if err != nil {
			return nil, err
		}
	}

	newMessagesIds := map[string]struct{}{}
	for _, message := range messagesToSave {
		if message.New {
			newMessagesIds[message.ID] = struct{}{}
		}
	}

	messagesWithResponses, err := m.pullMessagesAndResponsesFromDB(messagesToSave)
	if err != nil {
		return nil, err
	}
	messagesByID := map[string]*common.Message{}
	for _, message := range messagesWithResponses {
		messagesByID[message.ID] = message
	}
	messageState.Response.SetMessages(messagesWithResponses)

	notificationsEnabled, err := m.settings.GetNotificationsEnabled()
	if err != nil {
		return nil, err
	}

	profilePicturesVisibility, err := m.settings.GetProfilePicturesVisibility()
	if err != nil {
		return nil, err
	}

	m.prepareMessages(messageState.Response.messages)

	for _, message := range messageState.Response.messages {
		if _, ok := newMessagesIds[message.ID]; ok {
			message.New = true

			if notificationsEnabled {
				// Create notification body to be eventually passed to `localnotifications.SendMessageNotifications()`
				if err = messageState.addNewMessageNotification(m.identity.PublicKey, message, messagesByID[message.ResponseTo], profilePicturesVisibility); err != nil {
					return nil, err
				}
			}

			// Create activity center notification body to be eventually passed to `activitycenter.SendActivityCenterNotifications()`
			if err = messageState.addNewActivityCenterNotification(m.identity.PublicKey, m, message, messagesByID[message.ResponseTo]); err != nil {
				return nil, err
			}
		}
	}

	// Reset installations
	m.modifiedInstallations = new(stringBoolMap)

	if len(messageState.AllBookmarks) > 0 {
		bookmarks, err := m.storeSyncBookmarks(messageState.AllBookmarks)
		if err != nil {
			return nil, err
		}
		messageState.Response.AddBookmarks(bookmarks)
	}

	if len(messageState.AllVerificationRequests) > 0 {
		for _, vr := range messageState.AllVerificationRequests {
			messageState.Response.AddVerificationRequest(vr)
		}
	}

	if len(messageState.AllTrustStatus) > 0 {
		messageState.Response.AddTrustStatuses(messageState.AllTrustStatus)
	}

	// Hydrate pinned messages
	for _, pinnedMessage := range messageState.Response.PinMessages() {
		if pinnedMessage.Pinned {
			pinnedMessage.Message = &common.PinnedMessage{
				Message:  messageState.Response.GetMessage(pinnedMessage.MessageId),
				PinnedBy: pinnedMessage.From,
				PinnedAt: pinnedMessage.Clock,
			}
		}
	}

	return messageState.Response, nil
}

func (m *Messenger) storeSyncBookmarks(bookmarkMap map[string]*browsers.Bookmark) ([]*browsers.Bookmark, error) {
	var bookmarks []*browsers.Bookmark
	for _, bookmark := range bookmarkMap {
		bookmarks = append(bookmarks, bookmark)
	}
	return m.browserDatabase.StoreSyncBookmarks(bookmarks)
}

func (m *Messenger) MessageByID(id string) (*common.Message, error) {
	return m.persistence.MessageByID(id)
}

func (m *Messenger) MessagesExist(ids []string) (map[string]bool, error) {
	return m.persistence.MessagesExist(ids)
}

func (m *Messenger) FirstUnseenMessageID(chatID string) (string, error) {
	return m.persistence.FirstUnseenMessageID(chatID)
}

func (m *Messenger) latestIncomingMessageClock(chatID string) (uint64, error) {
	return m.persistence.latestIncomingMessageClock(chatID)
}

func (m *Messenger) MessageByChatID(chatID, cursor string, limit int) ([]*common.Message, string, error) {
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, "", err
	}

	if chat == nil {
		return nil, "", ErrChatNotFound
	}

	var msgs []*common.Message
	var nextCursor string

	if chat.Timeline() {
		var chatIDs = []string{"@" + contactIDFromPublicKey(&m.identity.PublicKey)}
		m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
			if contact.added() {
				chatIDs = append(chatIDs, "@"+contact.ID)
			}
			return true
		})
		msgs, nextCursor, err = m.persistence.MessageByChatIDs(chatIDs, cursor, limit)
		if err != nil {
			return nil, "", err
		}
	} else {
		msgs, nextCursor, err = m.persistence.MessageByChatID(chatID, cursor, limit)
		if err != nil {
			return nil, "", err
		}

	}
	if m.httpServer != nil {
		for idx := range msgs {
			m.prepareMessage(msgs[idx], m.httpServer)
		}
	}

	return msgs, nextCursor, nil
}

func (m *Messenger) prepareMessages(messages map[string]*common.Message) {
	if m.httpServer != nil {
		for idx := range messages {
			m.prepareMessage(messages[idx], m.httpServer)
		}
	}
}

func (m *Messenger) prepareMessage(msg *common.Message, s *server.MediaServer) {
	if msg.QuotedMessage != nil && msg.QuotedMessage.ContentType == int64(protobuf.ChatMessage_IMAGE) {
		msg.QuotedMessage.ImageLocalURL = s.MakeImageURL(msg.QuotedMessage.ID)
	}
	if msg.QuotedMessage != nil && msg.QuotedMessage.ContentType == int64(protobuf.ChatMessage_AUDIO) {
		msg.QuotedMessage.AudioLocalURL = s.MakeAudioURL(msg.QuotedMessage.ID)
	}
	if msg.QuotedMessage != nil && msg.QuotedMessage.ContentType == int64(protobuf.ChatMessage_STICKER) {
		msg.QuotedMessage.HasSticker = true
	}
	if msg.QuotedMessage != nil && msg.QuotedMessage.ContentType == int64(protobuf.ChatMessage_DISCORD_MESSAGE) {
		dm := msg.QuotedMessage.DiscordMessage
		exists, err := m.persistence.HasDiscordMessageAuthorImagePayload(dm.Author.Id)
		if err != nil {
			return
		}

		if exists {
			msg.QuotedMessage.DiscordMessage.Author.LocalUrl = s.MakeDiscordAuthorAvatarURL(dm.Author.Id)
		}
	}

	if msg.ContentType == protobuf.ChatMessage_IMAGE {
		msg.ImageLocalURL = s.MakeImageURL(msg.ID)
	}

	if msg.ContentType == protobuf.ChatMessage_DISCORD_MESSAGE {

		dm := msg.GetDiscordMessage()
		exists, err := m.persistence.HasDiscordMessageAuthorImagePayload(dm.Author.Id)
		if err != nil {
			return
		}

		if exists {
			dm.Author.LocalUrl = s.MakeDiscordAuthorAvatarURL(dm.Author.Id)
		}

		for idx, attachment := range dm.Attachments {
			if strings.Contains(attachment.ContentType, "image") {
				hasPayload, err := m.persistence.HasDiscordMessageAttachmentPayload(attachment.Id, dm.Id)
				if err != nil {
					m.logger.Error("failed to check if message attachment exist", zap.Error(err))
					continue
				}
				if hasPayload {
					localURL := s.MakeDiscordAttachmentURL(dm.Id, attachment.Id)
					dm.Attachments[idx].LocalUrl = localURL
				}
			}
		}
		msg.Payload = &protobuf.ChatMessage_DiscordMessage{
			DiscordMessage: dm,
		}
	}
	if msg.ContentType == protobuf.ChatMessage_AUDIO {
		msg.AudioLocalURL = s.MakeAudioURL(msg.ID)
	}
	if msg.ContentType == protobuf.ChatMessage_STICKER {
		msg.StickerLocalURL = s.MakeStickerURL(msg.GetSticker().Hash)
	}

	msg.LinkPreviews = msg.ConvertFromProtoToLinkPreviews(s.MakeLinkPreviewThumbnailURL)
}

func (m *Messenger) AllMessageByChatIDWhichMatchTerm(chatID string, searchTerm string, caseSensitive bool) ([]*common.Message, error) {
	_, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, err
	}

	return m.persistence.AllMessageByChatIDWhichMatchTerm(chatID, searchTerm, caseSensitive)
}

func (m *Messenger) AllMessagesFromChatsAndCommunitiesWhichMatchTerm(communityIds []string, chatIds []string, searchTerm string, caseSensitive bool) ([]*common.Message, error) {
	return m.persistence.AllMessagesFromChatsAndCommunitiesWhichMatchTerm(communityIds, chatIds, searchTerm, caseSensitive)
}

func (m *Messenger) SaveMessages(messages []*common.Message) error {
	return m.persistence.SaveMessages(messages)
}

func (m *Messenger) DeleteMessage(id string) error {
	return m.persistence.DeleteMessage(id)
}

func (m *Messenger) DeleteMessagesByChatID(id string) error {
	return m.persistence.DeleteMessagesByChatID(id)
}

// MarkMessagesSeen marks messages with `ids` as seen in the chat `chatID`.
// It returns the number of affected messages or error. If there is an error,
// the number of affected messages is always zero.
func (m *Messenger) MarkMessagesSeen(chatID string, ids []string) (uint64, uint64, error) {
	count, countWithMentions, err := m.persistence.MarkMessagesSeen(chatID, ids)
	if err != nil {
		return 0, 0, err
	}
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return 0, 0, err
	}
	m.allChats.Store(chatID, chat)
	return count, countWithMentions, nil
}

func (m *Messenger) syncChatMessagesRead(ctx context.Context, chatID string, clock uint64, rawMessageHandler RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	_, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncChatMessagesRead{
		Clock: clock,
		Id:    chatID,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_CHAT_MESSAGES_READ,
		ResendAutomatically: true,
	}

	_, err = rawMessageHandler(ctx, rawMessage)

	return err
}

func (m *Messenger) markAllRead(chatID string, clock uint64, shouldBeSynced bool) error {
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return errors.New("chat not found")
	}

	_, _, err := m.persistence.MarkAllRead(chatID, clock)
	if err != nil {
		return err
	}

	if shouldBeSynced {
		err := m.syncChatMessagesRead(context.Background(), chatID, clock, m.dispatchMessage)
		if err != nil {
			return err
		}
	}

	chat.ReadMessagesAtClockValue = clock
	chat.Highlight = false

	chat.UnviewedMessagesCount = 0
	chat.UnviewedMentionsCount = 0

	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chat.ID, chat)
	return m.persistence.SaveChats([]*Chat{chat})
}

func (m *Messenger) MarkAllRead(chatID string) error {
	notifications, err := m.persistence.DismissAllActivityCenterNotificationsFromChatID(chatID, m.getCurrentTimeInMillis())
	if err != nil {
		return err
	}
	err = m.syncActivityCenterNotifications(notifications)
	if err != nil {
		m.logger.Error("MarkAllRead, failed to sync activity center notifications", zap.Error(err))
		return err
	}

	clock, _ := m.latestIncomingMessageClock(chatID)

	if clock == 0 {
		chat, ok := m.allChats.Load(chatID)
		if !ok {
			return errors.New("chat not found")
		}
		clock, _ = chat.NextClockAndTimestamp(m.getTimesource())
	}

	return m.markAllRead(chatID, clock, true)
}

func (m *Messenger) MarkAllReadInCommunity(communityID string) ([]string, error) {
	notifications, err := m.persistence.DismissAllActivityCenterNotificationsFromCommunity(communityID, m.getCurrentTimeInMillis())
	if err != nil {
		return nil, err
	}

	chatIDs, err := m.persistence.AllChatIDsByCommunity(communityID)
	if err != nil {
		return nil, err
	}

	err = m.persistence.MarkAllReadMultiple(chatIDs)
	if err != nil {
		return nil, err
	}

	for _, chatID := range chatIDs {
		chat, ok := m.allChats.Load(chatID)

		if ok {
			chat.UnviewedMessagesCount = 0
			chat.UnviewedMentionsCount = 0
			m.allChats.Store(chat.ID, chat)
		} else {
			err = errors.New(fmt.Sprintf("chat with chatID %s not found", chatID))
		}
	}
	if err != nil {
		return chatIDs, err
	}

	err = m.syncActivityCenterNotifications(notifications)
	if err != nil {
		m.logger.Error("MarkAllReadInCommunity, error syncing activity center notifications", zap.Error(err))
	}

	return chatIDs, err
}

// MuteChat signals to the messenger that we don't want to be notified
// on new messages from this chat
func (m *Messenger) MuteChat(request *requests.MuteChat) (time.Time, error) {
	chat, ok := m.allChats.Load(request.ChatID)
	if !ok {
		// Only one to one chan be muted when it's not in the database
		publicKey, err := common.HexToPubkey(request.ChatID)
		if err != nil {
			return time.Time{}, err
		}

		// Create a one to one chat and set active to false
		chat = CreateOneToOneChat(request.ChatID, publicKey, m.getTimesource())
		chat.Active = false
		err = m.initChatSyncFields(chat)
		if err != nil {
			return time.Time{}, err
		}
		err = m.saveChat(chat)
		if err != nil {
			return time.Time{}, err
		}
	}

	var contact *Contact
	if chat.OneToOne() {
		contact, _ = m.allContacts.Load(request.ChatID)
	}

	var MuteTill time.Time

	switch request.MutedType {
	case MuteTill1Min:
		MuteTill = time.Now().Add(MuteFor1MinDuration)
	case MuteFor15Min:
		MuteTill = time.Now().Add(MuteFor15MinsDuration)
	case MuteFor1Hr:
		MuteTill = time.Now().Add(MuteFor1HrsDuration)
	case MuteFor8Hr:
		MuteTill = time.Now().Add(MuteFor8HrsDuration)
	case MuteFor1Week:
		MuteTill = time.Now().Add(MuteFor1WeekDuration)
	default:
		MuteTill = time.Time{}
	}
	err := m.saveChat(chat)
	if err != nil {
		return time.Time{}, err
	}

	muteTillTimeRemoveMs, err := time.Parse(time.RFC3339, MuteTill.Format(time.RFC3339))

	if err != nil {
		return time.Time{}, err
	}

	return m.muteChat(chat, contact, muteTillTimeRemoveMs)
}

func (m *Messenger) MuteChatV2(muteParams *requests.MuteChat) (time.Time, error) {
	return m.MuteChat(muteParams)
}

func (m *Messenger) muteChat(chat *Chat, contact *Contact, mutedTill time.Time) (time.Time, error) {
	err := m.persistence.MuteChat(chat.ID, mutedTill)
	if err != nil {
		return time.Time{}, err
	}

	chat.Muted = true
	chat.MuteTill = mutedTill
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chat.ID, chat)

	if contact != nil {
		err := m.syncContact(context.Background(), contact, m.dispatchMessage)
		if err != nil {
			return time.Time{}, err
		}
	}

	if !chat.MuteTill.IsZero() {
		err := m.reregisterForPushNotifications()
		if err != nil {
			return time.Time{}, err
		}
		return mutedTill, nil
	}

	return time.Time{}, m.reregisterForPushNotifications()
}

// UnmuteChat signals to the messenger that we want to be notified
// on new messages from this chat
func (m *Messenger) UnmuteChat(chatID string) error {
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return errors.New("chat not found")
	}

	var contact *Contact
	if chat.OneToOne() {
		contact, _ = m.allContacts.Load(chatID)
	}

	return m.unmuteChat(chat, contact)
}

func (m *Messenger) unmuteChat(chat *Chat, contact *Contact) error {
	err := m.persistence.UnmuteChat(chat.ID)
	if err != nil {
		return err
	}

	chat.Muted = false
	chat.MuteTill = time.Time{}
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chat.ID, chat)

	if chat.CommunityChat() {
		community, err := m.communitiesManager.GetByIDString(chat.CommunityID)
		if err != nil {
			return err
		}

		err = m.communitiesManager.SetMuted(community.ID(), false)
		if err != nil {
			return err
		}
	}

	if contact != nil {
		err := m.syncContact(context.Background(), contact, m.dispatchMessage)
		if err != nil {
			return err
		}
	}
	return m.reregisterForPushNotifications()
}

func (m *Messenger) UpdateMessageOutgoingStatus(id, newOutgoingStatus string) error {
	return m.persistence.UpdateMessageOutgoingStatus(id, newOutgoingStatus)
}

// Identicon returns an identicon based on the input string
func Identicon(id string) (string, error) {
	return identicon.GenerateBase64(id)
}

// GenerateAlias name returns the generated name given a public key hex encoded prefixed with 0x
func GenerateAlias(id string) (string, error) {
	return alias.GenerateFromPublicKeyString(id)
}

func (m *Messenger) RequestTransaction(ctx context.Context, chatID, value, contract, address string) (*MessengerResponse, error) {
	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	message := &common.Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.Seen = true
	message.Text = "Request transaction"

	request := &protobuf.RequestTransaction{
		Clock:    message.Clock,
		Address:  address,
		Value:    value,
		Contract: contract,
		ChatId:   chatID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_TRANSACTION,
		ResendAutomatically: true,
	})

	message.CommandParameters = &common.CommandParameters{
		ID:           rawMessage.ID,
		Value:        value,
		Address:      address,
		Contract:     contract,
		CommandState: common.CommandStateRequestTransaction,
	}

	if err != nil {
		return nil, err
	}
	messageID := rawMessage.ID

	message.ID = messageID
	message.CommandParameters.ID = messageID
	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	return m.addMessagesAndChat(chat, []*common.Message{message}, &response)
}

func (m *Messenger) RequestAddressForTransaction(ctx context.Context, chatID, from, value, contract string) (*MessengerResponse, error) {
	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	message := &common.Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.Seen = true
	message.Text = "Request address for transaction"

	request := &protobuf.RequestAddressForTransaction{
		Clock:    message.Clock,
		Value:    value,
		Contract: contract,
		ChatId:   chatID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	message.CommandParameters = &common.CommandParameters{
		ID:           rawMessage.ID,
		From:         from,
		Value:        value,
		Contract:     contract,
		CommandState: common.CommandStateRequestAddressForTransaction,
	}

	if err != nil {
		return nil, err
	}
	messageID := rawMessage.ID

	message.ID = messageID
	message.CommandParameters.ID = messageID
	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	return m.addMessagesAndChat(chat, []*common.Message{message}, &response)
}

func (m *Messenger) AcceptRequestAddressForTransaction(ctx context.Context, messageID, address string) (*MessengerResponse, error) {
	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = "Request address for transaction accepted"
	message.Seen = true
	message.OutgoingStatus = common.OutgoingStatusSending

	// Hide previous message
	previousMessage, err := m.persistence.MessageByCommandID(chatID, messageID)
	if err != nil {
		return nil, err
	}

	if previousMessage == nil {
		return nil, errors.New("No previous message found")
	}

	err = m.persistence.HideMessage(previousMessage.ID)
	if err != nil {
		return nil, err
	}

	message.Replace = previousMessage.ID

	request := &protobuf.AcceptRequestAddressForTransaction{
		Clock:   message.Clock,
		Id:      messageID,
		Address: address,
		ChatId:  chatID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_ACCEPT_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.Address = address
	message.CommandParameters.CommandState = common.CommandStateRequestAddressForTransactionAccepted

	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	return m.addMessagesAndChat(chat, []*common.Message{message}, &response)
}

func (m *Messenger) DeclineRequestTransaction(ctx context.Context, messageID string) (*MessengerResponse, error) {
	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = "Transaction request declined"
	message.Seen = true
	message.OutgoingStatus = common.OutgoingStatusSending
	message.Replace = messageID

	err = m.persistence.HideMessage(messageID)
	if err != nil {
		return nil, err
	}

	request := &protobuf.DeclineRequestTransaction{
		Clock:  message.Clock,
		Id:     messageID,
		ChatId: chatID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.CommandState = common.CommandStateRequestTransactionDeclined

	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	return m.addMessagesAndChat(chat, []*common.Message{message}, &response)
}

func (m *Messenger) DeclineRequestAddressForTransaction(ctx context.Context, messageID string) (*MessengerResponse, error) {
	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = "Request address for transaction declined"
	message.Seen = true
	message.OutgoingStatus = common.OutgoingStatusSending
	message.Replace = messageID

	err = m.persistence.HideMessage(messageID)
	if err != nil {
		return nil, err
	}

	request := &protobuf.DeclineRequestAddressForTransaction{
		Clock:  message.Clock,
		Id:     messageID,
		ChatId: chatID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.CommandState = common.CommandStateRequestAddressForTransactionDeclined

	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	return m.addMessagesAndChat(chat, []*common.Message{message}, &response)
}

func (m *Messenger) AcceptRequestTransaction(ctx context.Context, transactionHash, messageID string, signature []byte) (*MessengerResponse, error) {
	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Seen = true
	message.Text = transactionSentTxt
	message.OutgoingStatus = common.OutgoingStatusSending

	// Hide previous message
	previousMessage, err := m.persistence.MessageByCommandID(chatID, messageID)
	if err != nil && err != common.ErrRecordNotFound {
		return nil, err
	}

	if previousMessage != nil {
		err = m.persistence.HideMessage(previousMessage.ID)
		if err != nil {
			return nil, err
		}
		message.Replace = previousMessage.ID
	}

	err = m.persistence.HideMessage(messageID)
	if err != nil {
		return nil, err
	}

	request := &protobuf.SendTransaction{
		Clock:           message.Clock,
		Id:              messageID,
		TransactionHash: transactionHash,
		Signature:       signature,
		ChatId:          chatID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SEND_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.TransactionHash = transactionHash
	message.CommandParameters.Signature = signature
	message.CommandParameters.CommandState = common.CommandStateTransactionSent

	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	return m.addMessagesAndChat(chat, []*common.Message{message}, &response)
}

func (m *Messenger) SendTransaction(ctx context.Context, chatID, value, contract, transactionHash string, signature []byte) (*MessengerResponse, error) {
	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	message := &common.Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.LocalChatID = chatID

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Seen = true
	message.Timestamp = timestamp
	message.Text = transactionSentTxt

	request := &protobuf.SendTransaction{
		Clock:           message.Clock,
		TransactionHash: transactionHash,
		Signature:       signature,
		ChatId:          chatID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SEND_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters = &common.CommandParameters{
		TransactionHash: transactionHash,
		Value:           value,
		Contract:        contract,
		Signature:       signature,
		CommandState:    common.CommandStateTransactionSent,
	}

	err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	return m.addMessagesAndChat(chat, []*common.Message{message}, &response)
}

func (m *Messenger) ValidateTransactions(ctx context.Context, addresses []types.Address) (*MessengerResponse, error) {
	if m.verifyTransactionClient == nil {
		return nil, nil
	}

	logger := m.logger.With(zap.String("site", "ValidateTransactions"))
	logger.Debug("Validating transactions")
	txs, err := m.persistence.TransactionsToValidate()
	if err != nil {
		logger.Error("Error pulling", zap.Error(err))
		return nil, err
	}
	logger.Debug("Txs", zap.Int("count", len(txs)), zap.Any("txs", txs))
	var response MessengerResponse
	validator := NewTransactionValidator(addresses, m.persistence, m.verifyTransactionClient, m.logger)
	responses, err := validator.ValidateTransactions(ctx)
	if err != nil {
		logger.Error("Error validating", zap.Error(err))
		return nil, err
	}
	for _, validationResult := range responses {
		var message *common.Message
		chatID := contactIDFromPublicKey(validationResult.Transaction.From)
		chat, ok := m.allChats.Load(chatID)
		if !ok {
			chat = OneToOneFromPublicKey(validationResult.Transaction.From, m.transport)
		}
		if validationResult.Message != nil {
			message = validationResult.Message
		} else {
			message = &common.Message{}
			err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
			if err != nil {
				return nil, err
			}
		}

		message.MessageType = protobuf.MessageType_ONE_TO_ONE
		message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
		message.LocalChatID = chatID
		message.OutgoingStatus = ""

		clock, timestamp := chat.NextClockAndTimestamp(m.transport)
		message.Clock = clock
		message.Timestamp = timestamp
		message.WhisperTimestamp = timestamp
		message.Text = "Transaction received"
		message.Seen = false

		message.ID = validationResult.Transaction.MessageID
		if message.CommandParameters == nil {
			message.CommandParameters = &common.CommandParameters{}
		} else {
			message.CommandParameters = validationResult.Message.CommandParameters
		}

		message.CommandParameters.Value = validationResult.Value
		message.CommandParameters.Contract = validationResult.Contract
		message.CommandParameters.Address = validationResult.Address
		message.CommandParameters.CommandState = common.CommandStateTransactionSent
		message.CommandParameters.TransactionHash = validationResult.Transaction.TransactionHash

		err = message.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
		if err != nil {
			return nil, err
		}

		err = chat.UpdateFromMessage(message, m.transport)
		if err != nil {
			return nil, err
		}

		if len(message.CommandParameters.ID) != 0 {
			// Hide previous message
			previousMessage, err := m.persistence.MessageByCommandID(chatID, message.CommandParameters.ID)
			if err != nil && err != common.ErrRecordNotFound {
				return nil, err
			}

			if previousMessage != nil {
				err = m.persistence.HideMessage(previousMessage.ID)
				if err != nil {
					return nil, err
				}
				message.Replace = previousMessage.ID
			}
		}

		response.AddMessage(message)
		m.allChats.Store(chat.ID, chat)
		response.AddChat(chat)

		contact, err := m.getOrBuildContactFromMessage(message)
		if err != nil {
			return nil, err
		}

		notificationsEnabled, err := m.settings.GetNotificationsEnabled()
		if err != nil {
			return nil, err
		}

		profilePicturesVisibility, err := m.settings.GetProfilePicturesVisibility()
		if err != nil {
			return nil, err
		}

		if notificationsEnabled {
			notification, err := NewMessageNotification(message.ID, message, chat, contact, m.allContacts, profilePicturesVisibility)
			if err != nil {
				return nil, err
			}
			response.AddNotification(notification)
		}

	}

	if len(response.messages) > 0 {
		err = m.SaveMessages(response.Messages())
		if err != nil {
			return nil, err
		}
	}
	return &response, nil
}

// pullMessagesAndResponsesFromDB pulls all the messages and the one that have
// been replied to from the database
func (m *Messenger) pullMessagesAndResponsesFromDB(messages []*common.Message) ([]*common.Message, error) {
	var messageIDs []string
	for _, message := range messages {
		messageIDs = append(messageIDs, message.ID)
		if len(message.ResponseTo) != 0 {
			messageIDs = append(messageIDs, message.ResponseTo)
		}

	}
	// We pull from the database all the messages & replies involved,
	// so we let the db build the correct messages
	return m.persistence.MessagesByIDs(messageIDs)
}

func (m *Messenger) SignMessage(message string) ([]byte, error) {
	hash := crypto.TextHash([]byte(message))
	return crypto.Sign(hash, m.identity)
}

func (m *Messenger) getTimesource() common.TimeSource {
	return m.transport
}

func (m *Messenger) getCurrentTimeInMillis() uint64 {
	return m.getTimesource().GetCurrentTime()
}

// AddPushNotificationsServer adds a push notification server
func (m *Messenger) AddPushNotificationsServer(ctx context.Context, publicKey *ecdsa.PublicKey, serverType pushnotificationclient.ServerType) error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	return m.pushNotificationClient.AddPushNotificationsServer(publicKey, serverType)
}

// RemovePushNotificationServer removes a push notification server
func (m *Messenger) RemovePushNotificationServer(ctx context.Context, publicKey *ecdsa.PublicKey) error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	return m.pushNotificationClient.RemovePushNotificationServer(publicKey)
}

// UnregisterFromPushNotifications unregister from any server
func (m *Messenger) UnregisterFromPushNotifications(ctx context.Context) error {
	return m.pushNotificationClient.Unregister()
}

// DisableSendingPushNotifications signals the client not to send any push notification
func (m *Messenger) DisableSendingPushNotifications() error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	m.pushNotificationClient.DisableSending()
	return nil
}

// EnableSendingPushNotifications signals the client to send push notifications
func (m *Messenger) EnableSendingPushNotifications() error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	m.pushNotificationClient.EnableSending()
	return nil
}

func (m *Messenger) pushNotificationOptions() *pushnotificationclient.RegistrationOptions {
	var contactIDs []*ecdsa.PublicKey
	var mutedChatIDs []string
	var publicChatIDs []string
	var blockedChatIDs []string

	m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		if contact.added() && !contact.Blocked {
			pk, err := contact.PublicKey()
			if err != nil {
				m.logger.Warn("could not parse contact public key")
				return true
			}
			contactIDs = append(contactIDs, pk)
		} else if contact.Blocked {
			blockedChatIDs = append(blockedChatIDs, contact.ID)
		}
		return true
	})

	m.allChats.Range(func(chatID string, chat *Chat) (shouldContinue bool) {
		if chat.Muted {
			mutedChatIDs = append(mutedChatIDs, chat.ID)
			return true
		}
		if chat.Active && (chat.Public() || chat.CommunityChat()) {
			publicChatIDs = append(publicChatIDs, chat.ID)
		}
		return true
	})

	return &pushnotificationclient.RegistrationOptions{
		ContactIDs:     contactIDs,
		MutedChatIDs:   mutedChatIDs,
		PublicChatIDs:  publicChatIDs,
		BlockedChatIDs: blockedChatIDs,
	}
}

// RegisterForPushNotification register deviceToken with any push notification server enabled
func (m *Messenger) RegisterForPushNotifications(ctx context.Context, deviceToken, apnTopic string, tokenType protobuf.PushNotificationRegistration_TokenType) error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.pushNotificationClient.Register(deviceToken, apnTopic, tokenType, m.pushNotificationOptions())
	if err != nil {
		m.logger.Error("failed to register for push notifications", zap.Error(err))
		return err
	}
	return nil
}

// RegisteredForPushNotifications returns whether we successfully registered with all the servers
func (m *Messenger) RegisteredForPushNotifications() (bool, error) {
	if m.pushNotificationClient == nil {
		return false, errors.New("no push notification client")
	}
	return m.pushNotificationClient.Registered()
}

// EnablePushNotificationsFromContactsOnly is used to indicate that we want to received push notifications only from contacts
func (m *Messenger) EnablePushNotificationsFromContactsOnly() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.EnablePushNotificationsFromContactsOnly(m.pushNotificationOptions())
}

// DisablePushNotificationsFromContactsOnly is used to indicate that we want to received push notifications from anyone
func (m *Messenger) DisablePushNotificationsFromContactsOnly() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.DisablePushNotificationsFromContactsOnly(m.pushNotificationOptions())
}

// EnablePushNotificationsBlockMentions is used to indicate that we dont want to received push notifications for mentions
func (m *Messenger) EnablePushNotificationsBlockMentions() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.EnablePushNotificationsBlockMentions(m.pushNotificationOptions())
}

// DisablePushNotificationsBlockMentions is used to indicate that we want to received push notifications for mentions
func (m *Messenger) DisablePushNotificationsBlockMentions() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.DisablePushNotificationsBlockMentions(m.pushNotificationOptions())
}

// GetPushNotificationsServers returns the servers used for push notifications
func (m *Messenger) GetPushNotificationsServers() ([]*pushnotificationclient.PushNotificationServer, error) {
	if m.pushNotificationClient == nil {
		return nil, errors.New("no push notification client")
	}
	return m.pushNotificationClient.GetServers()
}

// StartPushNotificationsServer initialize and start a push notification server, using the current messenger identity key
func (m *Messenger) StartPushNotificationsServer() error {
	if m.pushNotificationServer == nil {
		pushNotificationServerPersistence := pushnotificationserver.NewSQLitePersistence(m.database)
		config := &pushnotificationserver.Config{
			Enabled:  true,
			Logger:   m.logger,
			Identity: m.identity,
		}
		m.pushNotificationServer = pushnotificationserver.New(config, pushNotificationServerPersistence, m.sender)
	}

	return m.pushNotificationServer.Start()
}

// StopPushNotificationServer stops the push notification server if running
func (m *Messenger) StopPushNotificationsServer() error {
	m.pushNotificationServer = nil
	return nil
}

func generateAliasAndIdenticon(pk string) (string, string, error) {
	identicon, err := identicon.GenerateBase64(pk)
	if err != nil {
		return "", "", err
	}

	name, err := alias.GenerateFromPublicKeyString(pk)
	if err != nil {
		return "", "", err
	}
	return name, identicon, nil

}

func (m *Messenger) UnfurlURLs(urls []string) ([]common.LinkPreview, error) {
	return linkpreview.UnfurlURLs(m.logger, linkpreview.NewDefaultHTTPClient(), urls)
}

func (m *Messenger) SendEmojiReaction(ctx context.Context, chatID, messageID string, emojiID protobuf.EmojiReaction_Type) (*MessengerResponse, error) {
	var response MessengerResponse

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFound
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	emojiR := &EmojiReaction{
		EmojiReaction: protobuf.EmojiReaction{
			Clock:     clock,
			MessageId: messageID,
			ChatId:    chatID,
			Type:      emojiID,
		},
		LocalChatID: chatID,
		From:        types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
	}
	encodedMessage, err := m.encodeChatEntity(chat, emojiR)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:          chatID,
		Payload:              encodedMessage,
		SkipGroupMessageWrap: true,
		MessageType:          protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		// Don't resend using datasync, that would create quite a lot
		// of traffic if clicking too eagelry
		ResendAutomatically: false,
	})
	if err != nil {
		return nil, err
	}

	response.AddEmojiReaction(emojiR)
	response.AddChat(chat)

	err = m.persistence.SaveEmojiReaction(emojiR)
	if err != nil {
		return nil, errors.Wrap(err, "Can't save emoji reaction in db")
	}

	return &response, nil
}

func (m *Messenger) EmojiReactionsByChatID(chatID string, cursor string, limit int) ([]*EmojiReaction, error) {
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, err
	}

	if chat.Timeline() {
		var chatIDs = []string{"@" + contactIDFromPublicKey(&m.identity.PublicKey)}
		m.allContacts.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
			if contact.added() {
				chatIDs = append(chatIDs, "@"+contact.ID)
			}
			return true
		})
		return m.persistence.EmojiReactionsByChatIDs(chatIDs, cursor, limit)
	}
	return m.persistence.EmojiReactionsByChatID(chatID, cursor, limit)
}

func (m *Messenger) EmojiReactionsByChatIDMessageID(chatID string, messageID string) ([]*EmojiReaction, error) {
	_, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, err
	}

	return m.persistence.EmojiReactionsByChatIDMessageID(chatID, messageID)
}

func (m *Messenger) SendEmojiReactionRetraction(ctx context.Context, emojiReactionID string) (*MessengerResponse, error) {
	emojiR, err := m.persistence.EmojiReactionByID(emojiReactionID)
	if err != nil {
		return nil, err
	}

	// Check that the sender is the key owner
	pk := types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))
	if emojiR.From != pk {
		return nil, errors.Errorf("identity mismatch, "+
			"emoji reactions can only be retracted by the reaction sender, "+
			"emoji reaction sent by '%s', current identity '%s'",
			emojiR.From, pk,
		)
	}

	// Get chat and clock
	chat, ok := m.allChats.Load(emojiR.GetChatId())
	if !ok {
		return nil, ErrChatNotFound
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	// Update the relevant fields
	emojiR.Clock = clock
	emojiR.Retracted = true

	encodedMessage, err := m.encodeChatEntity(chat, emojiR)
	if err != nil {
		return nil, err
	}

	// Send the marshalled EmojiReactionRetraction protobuf
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:          emojiR.GetChatId(),
		Payload:              encodedMessage,
		SkipGroupMessageWrap: true,
		MessageType:          protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		// Don't resend using datasync, that would create quite a lot
		// of traffic if clicking too eagelry
		ResendAutomatically: false,
	})
	if err != nil {
		return nil, err
	}

	// Update MessengerResponse
	response := MessengerResponse{}
	emojiR.Retracted = true
	response.AddEmojiReaction(emojiR)
	response.AddChat(chat)

	// Persist retraction state for emoji reaction
	err = m.persistence.SaveEmojiReaction(emojiR)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (m *Messenger) encodeChatEntity(chat *Chat, message common.ChatEntity) ([]byte, error) {
	var encodedMessage []byte
	var err error
	l := m.logger.With(zap.String("site", "Send"), zap.String("chatID", chat.ID))

	switch chat.ChatType {
	case ChatTypeOneToOne:
		l.Debug("sending private message")
		message.SetMessageType(protobuf.MessageType_ONE_TO_ONE)
		encodedMessage, err = proto.Marshal(message.GetProtobuf())
		if err != nil {
			return nil, err
		}

	case ChatTypePublic, ChatTypeProfile:
		l.Debug("sending public message", zap.String("chatName", chat.Name))
		message.SetMessageType(protobuf.MessageType_PUBLIC_GROUP)
		encodedMessage, err = proto.Marshal(message.GetProtobuf())
		if err != nil {
			return nil, err
		}

	case ChatTypeCommunityChat:
		l.Debug("sending community chat message", zap.String("chatName", chat.Name))
		// TODO: add grant
		message.SetMessageType(protobuf.MessageType_COMMUNITY_CHAT)
		encodedMessage, err = proto.Marshal(message.GetProtobuf())
		if err != nil {
			return nil, err
		}

	case ChatTypePrivateGroupChat:
		message.SetMessageType(protobuf.MessageType_PRIVATE_GROUP)
		l.Debug("sending group message", zap.String("chatName", chat.Name))
		if !message.WrapGroupMessage() {
			encodedMessage, err = proto.Marshal(message.GetProtobuf())
			if err != nil {
				return nil, err
			}
		} else {

			group, err := newProtocolGroupFromChat(chat)
			if err != nil {
				return nil, err
			}

			// NOTE(cammellos): Disabling for now since the optimiziation is not
			// applicable anymore after we changed group rules to allow
			// anyone to change group details
			encodedMessage, err = m.sender.EncodeMembershipUpdate(group, message)
			if err != nil {
				return nil, err
			}
		}

	default:
		return nil, errors.New("chat type not supported")
	}

	return encodedMessage, nil
}

func (m *Messenger) getOrBuildContactFromMessage(msg *common.Message) (*Contact, error) {
	if c, ok := m.allContacts.Load(msg.From); ok {
		return c, nil
	}

	senderPubKey, err := msg.GetSenderPubKey()
	if err != nil {
		return nil, err
	}
	senderID := contactIDFromPublicKey(senderPubKey)
	c, err := buildContact(senderID, senderPubKey)
	if err != nil {
		return nil, err
	}

	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allContacts.Store(msg.From, c)
	return c, nil
}

func (m *Messenger) BloomFilter() []byte {
	return m.transport.BloomFilter()
}

func (m *Messenger) getSettings() (settings.Settings, error) {
	sDB, err := accounts.NewDB(m.database)
	if err != nil {
		return settings.Settings{}, err
	}
	return sDB.GetSettings()
}

func (m *Messenger) getEnsUsernameDetails() (result []*ensservice.UsernameDetail, err error) {
	db := ensservice.NewEnsDatabase(m.database)
	return db.GetEnsUsernames(nil)
}

func (m *Messenger) handleSyncBookmark(state *ReceivedMessageState, message protobuf.SyncBookmark) error {
	bookmark := &browsers.Bookmark{
		URL:      message.Url,
		Name:     message.Name,
		ImageURL: message.ImageUrl,
		Removed:  message.Removed,
		Clock:    message.Clock,
	}
	state.AllBookmarks[message.Url] = bookmark
	return nil
}

func (m *Messenger) handleSyncClearHistory(state *ReceivedMessageState, message protobuf.SyncClearHistory) error {
	chatID := message.ChatId
	existingChat, ok := state.AllChats.Load(chatID)
	if !ok {
		return ErrChatNotFound
	}

	if existingChat.DeletedAtClockValue >= message.ClearedAt {
		return nil
	}

	err := m.persistence.ClearHistoryFromSyncMessage(existingChat, message.ClearedAt)
	if err != nil {
		return err
	}

	if existingChat.Public() {
		err = m.transport.ClearProcessedMessageIDsCache()
		if err != nil {
			return err
		}
	}

	state.AllChats.Store(chatID, existingChat)
	state.Response.AddChat(existingChat)
	state.Response.AddClearedHistory(&ClearedHistory{
		ClearedAt: message.ClearedAt,
		ChatID:    chatID,
	})
	return nil
}

func (m *Messenger) handleSyncTrustedUser(state *ReceivedMessageState, message protobuf.SyncTrustedUser) error {
	updated, err := m.verificationDatabase.UpsertTrustStatus(message.Id, verification.TrustStatus(message.Status), message.Clock)
	if err != nil {
		return err
	}

	if updated {
		state.AllTrustStatus[message.Id] = verification.TrustStatus(message.Status)

		contact, ok := m.allContacts.Load(message.Id)
		if !ok {
			m.logger.Info("contact not found")
			return nil
		}

		contact.TrustStatus = verification.TrustStatus(message.Status)
		m.allContacts.Store(contact.ID, contact)
		state.ModifiedContacts.Store(contact.ID, true)
	}

	return nil
}

func ToVerificationRequest(message protobuf.SyncVerificationRequest) *verification.Request {
	return &verification.Request{
		From:          message.From,
		To:            message.To,
		Challenge:     message.Challenge,
		Response:      message.Response,
		RequestedAt:   message.RequestedAt,
		RepliedAt:     message.RepliedAt,
		RequestStatus: verification.RequestStatus(message.VerificationStatus),
	}
}

func (m *Messenger) handleSyncVerificationRequest(state *ReceivedMessageState, message protobuf.SyncVerificationRequest) error {
	verificationRequest := ToVerificationRequest(message)

	err := m.verificationDatabase.SaveVerificationRequest(verificationRequest)
	if err != nil {
		return err
	}

	myPubKey := hexutil.Encode(crypto.FromECDSAPub(&m.identity.PublicKey))

	state.AllVerificationRequests = append(state.AllVerificationRequests, verificationRequest)

	if message.From == myPubKey { // Verification requests we sent
		contact, ok := m.allContacts.Load(message.To)
		if !ok {
			m.logger.Info("contact not found")
			return nil
		}

		contact.VerificationStatus = VerificationStatus(message.VerificationStatus)
		if err := m.persistence.SaveContact(contact, nil); err != nil {
			return err
		}

		m.allContacts.Store(contact.ID, contact)
		state.ModifiedContacts.Store(contact.ID, true)

		// TODO: create activity center notif

	}
	// else { // Verification requests we received
	// // TODO: activity center notif
	//}

	return nil
}

func (m *Messenger) ImageServerURL() string {
	return m.httpServer.MakeImageServerURL()
}

func (m *Messenger) myHexIdentity() string {
	return common.PubkeyToHex(&m.identity.PublicKey)
}

func (m *Messenger) GetMentionsManager() *MentionManager {
	return m.mentionsManager
}

func (m *Messenger) getConnectedMessages(message *common.Message, chatID string) ([]*common.Message, error) {
	var connectedMessages []*common.Message
	// In case of Image messages, we need to delete all the images in the album
	if message.ContentType == protobuf.ChatMessage_IMAGE {
		image := message.GetImage()
		if image != nil && image.AlbumId != "" {
			messagesInTheAlbum, err := m.persistence.albumMessages(chatID, image.GetAlbumId())
			if err != nil {
				return nil, err
			}
			connectedMessages = append(connectedMessages, messagesInTheAlbum...)
			return connectedMessages, nil
		}
	}
	return append(connectedMessages, message), nil
}

func (m *Messenger) withChatClock(callback func(string, uint64) error) error {
	clock, chat := m.getLastClockWithRelatedChat()
	err := callback(chat.ID, clock)
	if err != nil {
		return err
	}
	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncDeleteForMeMessage(ctx context.Context, rawMessageDispatcher RawMessageHandler) error {
	deleteForMes, err := m.persistence.GetDeleteForMeMessages()
	if err != nil {
		return err
	}

	return m.withChatClock(func(chatID string, _ uint64) error {
		for _, deleteForMe := range deleteForMes {
			encodedMessage, err2 := proto.Marshal(deleteForMe)
			if err2 != nil {
				return err2
			}
			rawMessage := common.RawMessage{
				LocalChatID:         chatID,
				Payload:             encodedMessage,
				MessageType:         protobuf.ApplicationMetadataMessage_SYNC_DELETE_FOR_ME_MESSAGE,
				ResendAutomatically: true,
			}
			_, err2 = rawMessageDispatcher(ctx, rawMessage)
			if err2 != nil {
				return err2
			}
		}
		return nil
	})
}

func (m *Messenger) syncSocialLinks(ctx context.Context, rawMessageDispatcher RawMessageHandler) error {
	if !m.hasPairedDevices() {
		return nil
	}

	dbSocialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return err
	}

	dbClock, err := m.settings.GetSocialLinksClock()
	if err != nil {
		return err
	}

	_, chat := m.getLastClockWithRelatedChat()
	encodedMessage, err := proto.Marshal(dbSocialLinks.ToSyncProtobuf(dbClock))
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_SOCIAL_LINKS,
		ResendAutomatically: true,
	}

	_, err = rawMessageDispatcher(ctx, rawMessage)
	return err
}

func (m *Messenger) HandleSyncSocialLinks(state *ReceivedMessageState, message protobuf.SyncSocialLinks) error {
	return m.handleSyncSocialLinks(&message, func(links identity.SocialLinks) {
		state.Response.SocialLinksInfo = &identity.SocialLinksInfo{
			Links:   links,
			Removed: len(links) == 0,
		}
	})
}

func (m *Messenger) handleSyncSocialLinks(message *protobuf.SyncSocialLinks, callback func(identity.SocialLinks)) error {
	if message == nil {
		return nil
	}
	var (
		links identity.SocialLinks
		err   error
	)
	for _, sl := range message.SocialLinks {
		link := &identity.SocialLink{
			Text: sl.Text,
			URL:  sl.Url,
		}
		err = ValidateSocialLink(link)
		if err != nil {
			return err
		}

		links = append(links, link)
	}

	err = m.settings.AddOrReplaceSocialLinksIfNewer(links, message.Clock)
	if err != nil {
		if err == sociallinkssettings.ErrOlderSocialLinksProvided {
			return nil
		}
		return err
	}

	callback(links)

	return nil
}

func (m *Messenger) GetDeleteForMeMessages() ([]*protobuf.DeleteForMeMessage, error) {
	return m.persistence.GetDeleteForMeMessages()
}
