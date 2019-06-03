package shhext

import (
	"crypto/ecdsa"
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
	"github.com/status-im/status-go/services/shhext/filter"
	"github.com/status-im/status-go/services/shhext/mailservers"
	"github.com/status-im/status-go/services/shhext/publisher"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
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
	*publisher.Service
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
	publisherConfig := &publisher.Config{
		PfsEnabled:     config.PFSEnabled,
		DataDir:        config.BackupDisabledDataDir,
		InstallationID: config.InstallationID,
	}
	publisherService := publisher.New(publisherConfig, w)
	return &Service{
		Service:          publisherService,
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
	return s.Service.Start(s.online, true)
}

func (s *Service) online() bool {
	return s.server.PeerCount() != 0
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Stop() error {
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

	return s.Service.Stop()
}
