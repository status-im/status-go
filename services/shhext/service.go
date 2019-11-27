// +build !nimbus

package shhext

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/status-im/status-go/logutils"

	commongethtypes "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext/mailservers"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/signal"

	"github.com/syndtr/goleveldb/leveldb"
	"go.uber.org/zap"

	coretypes "github.com/status-im/status-go/eth-node/core/types"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/transport"
)

const (
	// defaultConnectionsTarget used in Service.Start if configured connection target is 0.
	defaultConnectionsTarget = 1
	// defaultTimeoutWaitAdded is a timeout to use to establish initial connections.
	defaultTimeoutWaitAdded = 5 * time.Second
)

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent([][]byte)
	EnvelopeExpired([][]byte, error)
	MailServerRequestCompleted(types.Hash, types.Hash, []byte, error)
	MailServerRequestExpired(types.Hash)
}

// Service is a service that provides some additional Whisper API.
type Service struct {
	apiName          string
	messenger        *protocol.Messenger
	identity         *ecdsa.PrivateKey
	cancelMessenger  chan struct{}
	storage          db.TransactionalStorage
	n                types.Node
	w                types.Whisper
	config           params.ShhextConfig
	mailMonitor      *MailRequestMonitor
	requestsRegistry *RequestsRegistry
	historyUpdates   *HistoryUpdateReactor
	server           *p2p.Server
	nodeID           *ecdsa.PrivateKey
	peerStore        *mailservers.PeerStore
	cache            *mailservers.Cache
	connManager      *mailservers.ConnectionManager
	lastUsedMonitor  *mailservers.LastUsedConnectionMonitor
	accountsDB       *accounts.Database
}

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// New returns a new shhext Service.
func New(n types.Node, ctx interface{}, apiName string, handler EnvelopeEventsHandler, ldb *leveldb.DB, config params.ShhextConfig) *Service {
	w, err := n.GetWhisper(ctx)
	if err != nil {
		panic(err)
	}
	cache := mailservers.NewCache(ldb)
	ps := mailservers.NewPeerStore(cache)
	delay := defaultRequestsDelay
	if config.RequestsDelay != 0 {
		delay = config.RequestsDelay
	}
	requestsRegistry := NewRequestsRegistry(delay)
	historyUpdates := NewHistoryUpdateReactor()
	mailMonitor := &MailRequestMonitor{
		w:                w,
		handler:          handler,
		cache:            map[types.Hash]EnvelopeState{},
		requestsRegistry: requestsRegistry,
	}
	return &Service{
		apiName:          apiName,
		storage:          db.NewLevelDBStorage(ldb),
		n:                n,
		w:                w,
		config:           config,
		mailMonitor:      mailMonitor,
		requestsRegistry: requestsRegistry,
		historyUpdates:   historyUpdates,
		peerStore:        ps,
		cache:            cache,
	}
}

func (s *Service) InitProtocol(identity *ecdsa.PrivateKey, db *sql.DB) error { // nolint: gocyclo
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

	// Create a custom zap.Logger which will forward logs from status-go/protocol to status-go logger.
	zapLogger, err := logutils.NewZapLoggerWithAdapter(logutils.Logger())
	if err != nil {
		return err
	}

	envelopesMonitorConfig := &transport.EnvelopesMonitorConfig{
		MaxAttempts:                    s.config.MaxMessageDeliveryAttempts,
		MailserverConfirmationsEnabled: s.config.MailServerConfirmations,
		IsMailserver: func(peer types.EnodeID) bool {
			return s.peerStore.Exist(peer)
		},
		EnvelopeEventsHandler: EnvelopeSignalHandler{},
		Logger:                zapLogger,
	}
	options := buildMessengerOptions(s.config, db, envelopesMonitorConfig, zapLogger)

	messenger, err := protocol.NewMessenger(
		identity,
		s.n,
		s.config.InstallationID,
		options...,
	)
	if err != nil {
		return err
	}
	s.accountsDB = accounts.NewDB(db)
	s.messenger = messenger
	// Start a loop that retrieves all messages and propagates them to status-react.
	s.cancelMessenger = make(chan struct{})
	go s.retrieveMessagesLoop(time.Second, s.cancelMessenger)
	go s.verifyTransactionLoop(30*time.Second, s.cancelMessenger)

	return s.messenger.Init()
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
			if !response.IsEmpty() {
				PublisherSignalHandler{}.NewMessages(response)
			}
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
		} else {
			return coremessage, coretypes.TransactionStatusFailed, nil
		}

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
				if account.Wallet {
					wallets = append(wallets, types.BytesToAddress(account.Address.Bytes()))
				}
			}

			response, err := s.messenger.ValidateTransactions(ctx, wallets)
			if err != nil {
				log.Error("failed to validate transactions", "err", err)
				continue
			}
			if !response.IsEmpty() {
				PublisherSignalHandler{}.NewMessages(response)
			}
		case <-cancel:
			cancelVerifyTransaction()
			return
		}
	}
}

