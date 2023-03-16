package ext

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"go.uber.org/zap"

	commongethtypes "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/db"
	coretypes "github.com/status-im/status-go/eth-node/core/types"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/anonmetrics"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/ext/mailservers"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/wallet/transfer"
)

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(types.Hash, types.Hash, []byte, error)
	MailServerRequestExpired(types.Hash)
}

// Service is a service that provides some additional API to whisper-based protocols like Whisper or Waku.
type Service struct {
	messenger       *protocol.Messenger
	identity        *ecdsa.PrivateKey
	cancelMessenger chan struct{}
	storage         db.TransactionalStorage
	n               types.Node
	rpcClient       *rpc.Client
	config          params.NodeConfig
	mailMonitor     *MailRequestMonitor
	server          *p2p.Server
	peerStore       *mailservers.PeerStore
	accountsDB      *accounts.Database
	multiAccountsDB *multiaccounts.Database
	account         *multiaccounts.Account
}

// Make sure that Service implements node.Service interface.
var _ node.Lifecycle = (*Service)(nil)

func New(
	config params.NodeConfig,
	n types.Node,
	rpcClient *rpc.Client,
	ldb *leveldb.DB,
	mailMonitor *MailRequestMonitor,
	eventSub mailservers.EnvelopeEventSubscriber,
) *Service {
	cache := mailservers.NewCache(ldb)
	peerStore := mailservers.NewPeerStore(cache)
	return &Service{
		storage:     db.NewLevelDBStorage(ldb),
		n:           n,
		rpcClient:   rpcClient,
		config:      config,
		mailMonitor: mailMonitor,
		peerStore:   peerStore,
	}
}

func (s *Service) NodeID() *ecdsa.PrivateKey {
	if s.server == nil {
		return nil
	}
	return s.server.PrivateKey
}

func (s *Service) GetPeer(rawURL string) (*enode.Node, error) {
	if len(rawURL) == 0 {
		return mailservers.GetFirstConnected(s.server, s.peerStore)
	}
	return enode.ParseV4(rawURL)
}

func (s *Service) InitProtocol(nodeName string, identity *ecdsa.PrivateKey, db *sql.DB, httpServer *server.MediaServer, multiAccountDb *multiaccounts.Database, acc *multiaccounts.Account, accountManager *account.GethManager, rpcClient *rpc.Client, logger *zap.Logger) error {
	var err error
	if !s.config.ShhextConfig.PFSEnabled {
		return nil
	}

	// If Messenger has been already set up, we need to shut it down
	// before we init it again. Otherwise, it will lead to goroutines leakage
	// due to not stopped filters.
	if s.messenger != nil {
		if err := s.messenger.Shutdown(); err != nil {
			return err
		}
	}

	s.identity = identity

	dataDir := filepath.Clean(s.config.ShhextConfig.BackupDisabledDataDir)

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return err
	}

	envelopesMonitorConfig := &transport.EnvelopesMonitorConfig{
		MaxAttempts:                      s.config.ShhextConfig.MaxMessageDeliveryAttempts,
		AwaitOnlyMailServerConfirmations: s.config.ShhextConfig.MailServerConfirmations,
		IsMailserver: func(peer types.EnodeID) bool {
			return s.peerStore.Exist(peer)
		},
		EnvelopeEventsHandler: EnvelopeSignalHandler{},
		Logger:                logger,
	}
	s.accountsDB, err = accounts.NewDB(db)
	if err != nil {
		return err
	}
	s.multiAccountsDB = multiAccountDb
	s.account = acc

	options, err := buildMessengerOptions(s.config, identity, db, httpServer, s.rpcClient, s.multiAccountsDB, acc, envelopesMonitorConfig, s.accountsDB, logger, &MessengerSignalsHandler{})
	if err != nil {
		return err
	}

	messenger, err := protocol.NewMessenger(
		nodeName,
		identity,
		s.n,
		s.config.ShhextConfig.InstallationID,
		s.peerStore,
		accountManager,
		rpcClient,
		options...,
	)
	if err != nil {
		return err
	}
	s.messenger = messenger
	s.messenger.SetP2PServer(s.server)
	return messenger.Init()
}

func (s *Service) StartMessenger() (*protocol.MessengerResponse, error) {
	// Start a loop that retrieves all messages and propagates them to status-mobile.
	s.cancelMessenger = make(chan struct{})
	response, err := s.messenger.Start()
	if err != nil {
		return nil, err
	}
	go s.retrieveMessagesLoop(time.Second, s.cancelMessenger)
	go s.verifyTransactionLoop(30*time.Second, s.cancelMessenger)

	if s.config.ShhextConfig.BandwidthStatsEnabled {
		go s.retrieveStats(5*time.Second, s.cancelMessenger)
	}

	return response, nil
}

