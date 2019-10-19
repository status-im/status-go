package shhext

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/status-im/status-go/logutils"

	"github.com/ethereum/go-ethereum/crypto"
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
	protocolwhisper "github.com/status-im/status-protocol-go/transport/whisper"
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
	"github.com/syndtr/goleveldb/leveldb"
	"go.uber.org/zap"
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
	MailServerRequestCompleted(statusproto.Hash, statusproto.Hash, []byte, error)
	MailServerRequestExpired(statusproto.Hash)
}

// Service is a service that provides some additional Whisper API.
type Service struct {
	messenger       *protocol.Messenger
	cancelMessenger chan struct{}

	storage          db.TransactionalStorage
	w                whispertypes.Whisper
	config           params.ShhextConfig
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
func New(w whispertypes.Whisper, handler EnvelopeEventsHandler, ldb *leveldb.DB, config params.ShhextConfig) *Service {
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
		cache:            map[statusproto.Hash]EnvelopeState{},
		requestsRegistry: requestsRegistry,
	}
	return &Service{
		storage:          db.NewLevelDBStorage(ldb),
		w:                w,
		config:           config,
		mailMonitor:      mailMonitor,
		requestsRegistry: requestsRegistry,
		historyUpdates:   historyUpdates,
		deduplicator:     dedup.NewDeduplicator(w, ldb),
		peerStore:        ps,
		cache:            cache,
	}
}

func (s *Service) InitProtocol(db *sql.DB) error { // nolint: gocyclo
	if !s.config.PFSEnabled {
		return nil
	}

	dataDir := filepath.Clean(s.config.BackupDisabledDataDir)

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return err
	}

	// Create a custom zap.Logger which will forward logs from status-protocol-go to status-go logger.
	zapLogger, err := logutils.NewZapLoggerWithAdapter(logutils.Logger())
	if err != nil {
		return err
	}

	envelopesMonitorConfig := &protocolwhisper.EnvelopesMonitorConfig{
		MaxAttempts:                    s.config.MaxMessageDeliveryAttempts,
		MailserverConfirmationsEnabled: s.config.MailServerConfirmations,
		IsMailserver: func(peer whispertypes.EnodeID) bool {
			return s.peerStore.Exist(peer)
		},
		EnvelopeEventsHandler: EnvelopeSignalHandler{},
		Logger:                zapLogger,
	}
	options := buildMessengerOptions(s.config, db, envelopesMonitorConfig, zapLogger)

	selectedKeyID := s.w.SelectedKeyPairID()
	identity, err := s.w.GetPrivateKey(selectedKeyID)
	if err != nil {
		return err
	}

	messenger, err := protocol.NewMessenger(
		identity,
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
			var messageIDs []string

			for _, messages := range chatWithMessages {
				for _, message := range messages {
					messageIDs = append(messageIDs, message.ID.String())
				}
			}

			existingMessages, err := s.messenger.MessagesExist(messageIDs)
			if err != nil {
				log.Error("failed to check existing messages", "err", err)
				continue
			}

			var signalMessages []*signal.Messages

			for chat, messages := range chatWithMessages {

				var dedupMessages []*dedup.DeduplicateMessage
				// Filter out already saved messages
				for _, message := range messages {
					if !existingMessages[message.ID.String()] {
						id := fmt.Sprintf("0x%s", hex.EncodeToString(crypto.FromECDSAPub(message.SigPubKey())))

						identicon, err := protocol.Identicon(id)
						if err != nil {
							log.Error("failed to generate identicon", "err", err)
							continue

						}
						alias, err := protocol.GenerateAlias(id)
						if err != nil {
							log.Error("failed to generate identicon", "err", err)
							continue

						}

						dedupMessage := &dedup.DeduplicateMessage{
							Metadata: dedup.Metadata{
								Author: dedup.Author{
									PublicKey: crypto.FromECDSAPub(message.SigPubKey()),
									Alias:     alias,
									Identicon: identicon,
								},
								MessageID:    message.ID,
								EncryptionID: message.Hash,
							},
							Message: message.TransportMessage,
						}
						dedupMessage.Message.Payload = message.DecryptedPayload
						dedupMessage.Payload = string(message.DecryptedPayload)
						dedupMessages = append(dedupMessages, dedupMessage)
					}
				}
				dedupMessages = s.deduplicator.Deduplicate(dedupMessages)

				if len(dedupMessages) != 0 {
					signalMessage := &signal.Messages{
						Chat:     chat,
						Error:    nil, // TODO: what is it needed for?
						Messages: dedupMessages,
					}

					signalMessages = append(signalMessages, signalMessage)
				}
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

func (s *Service) syncMessages(ctx context.Context, mailServerID []byte, r whispertypes.SyncMailRequest) (resp whispertypes.SyncEventResponse, err error) {
	err = s.w.SyncMessages(mailServerID, r)
	if err != nil {
		return
	}

	// Wait for the response which is received asynchronously as a p2p packet.
	// This packet handler will send an event which contains the response payload.
	events := make(chan whispertypes.EnvelopeEvent, 1024)
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
			if event.Event != whispertypes.EventMailServerSyncFinished {
				continue
			}

			log.Info("received EventMailServerSyncFinished event", "data", event.Data)

			var ok bool

			resp, ok = event.Data.(whispertypes.SyncEventResponse)
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

func onNegotiatedFilters(filters []*protocolwhisper.Filter) {
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

func buildMessengerOptions(config params.ShhextConfig, db *sql.DB, envelopesMonitorConfig *protocolwhisper.EnvelopesMonitorConfig, logger *zap.Logger) []protocol.Option {

	options := []protocol.Option{
		protocol.WithCustomLogger(logger),
		protocol.WithDatabase(db),
		protocol.WithEnvelopesMonitorConfig(envelopesMonitorConfig),
		protocol.WithOnNegotiatedFilters(onNegotiatedFilters),
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
	return options
}