func (s *Service) ConfirmMessagesProcessed(messageIDs [][]byte) error {
	return s.messenger.ConfirmMessagesProcessed(messageIDs)
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
	if err := s.peerStore.Update(nodes); err != nil {
		return err
	}
	if s.connManager != nil {
		s.connManager.Notify(nodes)
	}
	return nil
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: s.apiName,
			Version:   "1.0",
			Service:   NewPublicAPI(s),
			Public:    true,
		},
	}
	return apis
}

// Start is run when a service is started.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Start(server *p2p.Server) error {
	if s.config.EnableConnectionManager {
		connectionsTarget := s.config.ConnectionTarget
		if connectionsTarget == 0 {
			connectionsTarget = defaultConnectionsTarget
		}
		maxFailures := s.config.MaxServerFailures
		// if not defined change server on first expired event
		if maxFailures == 0 {
			maxFailures = 1
		}
		s.connManager = mailservers.NewConnectionManager(server, s.w, connectionsTarget, maxFailures, defaultTimeoutWaitAdded)
		s.connManager.Start()
		if err := mailservers.EnsureUsedRecordsAddedFirst(s.peerStore, s.connManager); err != nil {
			return err
		}
	}
	if s.config.EnableLastUsedMonitor {
		s.lastUsedMonitor = mailservers.NewLastUsedConnectionMonitor(s.peerStore, s.cache, s.w)
		s.lastUsedMonitor.Start()
	}
	s.mailMonitor.Start()
	s.nodeID = server.PrivateKey
	s.server = server
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	log.Info("Stopping shhext service")
	if s.config.EnableConnectionManager {
		s.connManager.Stop()
	}
	if s.config.EnableLastUsedMonitor {
		s.lastUsedMonitor.Stop()
	}
	s.requestsRegistry.Clear()
	s.mailMonitor.Stop()

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
			return err
		}
	}

	return nil
}

func (s *Service) syncMessages(ctx context.Context, mailServerID []byte, r types.SyncMailRequest) (resp types.SyncEventResponse, err error) {
	err = s.w.SyncMessages(mailServerID, r)
	if err != nil {
		return
	}

	// Wait for the response which is received asynchronously as a p2p packet.
	// This packet handler will send an event which contains the response payload.
	events := make(chan types.EnvelopeEvent, 1024)
	sub := s.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	// Add explicit timeout context, otherwise the request
	// can hang indefinitely if not specified by the sender.
	// Sender is usually through netcat or some bash tool
	// so it's not really possible to specify the timeout.
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	for {
		select {
		case event := <-events:
			if event.Event != types.EventMailServerSyncFinished {
				continue
			}

			log.Info("received EventMailServerSyncFinished event", "data", event.Data)

			var ok bool

			resp, ok = event.Data.(types.SyncEventResponse)
			if !ok {
				err = fmt.Errorf("did not understand the response event data")
				return
			}
			return
		case <-timeoutCtx.Done():
			err = timeoutCtx.Err()
			return
		}
	}
}

func onNegotiatedFilters(filters []*transport.Filter) {
	var signalFilters []*signal.Filter
	for _, filter := range filters {

		signalFilter := &signal.Filter{
			ChatID:   filter.ChatID,
			SymKeyID: filter.SymKeyID,
			Listen:   filter.Listen,
			FilterID: filter.FilterID,
			Identity: filter.Identity,
			Topic:    filter.Topic,
		}

		signalFilters = append(signalFilters, signalFilter)
	}
	if len(filters) != 0 {
		handler := PublisherSignalHandler{}
		handler.WhisperFilterAdded(signalFilters)
	}
}

func buildMessengerOptions(
	config params.ShhextConfig,
	db *sql.DB,
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig,
	logger *zap.Logger,
) []protocol.Option {
	options := []protocol.Option{
		protocol.WithCustomLogger(logger),
		protocol.WithDatabase(db),
		protocol.WithEnvelopesMonitorConfig(envelopesMonitorConfig),
		protocol.WithOnNegotiatedFilters(onNegotiatedFilters),
	}

	if config.DataSyncEnabled {
		options = append(options, protocol.WithDatasync())
	}

	if config.VerifyTransactionURL != "" {
		client := &verifyTransactionClient{
			url:     config.VerifyTransactionURL,
			chainID: big.NewInt(config.VerifyTransactionChainID),
		}
		options = append(options, protocol.WithVerifyTransactionClient(client))
	}

	return options
}