func publishMessengerResponse(response *protocol.MessengerResponse) {
	if !response.IsEmpty() {
		notifications := response.Notifications()
		// Clear notifications as not used for now
		response.ClearNotifications()
		PublisherSignalHandler{}.NewMessages(response)
		localnotifications.PushMessages(notifications)
	}
}

func (s *Service) retrieveMessagesLoop(tick time.Duration, cancel <-chan struct{}) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// We might be shutting down here
			if s.messenger == nil {
				return
			}
			response, err := s.messenger.RetrieveAll()
			if err != nil {
				log.Error("failed to retrieve raw messages", "err", err)
				continue
			}
			publishMessengerResponse(response)
		case <-cancel:
			return
		}
	}
}

func (s *Service) retrieveStats(tick time.Duration, cancel <-chan struct{}) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			response := s.messenger.GetStats()
			PublisherSignalHandler{}.Stats(response)
		case <-cancel:
			return
		}
	}
}

type verifyTransactionClient struct {
	chainID *big.Int
	url     string
}

func (c *verifyTransactionClient) TransactionByHash(ctx context.Context, hash types.Hash) (coretypes.Message, coretypes.TransactionStatus, error) {
	signer := gethtypes.NewLondonSigner(c.chainID)
	client, err := ethclient.Dial(c.url)
	if err != nil {
		return coretypes.Message{}, coretypes.TransactionStatusPending, err
	}

	transaction, pending, err := client.TransactionByHash(ctx, commongethtypes.BytesToHash(hash.Bytes()))
	if err != nil {
		return coretypes.Message{}, coretypes.TransactionStatusPending, err
	}

	message, err := transaction.AsMessage(signer, nil)
	if err != nil {
		return coretypes.Message{}, coretypes.TransactionStatusPending, err
	}
	from := types.BytesToAddress(message.From().Bytes())
	to := types.BytesToAddress(message.To().Bytes())

	if pending {
		return coretypes.NewMessage(
			from,
			&to,
			message.Nonce(),
			message.Value(),
			message.Gas(),
			message.GasPrice(),
			message.Data(),
			message.CheckNonce(),
		), coretypes.TransactionStatusPending, nil
	}

	receipt, err := client.TransactionReceipt(ctx, commongethtypes.BytesToHash(hash.Bytes()))
	if err != nil {
		return coretypes.Message{}, coretypes.TransactionStatusPending, err
	}

	coremessage := coretypes.NewMessage(
		from,
		&to,
		message.Nonce(),
		message.Value(),
		message.Gas(),
		message.GasPrice(),
		message.Data(),
		message.CheckNonce(),
	)

	// Token transfer, check the logs
	if len(coremessage.Data()) != 0 {
		if transfer.IsTokenTransfer(receipt.Logs) {
			return coremessage, coretypes.TransactionStatus(receipt.Status), nil
		}
		return coremessage, coretypes.TransactionStatusFailed, nil
	}

	return coremessage, coretypes.TransactionStatus(receipt.Status), nil
}

func (s *Service) verifyTransactionLoop(tick time.Duration, cancel <-chan struct{}) {
	if s.config.ShhextConfig.VerifyTransactionURL == "" {
		log.Warn("not starting transaction loop")
		return
	}

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	ctx, cancelVerifyTransaction := context.WithCancel(context.Background())

	for {
		select {
		case <-ticker.C:
			accounts, err := s.accountsDB.GetAccounts()
			if err != nil {
				log.Error("failed to retrieve accounts", "err", err)
			}
			var wallets []types.Address
			for _, account := range accounts {
				if account.IsOwnAccount() {
					wallets = append(wallets, types.BytesToAddress(account.Address.Bytes()))
				}
			}

			response, err := s.messenger.ValidateTransactions(ctx, wallets)
			if err != nil {
				log.Error("failed to validate transactions", "err", err)
				continue
			}
			publishMessengerResponse(response)

		case <-cancel:
			cancelVerifyTransaction()
			return
		}
	}
}

