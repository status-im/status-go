package shhext

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/logutils"
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
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext/dedup"
	"github.com/status-im/status-go/services/shhext/mailservers"
	"github.com/status-im/status-go/signal"

	protocol "github.com/status-im/status-protocol-go"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/crypto/sha3"
)

const (
	// defaultConnectionsTarget used in Service.Start if configured connection target is 0.
	defaultConnectionsTarget = 1
	// defaultTimeoutWaitAdded is a timeout to use to establish initial connections.
	defaultTimeoutWaitAdded = 5 * time.Second
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
	messenger       *protocol.Messenger
	cancelMessenger chan struct{}

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
	return &Service{
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

func (s *Service) initProtocol(address, encKey, password string) error { // nolint: gocyclo
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
		if err := migrateDBFile(v0Path, v1Path, "ON", password); err != nil {
			return err
		}

		if err := migrateDBFile(v1Path, v2Path, password, encKey); err != nil {
			// Remove db file as created with a blank password and never used,
			// and there's no need to rekey in this case
			os.Remove(v1Path)
			os.Remove(v2Path)
		}
	}

	if err := migrateDBKeyKdfIterations(v2Path, v3Path, encKey); err != nil {
		os.Remove(v2Path)
		os.Remove(v3Path)
	}

	// Fix IOS not encrypting database
	if err := encryptDatabase(v3Path, v4Path, encKey); err != nil {
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

	// In one of the versions, we split the database file into multiple ones.
	// Later, we discovered that it really hurts the performance so we consolidated
	// it again but in a better way keeping migrations in separate packages.
	sessionsDatabasePath := filepath.Join(dataDir, fmt.Sprintf("%s.sessions.v4.sql", s.config.InstallationID))
	sessionsStat, sessionsStatErr := os.Stat(sessionsDatabasePath)
	v4PathStat, v4PathStatErr := os.Stat(v4Path)

	if sessionsStatErr == nil && os.IsNotExist(v4PathStatErr) {
		// This is a clear situation where we have the sessions.v4.sql file and v4Path does not exist.
		// In the previous migration, we removed v4Path when it is successfully copied into the sessions sql file.
		if err := os.Rename(sessionsDatabasePath, v4Path); err != nil {
			return err
		}
	} else if sessionsStatErr == nil && v4PathStatErr == nil {
		// Both files exist so probably the migration to split databases failed.
		if sessionsStat.ModTime().After(v4PathStat.ModTime()) {
			// Sessions sql file is newer.
			if err := os.Rename(sessionsDatabasePath, v4Path); err != nil {
				return err
			}
		}
	}

	options, err := buildMessengerOptions(s.config, v4Path, encKey)
	if err != nil {
		return err
	}

	selectedKeyID := s.w.SelectedKeyPairID()
	identity, err := s.w.GetPrivateKey(selectedKeyID)
	if err != nil {
		return err
	}

	messenger, err := protocol.NewMessenger(
		identity,
		&server{server: s.server},
		s.w,
		s.config.InstallationID,
		options...,
	)
	if err != nil {
		return err
	}
	s.messenger = messenger
	// Start a loop that retrieves all messages and propagates them to status-react.
	s.cancelMessenger = make(chan struct{})
	go s.retrieveMessagesLoop(time.Second, s.cancelMessenger)

	return nil
}

func (s *Service) retrieveMessagesLoop(tick time.Duration, cancel <-chan struct{}) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			chatWithMessages, err := s.messenger.RetrieveRawAll()
			if err != nil {
				log.Error("failed to retrieve raw messages", "err", err)
				continue
			}

			var signalMessages []*signal.Messages

			for chat, messages := range chatWithMessages {
				var retrievedMessages []*whisper.Message
				for _, message := range messages {
					whisperMessage := message.TransportMessage
					whisperMessage.Payload = message.DecryptedPayload
					retrievedMessages = append(retrievedMessages, whisperMessage)
				}

				signalMessage := &signal.Messages{
					Chat:     chat,
					Error:    nil, // TODO: what is it needed for?
					Messages: s.deduplicator.Deduplicate(retrievedMessages),
				}
				signalMessages = append(signalMessages, signalMessage)
			}

			log.Debug("retrieve messages loop", "messages", len(signalMessages))

			if len(signalMessages) == 0 {
				continue
			}

			PublisherSignalHandler{}.NewMessages(signalMessages)
		case <-cancel:
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
	s.envelopesMonitor.Stop()
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

func buildMessengerOptions(config params.ShhextConfig, dbPath, dbKey string) ([]protocol.Option, error) {
	// Create a custom zap.Logger which will forward logs from status-protocol-go to status-go logger.
	zapLogger, err := logutils.NewZapLoggerWithAdapter(logutils.Logger())
	if err != nil {
		return nil, err
	}

	options := []protocol.Option{
		protocol.WithCustomLogger(zapLogger),
		protocol.WithDatabaseConfig(dbPath, dbKey),
	}

	if !config.DisableGenericDiscoveryTopic {
		options = append(options, protocol.WithGenericDiscoveryTopicSupport())
	}

	if config.DataSyncEnabled {
		options = append(options, protocol.WithDatasync())
	}

	if config.SendV1Messages {
		options = append(options, protocol.WithSendV1Messages())
	}
	return options, nil
}

func (s *Service) afterPost(hash []byte, newMessage whisper.NewMessage) hexutil.Bytes {
	s.envelopesMonitor.Add(common.BytesToHash(hash), newMessage)
	mID := messageID(newMessage)
	return mID[:]
}
