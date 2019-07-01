package shhext

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/messaging/chat"
	chatDB "github.com/status-im/status-go/messaging/chat/db"
	"github.com/status-im/status-go/messaging/chat/multidevice"
	"github.com/status-im/status-go/messaging/chat/sharedsecret"
	"github.com/status-im/status-go/messaging/filter"
	"github.com/status-im/status-go/messaging/publisher"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext/dedup"
	"github.com/status-im/status-go/services/shhext/mailservers"
	"github.com/status-im/status-go/signal"

	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/crypto/sha3"
)

const (
	// defaultConnectionsTarget used in Service.Start if configured connection target is 0.
	defaultConnectionsTarget = 1
	// defaultTimeoutWaitAdded is a timeout to use to establish initial connections.
	defaultTimeoutWaitAdded = 5 * time.Second
	// maxInstallations is a maximum number of supported devices for one account.
	maxInstallations = 3
)

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent(common.Hash)
	EnvelopeExpired(common.Hash, error)
	MailServerRequestCompleted(common.Hash, common.Hash, []byte, error)
	MailServerRequestExpired(common.Hash)
}

// Service is a service that provides some additional Whisper API.
type Service struct {
	*publisher.Publisher
	storage          db.TransactionalStorage
	w                *whisper.Whisper
	config           params.ShhextConfig
	envelopesMonitor *EnvelopesMonitor
	mailMonitor      *MailRequestMonitor
	requestsRegistry *RequestsRegistry
	historyUpdates   *HistoryUpdateReactor
	server           *p2p.Server
	nodeID           *ecdsa.PrivateKey
	deduplicator     *dedup.Deduplicator
	peerStore        *mailservers.PeerStore
	cache            *mailservers.Cache
	connManager      *mailservers.ConnectionManager
	lastUsedMonitor  *mailservers.LastUsedConnectionMonitor
	filter           *filter.Service
}

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// New returns a new Service.
func New(w *whisper.Whisper, handler EnvelopeEventsHandler, ldb *leveldb.DB, config params.ShhextConfig) *Service {
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
		cache:            map[common.Hash]EnvelopeState{},
		requestsRegistry: requestsRegistry,
	}
	envelopesMonitor := NewEnvelopesMonitor(w, handler, config.MailServerConfirmations, ps, config.MaxMessageDeliveryAttempts)
	publisher := publisher.New(w, publisher.Config{PFSEnabled: config.PFSEnabled})
	return &Service{
		Publisher:        publisher,
		storage:          db.NewLevelDBStorage(ldb),
		w:                w,
		config:           config,
		envelopesMonitor: envelopesMonitor,
		mailMonitor:      mailMonitor,
		requestsRegistry: requestsRegistry,
		historyUpdates:   historyUpdates,
		deduplicator:     dedup.NewDeduplicator(w, ldb),
		peerStore:        ps,
		cache:            cache,
	}
}

func (s *Service) InitProtocolWithPassword(address string, password string) error {
	digest := sha3.Sum256([]byte(password))
	encKey := fmt.Sprintf("%x", digest)
	return s.initProtocol(address, encKey, password)
}

// InitProtocolWithEncyptionKey creates an instance of ProtocolService given an address and encryption key.
func (s *Service) InitProtocolWithEncyptionKey(address string, encKey string) error {
	return s.initProtocol(address, encKey, "")
}

