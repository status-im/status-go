package ext

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	commongethtypes "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/db"
	coretypes "github.com/status-im/status-go/eth-node/core/types"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/services/ext/mailservers"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/wallet"

	"go.uber.org/zap"
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
	config          params.ShhextConfig
	mailMonitor     *MailRequestMonitor
	server          *p2p.Server
	peerStore       *mailservers.PeerStore
	accountsDB      *accounts.Database
	multiAccountsDB *multiaccounts.Database
	account         *multiaccounts.Account
}

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

func New(
	config params.ShhextConfig,
	n types.Node,
	ldb *leveldb.DB,
	mailMonitor *MailRequestMonitor,
	eventSub mailservers.EnvelopeEventSubscriber,
) *Service {
	cache := mailservers.NewCache(ldb)
	peerStore := mailservers.NewPeerStore(cache)
	return &Service{
		storage:     db.NewLevelDBStorage(ldb),
		n:           n,
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

func (s *Service) InitProtocol(identity *ecdsa.PrivateKey, db *sql.DB, multiAccountDb *multiaccounts.Database, acc *multiaccounts.Account, logger *zap.Logger) error {
	if !s.config.PFSEnabled {
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

	dataDir := filepath.Clean(s.config.BackupDisabledDataDir)

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return err
	}

	envelopesMonitorConfig := &transport.EnvelopesMonitorConfig{
		MaxAttempts:                    s.config.MaxMessageDeliveryAttempts,
		MailserverConfirmationsEnabled: s.config.MailServerConfirmations,
		IsMailserver: func(peer types.EnodeID) bool {
			return s.peerStore.Exist(peer)
		},
		EnvelopeEventsHandler: EnvelopeSignalHandler{},
		Logger:                logger,
	}
	s.accountsDB = accounts.NewDB(db)
	s.multiAccountsDB = multiAccountDb
	s.account = acc

	options, err := buildMessengerOptions(s.config, identity, db, s.multiAccountsDB, acc, envelopesMonitorConfig, s.accountsDB, logger, &MessengerSignalsHandler{})
	if err != nil {
		return err
	}

	messenger, err := protocol.NewMessenger(
		identity,
		s.n,
		s.config.InstallationID,
		options...,
	)
	if err != nil {
		return err
	}
	s.messenger = messenger
	return messenger.Init()
}

func (s *Service) StartMessenger() (*protocol.MessengerResponse, error) {
	// Start a loop that retrieves all messages and propagates them to status-react.
	s.cancelMessenger = make(chan struct{})
	response, err := s.messenger.Start()
	if err != nil {
		return nil, err
	}
	go s.retrieveMessagesLoop(time.Second, s.cancelMessenger)
	go s.verifyTransactionLoop(30*time.Second, s.cancelMessenger)
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

type verifyTransactionClient struct {
	chainID *big.Int
	url     string
}

func (c *verifyTransactionClient) TransactionByHash(ctx context.Context, hash types.Hash) (coretypes.Message, coretypes.TransactionStatus, error) {
	signer := gethtypes.NewEIP155Signer(c.chainID)
	client, err := ethclient.Dial(c.url)
	if err != nil {
		return coretypes.Message{}, coretypes.TransactionStatusPending, err
	}

	transaction, pending, err := client.TransactionByHash(ctx, commongethtypes.BytesToHash(hash.Bytes()))
	if err != nil {
		return coretypes.Message{}, coretypes.TransactionStatusPending, err
	}

	message, err := transaction.AsMessage(signer)
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
		if wallet.IsTokenTransfer(receipt.Logs) {
			return coremessage, coretypes.TransactionStatus(receipt.Status), nil
		}
		return coremessage, coretypes.TransactionStatusFailed, nil
	}

	return coremessage, coretypes.TransactionStatus(receipt.Status), nil
}

func (s *Service) verifyTransactionLoop(tick time.Duration, cancel <-chan struct{}) {
	if s.config.VerifyTransactionURL == "" {
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

// UpdateMailservers updates information about selected mail servers.
func (s *Service) UpdateMailservers(nodes []*enode.Node) error {
	if len(nodes) > 0 && s.messenger != nil {
		s.messenger.SetMailserver(nodes[0].ID().Bytes())
	}
	for _, peer := range nodes {
		s.server.AddPeer(peer)
	}
	if err := s.peerStore.Update(nodes); err != nil {
		return err
	}
	return nil
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	panic("this is abstract service, use shhext or wakuext implementation")
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Start(server *p2p.Server) error {
	s.server = server
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
	}

	return nil
}

func buildMessengerOptions(
	config params.ShhextConfig,
	identity *ecdsa.PrivateKey,
	db *sql.DB,
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
		protocol.WithEnvelopesMonitorConfig(envelopesMonitorConfig),
		protocol.WithSignalsHandler(messengerSignalsHandler),
		protocol.WithENSVerificationConfig(publishMessengerResponse, config.VerifyENSURL, config.VerifyENSContractAddress),
	}

	if config.DataSyncEnabled {
		options = append(options, protocol.WithDatasync())
	}

	settings, err := accountsDB.GetSettings()
	if err != sql.ErrNoRows && err != nil {
		return nil, err
	}

	if settings.PushNotificationsServerEnabled {
		config := &pushnotificationserver.Config{
			Enabled: true,
			Logger:  logger,
		}
		options = append(options, protocol.WithPushNotificationServerConfig(config))
	}

	options = append(options, protocol.WithPushNotificationClientConfig(&pushnotificationclient.Config{
		DefaultServers:             config.DefaultPushNotificationsServers,
		BlockMentions:              settings.PushNotificationsBlockMentions,
		SendEnabled:                settings.SendPushNotifications,
		AllowFromContactsOnly:      settings.PushNotificationsFromContactsOnly,
		RemoteNotificationsEnabled: settings.RemotePushNotificationsEnabled,
	}))

	if config.VerifyTransactionURL != "" {
		client := &verifyTransactionClient{
			url:     config.VerifyTransactionURL,
			chainID: big.NewInt(config.VerifyTransactionChainID),
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
