package shhext

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/shhext/chat"
	"github.com/status-im/status-go/services/shhext/dedup"
	"github.com/status-im/status-go/services/shhext/mailservers"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
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

// LoadedFilter is a record containing data about an added filter on whisper
type LoadedFilter struct {
	SymKeyID string
	FilterID string
}

// Service is a service that provides some additional Whisper API.
type Service struct {
	w *whisper.Whisper
	// Here we would like to use whisper service instead of the public api
	// but at the present time quite a few logic is in the publicAPI, making
	// it difficult to use the service directly
	publicAPI      *whisper.PublicWhisperAPI
	config         *ServiceConfig
	tracker        *tracker
	server         *p2p.Server
	nodeID         *ecdsa.PrivateKey
	deduplicator   *dedup.Deduplicator
	protocol       *chat.ProtocolService
	debug          bool
	dataDir        string
	installationID string
	pfsEnabled     bool

	peerStore       *mailservers.PeerStore
	cache           *mailservers.Cache
	connManager     *mailservers.ConnectionManager
	lastUsedMonitor *mailservers.LastUsedConnectionMonitor

	loadedFilters map[string]LoadedFilter

	log log.Logger
}

type ServiceConfig struct {
	DataDir                 string
	InstallationID          string
	Debug                   bool
	PFSEnabled              bool
	MailServerConfirmations bool
	EnableConnectionManager bool
	EnableLastUsedMonitor   bool
	ConnectionTarget        int
}

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// New returns a new Service. dataDir is a folder path to a network-independent location
func New(w *whisper.Whisper, handler EnvelopeEventsHandler, db *leveldb.DB, config *ServiceConfig) *Service {
	cache := mailservers.NewCache(db)
	ps := mailservers.NewPeerStore(cache)
	track := &tracker{
		w:                      w,
		handler:                handler,
		cache:                  map[common.Hash]EnvelopeState{},
		batches:                map[common.Hash]map[common.Hash]struct{}{},
		mailPeers:              ps,
		mailServerConfirmation: config.MailServerConfirmations,
	}
	return &Service{
		w:              w,
		publicAPI:      whisper.NewPublicWhisperAPI(w),
		config:         config,
		tracker:        track,
		deduplicator:   dedup.NewDeduplicator(w, db),
		debug:          config.Debug,
		dataDir:        config.DataDir,
		installationID: config.InstallationID,
		pfsEnabled:     config.PFSEnabled,
		peerStore:      ps,
		cache:          cache,
		log:            log.New("package", "status-go/services/sshext.Service"),
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

	if err := chat.MigrateDBFile(v0Path, v1Path, "ON", password); err != nil {
		return err
	}

	if err := chat.MigrateDBFile(v1Path, v2Path, password, hashedPassword); err != nil {
		// Remove db file as created with a blank password and never used,
		// and there's no need to rekey in this case
		os.Remove(v1Path)
		os.Remove(v2Path)
	}

	persistence, err := chat.NewSQLLitePersistence(v2Path, hashedPassword)
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

func (s *Service) GetPublicBundle(identityKey *ecdsa.PublicKey) (*chat.Bundle, error) {
	if s.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return s.protocol.GetPublicBundle(identityKey)
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

// JoinPublicChats join a set of public chats deriving the key from the chat name
func (s *Service) JoinPublicChats(chats []string) ([]string, error) {
	var response []string

	for _, chatID := range chats {
		// Don't allow to join multiple times
		_, found := s.loadedFilters[chatID]
		if found {
			continue
		}

		symKeyID, err := s.w.AddSymKeyFromPassword(chatID)
		if err != nil {
			return nil, err
		}

		symKey, err := s.w.GetSymKey(symKeyID)
		if err != nil {
			return nil, err
		}

		topic := chat.ToWhisperTopic(chatID)
		topics := [][]byte{topic[:]}

		filterOpts := &whisper.Filter{
			KeySym:   symKey,
			Topics:   topics,
			AllowP2P: true,
			Messages: make(map[common.Hash]*whisper.ReceivedMessage),
		}
		filter, err := s.w.Subscribe(filterOpts)
		if err != nil {
			return nil, err
		}

		loadedFilter := LoadedFilter{
			SymKeyID: symKeyID,
			FilterID: filter,
		}

		s.loadedFilters[chatID] = loadedFilter
		response = append(response, filter)
	}
	return response, nil
}

// LeavePublicChats leaves set of public chats
func (s *Service) LeavePublicChats(chats []string) error {
	for _, chatID := range chats {
		loadedFilter, present := s.loadedFilters[chatID]
		if !present {
			continue
		}

		s.w.DeleteSymKey(loadedFilter.SymKeyID)

		err := s.w.Unsubscribe(loadedFilter.FilterID)
		if err != nil {
			return err
		}

		delete(s.loadedFilters, chatID)

	}
	return nil
}

// Post shamelessly copied from whisper codebase with slight modifications.
func (s *Service) Post(ctx context.Context, req whisper.NewMessage) (hash hexutil.Bytes, err error) {
	hash, err = s.publicAPI.Post(ctx, req)
	if err == nil {
		var envHash common.Hash
		envHash.SetBytes(hash)
		s.tracker.Add(envHash)
	}
	return hash, err
}

// SendPairingMessage sends a 1:1 chat message to our own devices to initiate a pairing session
func (s *Service) SendPairingMessage(ctx context.Context, msg chat.SendDirectMessageRPC) ([]hexutil.Bytes, error) {
	if !s.pfsEnabled {
		return nil, ErrPFSNotEnabled
	}
	// To be completely agnostic from whisper we should not be using whisper to store the key
	privateKey, err := s.w.GetPrivateKey(msg.Sig)
	if err != nil {
		return nil, err
	}

	msg.PubKey = crypto.FromECDSAPub(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	protocolMessage, err := s.protocol.BuildPairingMessage(privateKey, msg.Payload)
	if err != nil {
		return nil, err
	}

	var response []hexutil.Bytes

	// Enrich with transport layer info
	whisperMessage := chat.DirectMessageToWhisper(msg, protocolMessage)

	// And dispatch
	hash, err := s.Post(ctx, whisperMessage)
	if err != nil {
		return nil, err
	}
	response = append(response, hash)

	return response, nil
}

// SendPublicMessage sends a public chat message to the underlying transport
func (s *Service) SendPublicMessage(ctx context.Context, msg chat.SendPublicMessageRPC) (hexutil.Bytes, error) {

	if msg.PFS {
		privateKey, err := s.w.GetPrivateKey(msg.Sig)
		if err != nil {
			return nil, err
		}

		// This is transport layer agnostic
		protocolMessage, err := s.protocol.BuildPublicMessage(privateKey, msg.Payload)
		if err != nil {
			return nil, err
		}

		msg.Payload = protocolMessage
	}

	// Enrich with transport layer info
	whisperMessage, err := s.buildPublicMessage(msg)
	if err != nil {
		return nil, err
	}

	// And dispatch
	return s.Post(ctx, *whisperMessage)
}

// SendDirectMessage sends a 1:1 chat message to the underlying transport
func (s *Service) SendDirectMessage(ctx context.Context, msg chat.SendDirectMessageRPC) ([]hexutil.Bytes, error) {
	if !s.pfsEnabled {
		return nil, ErrPFSNotEnabled
	}
	// To be completely agnostic from whisper we should not be using whisper to store the key
	privateKey, err := s.w.GetPrivateKey(msg.Sig)
	if err != nil {
		return nil, err
	}

	publicKey, err := crypto.UnmarshalPubkey(msg.PubKey)
	if err != nil {
		return nil, err
	}

	// This is transport layer-agnostic
	protocolMessages, err := s.protocol.BuildDirectMessage(privateKey, msg.Payload, publicKey)
	if err != nil {
		return nil, err
	}

	var response []hexutil.Bytes

	for key, message := range protocolMessages {
		msg.PubKey = crypto.FromECDSAPub(key)
		// Enrich with transport layer info
		whisperMessage := chat.DirectMessageToWhisper(msg, message)

		// And dispatch
		hash, err := s.Post(ctx, whisperMessage)
		if err != nil {
			return nil, err
		}
		response = append(response, hash)

	}
	return response, nil
}

// SendGroupMessage sends a group messag chat message to the underlying transport
func (s *Service) SendGroupMessage(ctx context.Context, msg chat.SendGroupMessageRPC) ([]hexutil.Bytes, error) {
	if !s.pfsEnabled {
		return nil, ErrPFSNotEnabled
	}

	// To be completely agnostic from whisper we should not be using whisper to store the key
	privateKey, err := s.w.GetPrivateKey(msg.Sig)
	if err != nil {
		return nil, err
	}

	var keys []*ecdsa.PublicKey

	for _, k := range msg.PubKeys {
		publicKey, err := crypto.UnmarshalPubkey(k)
		if err != nil {
			return nil, err
		}
		keys = append(keys, publicKey)
	}

	// This is transport layer-agnostic
	protocolMessages, err := s.protocol.BuildDirectMessage(privateKey, msg.Payload, keys...)
	if err != nil {
		return nil, err
	}

	var response []hexutil.Bytes

	for key, message := range protocolMessages {
		directMessage := chat.SendDirectMessageRPC{
			PubKey:  crypto.FromECDSAPub(key),
			Payload: msg.Payload,
			Sig:     msg.Sig,
		}

		// Enrich with transport layer info
		whisperMessage := chat.DirectMessageToWhisper(directMessage, message)

		// And dispatch
		hash, err := s.Post(ctx, whisperMessage)
		if err != nil {
			return nil, err
		}
		response = append(response, hash)

	}
	return response, nil
}

// GetNewFilterMessages is a prototype method with deduplication
func (s *Service) GetNewFilterMessages(filterID string) ([]*whisper.Message, error) {
	msgs, err := s.publicAPI.GetFilterMessages(filterID)
	if err != nil {
		return nil, err
	}

	dedupMessages := s.deduplicator.Deduplicate(msgs)

	if s.pfsEnabled {
		// Attempt to decrypt message, otherwise leave unchanged
		for _, msg := range dedupMessages {
			if err := s.processPFSMessage(msg); err != nil {
				return nil, err
			}
		}
	}

	return dedupMessages, nil
}

// Login initialize the protocol and join the chat for advertising the bundle
func (s *Service) Login(address string, password string) error {
	s.loadedFilters = make(map[string]LoadedFilter)
	return s.InitProtocol(address, password)

}

// Logout cleans up any filter
func (s *Service) Logout() error {
	var chats []string
	for chat := range s.loadedFilters {
		chats = append(chats, chat)

	}

	return s.LeavePublicChats(chats)
}

func (s *Service) processPFSMessage(msg *whisper.Message) error {
	privateKeyID := s.w.SelectedKeyPairID()
	if privateKeyID == "" {
		return errors.New("no key selected")
	}

	privateKey, err := s.w.GetPrivateKey(privateKeyID)
	if err != nil {
		return err
	}

	publicKey, err := crypto.UnmarshalPubkey(msg.Sig)
	if err != nil {
		return err
	}

	response, err := s.protocol.HandleMessage(privateKey, publicKey, msg.Payload)

	// Notify that someone tried to contact us using an invalid bundle
	if err == chat.ErrDeviceNotFound && privateKey.PublicKey != *publicKey {
		s.log.Warn("Device not found, sending signal", "err", err)
		keyString := fmt.Sprintf("0x%x", crypto.FromECDSAPub(publicKey))
		handler := EnvelopeSignalHandler{}
		handler.DecryptMessageFailed(keyString)
		return nil
	} else if err != nil {
		// Ignore errors for now as those might be non-pfs messages
		s.log.Error("Failed handling message with error", "err", err)
		return nil
	}

	// Add unencrypted payload
	msg.Payload = response

	return nil
}

// buildPublicMessage sends a public chat message to the underlying transport
func (s *Service) buildPublicMessage(msg chat.SendPublicMessageRPC) (*whisper.NewMessage, error) {
	filter := s.loadedFilters[msg.Chat]
	symKeyID := filter.SymKeyID

	// Enrich with transport layer info
	whisperMessage := chat.PublicMessageToWhisper(msg, msg.Payload)
	whisperMessage.SymKeyID = symKeyID

	return &whisperMessage, nil
}