func (s *Service) initProtocol(address, encKey, password string) error {
	if !s.config.PFSEnabled {
		return nil
	}

	dataDir := filepath.Clean(s.config.BackupDisabledDataDir)

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return err
	}
	v0Path := filepath.Join(dataDir, fmt.Sprintf("%x.db", address))
	v1Path := filepath.Join(dataDir, fmt.Sprintf("%s.db", s.config.InstallationID))
	v2Path := filepath.Join(dataDir, fmt.Sprintf("%s.v2.db", s.config.InstallationID))
	v3Path := filepath.Join(dataDir, fmt.Sprintf("%s.v3.db", s.config.InstallationID))
	v4Path := filepath.Join(dataDir, fmt.Sprintf("%s.v4.db", s.config.InstallationID))

	if password != "" {
		if err := chatDB.MigrateDBFile(v0Path, v1Path, "ON", password); err != nil {
			return err
		}

		if err := chatDB.MigrateDBFile(v1Path, v2Path, password, encKey); err != nil {
			// Remove db file as created with a blank password and never used,
			// and there's no need to rekey in this case
			os.Remove(v1Path)
			os.Remove(v2Path)
		}
	}

	if err := chatDB.MigrateDBKeyKdfIterations(v2Path, v3Path, encKey); err != nil {
		os.Remove(v2Path)
		os.Remove(v3Path)
	}

	// Fix IOS not encrypting database
	if err := chatDB.EncryptDatabase(v3Path, v4Path, encKey); err != nil {
		os.Remove(v3Path)
		os.Remove(v4Path)
	}

	// Desktop was passing a network dependent directory, which meant that
	// if running on testnet it would not access the right db. This copies
	// the db from mainnet to the root location.
	networkDependentPath := filepath.Join(dataDir, "ethereum", "mainnet_rpc", fmt.Sprintf("%s.v4.db", s.config.InstallationID))
	if _, err := os.Stat(networkDependentPath); err == nil {
		if err := os.Rename(networkDependentPath, v4Path); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	persistence, err := chat.NewSQLLitePersistence(v4Path, encKey)
	if err != nil {
		return err
	}

	// Initialize sharedsecret
	sharedSecretService := sharedsecret.NewService(persistence.GetSharedSecretStorage())

	onNewMessagesHandler := func(messages []*filter.Messages) {
		handler := PublisherSignalHandler{}
		log.Info("NEW MESSAGES", "msgs", messages)
		handler.NewMessages(messages)
	}
	// Initialize filter
	filterService := filter.New(s.w, filter.NewSQLLitePersistence(persistence.DB), sharedSecretService, onNewMessagesHandler)
	go filterService.Start(300 * time.Millisecond)

	// Initialize multidevice
	multideviceConfig := &multidevice.Config{
		InstallationID:   s.config.InstallationID,
		ProtocolVersion:  chat.ProtocolVersion,
		MaxInstallations: maxInstallations,
	}
	multideviceService := multidevice.New(multideviceConfig, persistence.GetMultideviceStorage())

	addedBundlesHandler := func(addedBundles []*multidevice.Installation) {
		handler := PublisherSignalHandler{}
		for _, bundle := range addedBundles {
			handler.BundleAdded(bundle.Identity, bundle.ID)
		}
	}

	protocolService := chat.NewProtocolService(
		chat.NewEncryptionService(
			persistence,
			chat.DefaultEncryptionServiceConfig(s.config.InstallationID)),
		sharedSecretService,
		multideviceService,
		addedBundlesHandler,
		s.newSharedSecretHandler(filterService))

	s.Publisher.Init(persistence.DB, protocolService, filterService)

	return nil
}

func (s *Service) newSharedSecretHandler(filterService *filter.Service) func([]*sharedsecret.Secret) {
	return func(sharedSecrets []*sharedsecret.Secret) {
		var filters []*signal.Filter
		for _, sharedSecret := range sharedSecrets {
			chat, err := filterService.ProcessNegotiatedSecret(sharedSecret)
			if err != nil {
				log.Error("Failed to process negotiated secret", "err", err)
				return
			}

			filter := &signal.Filter{
				ChatID:   chat.ChatID,
				SymKeyID: chat.SymKeyID,
				Listen:   chat.Listen,
				FilterID: chat.FilterID,
				Identity: chat.Identity,
				Topic:    chat.Topic,
			}

			filters = append(filters, filter)
		}
		if len(filters) != 0 {
			handler := PublisherSignalHandler{}
			handler.WhisperFilterAdded(filters)
		}
	}
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
			Namespace: "shhext",
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
	s.envelopesMonitor.Start()
	s.mailMonitor.Start()
	s.nodeID = server.PrivateKey
	s.server = server
	return s.Publisher.Start(s.online, true)
}

func (s *Service) online() bool {
	return s.server.PeerCount() != 0
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
	s.envelopesMonitor.Stop()
	s.mailMonitor.Stop()
	if s.filter != nil {
		if err := s.filter.Stop(); err != nil {
			log.Error("Failed to stop filter service with error", "err", err)
		}
	}

	return s.Publisher.Stop()
}

func (s *Service) syncMessages(ctx context.Context, mailServerID []byte, r whisper.SyncMailRequest) (resp whisper.SyncEventResponse, err error) {
	err = s.w.SyncMessages(mailServerID, r)
	if err != nil {
		return
	}

	// Wait for the response which is received asynchronously as a p2p packet.
	// This packet handler will send an event which contains the response payload.
	events := make(chan whisper.EnvelopeEvent, 1024)
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
			if event.Event != whisper.EventMailServerSyncFinished {
				continue
			}

			log.Info("received EventMailServerSyncFinished event", "data", event.Data)

			var ok bool

			resp, ok = event.Data.(whisper.SyncEventResponse)
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
