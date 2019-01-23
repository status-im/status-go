package shhext

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext/chat"
	"github.com/status-im/status-go/services/shhext/dedup"
	"github.com/status-im/status-go/services/shhext/mailservers"
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

var errProtocolNotInitialized = errors.New("procotol is not initialized")

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent(common.Hash)
	EnvelopeExpired(common.Hash)
	MailServerRequestCompleted(common.Hash, common.Hash, []byte, error)
	MailServerRequestExpired(common.Hash)
}

// Service is a service that provides some additional Whisper API.
type Service struct {
	w                *whisper.Whisper
	config           params.ShhextConfig
	tracker          *tracker
	requestsRegistry *RequestsRegistry
	server           *p2p.Server
	nodeID           *ecdsa.PrivateKey
	deduplicator     *dedup.Deduplicator
	protocol         *chat.ProtocolService
	debug            bool
	dataDir          string
	installationID   string
	pfsEnabled       bool

	peerStore       *mailservers.PeerStore
	cache           *mailservers.Cache
	connManager     *mailservers.ConnectionManager
	lastUsedMonitor *mailservers.LastUsedConnectionMonitor
}

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// New returns a new Service. dataDir is a folder path to a network-independent location
func New(w *whisper.Whisper, handler EnvelopeEventsHandler, db *leveldb.DB, config params.ShhextConfig) *Service {
	cache := mailservers.NewCache(db)
	ps := mailservers.NewPeerStore(cache)
	delay := defaultRequestsDelay
	if config.RequestsDelay != 0 {
		delay = config.RequestsDelay
	}
	requestsRegistry := NewRequestsRegistry(delay)
	track := &tracker{
		w:                      w,
		handler:                handler,
		cache:                  map[common.Hash]EnvelopeState{},
		batches:                map[common.Hash]map[common.Hash]struct{}{},
		mailPeers:              ps,
		mailServerConfirmation: config.MailServerConfirmations,
		requestsRegistry:       requestsRegistry,
	}
	return &Service{
		w:                w,
		config:           config,
		tracker:          track,
		requestsRegistry: requestsRegistry,
		deduplicator:     dedup.NewDeduplicator(w, db),
		debug:            config.DebugAPIEnabled,
		dataDir:          config.BackupDisabledDataDir,
		installationID:   config.InstallationID,
		pfsEnabled:       config.PFSEnabled,
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

// InitProtocol create an instance of ProtocolService given an address and password
func (s *Service) InitProtocol(address string, password string) error {
	if !s.pfsEnabled {
		return nil
	}

	digest := sha3.Sum256([]byte(password))
	hashedPassword := fmt.Sprintf("%x", digest)

	if err := os.MkdirAll(filepath.Clean(s.dataDir), os.ModePerm); err != nil {
		return err
	}
	v0Path := filepath.Join(s.dataDir, fmt.Sprintf("%x.db", address))
	v1Path := filepath.Join(s.dataDir, fmt.Sprintf("%s.db", s.installationID))
	v2Path := filepath.Join(s.dataDir, fmt.Sprintf("%s.v2.db", s.installationID))
	v3Path := filepath.Join(s.dataDir, fmt.Sprintf("%s.v3.db", s.installationID))
	v4Path := filepath.Join(s.dataDir, fmt.Sprintf("%s.v4.db", s.installationID))

	if err := chat.MigrateDBFile(v0Path, v1Path, "ON", password); err != nil {
		return err
	}

	if err := chat.MigrateDBFile(v1Path, v2Path, password, hashedPassword); err != nil {
		// Remove db file as created with a blank password and never used,
		// and there's no need to rekey in this case
		os.Remove(v1Path)
		os.Remove(v2Path)
	}

	if err := chat.MigrateDBKeyKdfIterations(v2Path, v3Path, hashedPassword); err != nil {
		os.Remove(v2Path)
		os.Remove(v3Path)
	}

	// Fix IOS not encrypting database
	if err := chat.EncryptDatabase(v3Path, v4Path, hashedPassword); err != nil {
		os.Remove(v3Path)
		os.Remove(v4Path)
	}

	persistence, err := chat.NewSQLLitePersistence(v4Path, hashedPassword)
	if err != nil {
		return err
	}

	addedBundlesHandler := func(addedBundles []chat.IdentityAndIDPair) {
		handler := EnvelopeSignalHandler{}
		for _, bundle := range addedBundles {
			handler.BundleAdded(bundle[0], bundle[1])
		}
	}

	s.protocol = chat.NewProtocolService(chat.NewEncryptionService(persistence, chat.DefaultEncryptionServiceConfig(s.installationID)), addedBundlesHandler)

	return nil
}

func (s *Service) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, bundle *chat.Bundle) ([]chat.IdentityAndIDPair, error) {
	if s.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return s.protocol.ProcessPublicBundle(myIdentityKey, bundle)
}

func (s *Service) GetBundle(myIdentityKey *ecdsa.PrivateKey) (*chat.Bundle, error) {
	if s.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return s.protocol.GetBundle(myIdentityKey)
}

// EnableInstallation enables an installation for multi-device sync.
func (s *Service) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	if s.protocol == nil {
		return errProtocolNotInitialized
	}

	return s.protocol.EnableInstallation(myIdentityKey, installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (s *Service) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	if s.protocol == nil {
		return errProtocolNotInitialized
	}

	return s.protocol.DisableInstallation(myIdentityKey, installationID)
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

	if s.debug {
		apis = append(apis, rpc.API{
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewDebugAPI(s),
			Public:    true,
		})
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
		s.connManager = mailservers.NewConnectionManager(server, s.w, connectionsTarget, defaultTimeoutWaitAdded)
		s.connManager.Start()
		if err := mailservers.EnsureUsedRecordsAddedFirst(s.peerStore, s.connManager); err != nil {
			return err
		}
	}
	if s.config.EnableLastUsedMonitor {
		s.lastUsedMonitor = mailservers.NewLastUsedConnectionMonitor(s.peerStore, s.cache, s.w)
		s.lastUsedMonitor.Start()
	}
	s.tracker.Start()
	s.nodeID = server.PrivateKey
	s.server = server
	return nil
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
	s.tracker.Stop()
	return nil
}