func (s *Service) EnableInstallation(installationID string) error {
	return s.messenger.EnableInstallation(installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (s *Service) DisableInstallation(installationID string) error {
	return s.messenger.DisableInstallation(installationID)
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []gethrpc.API {
	panic("this is abstract service, use shhext or wakuext implementation")
}

func (s *Service) SetP2PServer(server *p2p.Server) {
	s.server = server
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Start() error {
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	log.Info("Stopping shhext service")
	if s.cancelMessenger != nil {
		select {
		case <-s.cancelMessenger:
			// channel already closed
		default:
			close(s.cancelMessenger)
			s.cancelMessenger = nil
		}
	}

	if s.messenger != nil {
		if err := s.messenger.Shutdown(); err != nil {
			log.Error("failed to stop messenger", "err", err)
			return err
		}
		s.messenger = nil
	}

	return nil
}

func buildMessengerOptions(
	config params.NodeConfig,
	identity *ecdsa.PrivateKey,
	db *sql.DB,
	httpServer *server.MediaServer,
	rpcClient *rpc.Client,
	multiAccounts *multiaccounts.Database,
	account *multiaccounts.Account,
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig,
	accountsDB *accounts.Database,
	logger *zap.Logger,
	messengerSignalsHandler protocol.MessengerSignalsHandler,
) ([]protocol.Option, error) {
	options := []protocol.Option{
		protocol.WithCustomLogger(logger),
		protocol.WithPushNotifications(),
		protocol.WithDatabase(db),
		protocol.WithMultiAccounts(multiAccounts),
		protocol.WithMailserversDatabase(mailserversDB.NewDB(db)),
		protocol.WithAccount(account),
		protocol.WithBrowserDatabase(browsers.NewDB(db)),
		protocol.WithEnvelopesMonitorConfig(envelopesMonitorConfig),
		protocol.WithSignalsHandler(messengerSignalsHandler),
		protocol.WithENSVerificationConfig(publishMessengerResponse, config.ShhextConfig.VerifyENSURL, config.ShhextConfig.VerifyENSContractAddress),
		protocol.WithClusterConfig(config.ClusterConfig),
		protocol.WithTorrentConfig(&config.TorrentConfig),
		protocol.WithHTTPServer(httpServer),
		protocol.WithRPCClient(rpcClient),
		protocol.WithMessageCSV(config.OutputMessageCSVEnabled),
	}

	if config.ShhextConfig.DataSyncEnabled {
		options = append(options, protocol.WithDatasync())
	}

	settings, err := accountsDB.GetSettings()
	if err != sql.ErrNoRows && err != nil {
		return nil, err
	}

	// Generate anon metrics client config
	if settings.AnonMetricsShouldSend {
		keyBytes, err := hex.DecodeString(config.ShhextConfig.AnonMetricsSendID)
		if err != nil {
			return nil, err
		}

		key, err := crypto.UnmarshalPubkey(keyBytes)
		if err != nil {
			return nil, err
		}

		amcc := &anonmetrics.ClientConfig{
			ShouldSend:  true,
			SendAddress: key,
		}
		options = append(options, protocol.WithAnonMetricsClientConfig(amcc))
	}

	// Generate anon metrics server config
	if config.ShhextConfig.AnonMetricsServerEnabled {
		if len(config.ShhextConfig.AnonMetricsServerPostgresURI) == 0 {
			return nil, errors.New("AnonMetricsServerPostgresURI must be set")
		}

		amsc := &anonmetrics.ServerConfig{
			Enabled:     true,
			PostgresURI: config.ShhextConfig.AnonMetricsServerPostgresURI,
		}
		options = append(options, protocol.WithAnonMetricsServerConfig(amsc))
	}

	if settings.TelemetryServerURL != "" {
		options = append(options, protocol.WithTelemetry(settings.TelemetryServerURL))
	}

	if settings.PushNotificationsServerEnabled {
		config := &pushnotificationserver.Config{
			Enabled: true,
			Logger:  logger,
		}
		options = append(options, protocol.WithPushNotificationServerConfig(config))
	}

	var pushNotifServKey []*ecdsa.PublicKey
	for _, d := range config.ShhextConfig.DefaultPushNotificationsServers {
		pushNotifServKey = append(pushNotifServKey, d.PublicKey)
	}

	options = append(options, protocol.WithPushNotificationClientConfig(&pushnotificationclient.Config{
		DefaultServers:             pushNotifServKey,
		BlockMentions:              settings.PushNotificationsBlockMentions,
		SendEnabled:                settings.SendPushNotifications,
		AllowFromContactsOnly:      settings.PushNotificationsFromContactsOnly,
		RemoteNotificationsEnabled: settings.RemotePushNotificationsEnabled,
	}))

	if config.ShhextConfig.VerifyTransactionURL != "" {
		client := &verifyTransactionClient{
			url:     config.ShhextConfig.VerifyTransactionURL,
			chainID: big.NewInt(config.ShhextConfig.VerifyTransactionChainID),
		}
		options = append(options, protocol.WithVerifyTransactionClient(client))
	}

	return options, nil
}

func (s *Service) ConnectionChanged(state connection.State) {
	if s.messenger != nil {
		s.messenger.ConnectionChanged(state)
	}
}

func (s *Service) Messenger() *protocol.Messenger {
	return s.messenger
}
