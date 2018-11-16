package shhext

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/services/shhext/chat"
	"github.com/status-im/status-go/services/shhext/dedup"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
)

var errProtocolNotInitialized = errors.New("procotol is not initialized")

// EnvelopeState in local tracker
type EnvelopeState int

const (
	// EnvelopePosted is set when envelope was added to a local whisper queue.
	EnvelopePosted EnvelopeState = iota
	// EnvelopeSent is set when envelope is sent to atleast one peer.
	EnvelopeSent
	// MailServerRequestSent is set when p2p request is sent to the mailserver
	MailServerRequestSent
)

// EnvelopeEventsHandler used for two different event types.
type EnvelopeEventsHandler interface {
	EnvelopeSent(common.Hash)
	EnvelopeExpired(common.Hash)
	MailServerRequestCompleted(common.Hash, common.Hash, []byte, error)
	MailServerRequestExpired(common.Hash)
}

// Service is a service that provides some additional Whisper API.
type Service struct {
	w              *whisper.Whisper
	tracker        *tracker
	nodeID         *ecdsa.PrivateKey
	deduplicator   *dedup.Deduplicator
	protocol       *chat.ProtocolService
	debug          bool
	dataDir        string
	installationID string
	pfsEnabled     bool
}

type ServiceConfig struct {
	DataDir        string
	InstallationID string
	Debug          bool
	PFSEnabled     bool
}

// Make sure that Service implements node.Service interface.
var _ node.Service = (*Service)(nil)

// New returns a new Service. dataDir is a folder path to a network-independent location
func New(w *whisper.Whisper, handler EnvelopeEventsHandler, db *leveldb.DB, config *ServiceConfig) *Service {
	track := &tracker{
		w:       w,
		handler: handler,
		cache:   map[common.Hash]EnvelopeState{},
	}
	return &Service{
		w:              w,
		tracker:        track,
		deduplicator:   dedup.NewDeduplicator(w, db),
		debug:          config.Debug,
		dataDir:        config.DataDir,
		installationID: config.InstallationID,
		pfsEnabled:     config.PFSEnabled,
	}
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

	if err := os.MkdirAll(filepath.Clean(s.dataDir), os.ModePerm); err != nil {
		return err
	}
	persistence, err := chat.NewSQLLitePersistence(filepath.Join(s.dataDir, fmt.Sprintf("%x.db", address)), password)
	if err != nil {
		return err
	}
	s.protocol = chat.NewProtocolService(chat.NewEncryptionService(persistence, chat.DefaultEncryptionServiceConfig(s.installationID)))

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
	s.tracker.Start()
	s.nodeID = server.PrivateKey
	return nil
}

// Stop is run when a service is stopped.
// It does nothing in this case but is required by `node.Service` interface.
func (s *Service) Stop() error {
	s.tracker.Stop()
	return nil
}

// tracker responsible for processing events for envelopes that we are interested in
// and calling specified handler.
type tracker struct {
	w       *whisper.Whisper
	handler EnvelopeEventsHandler

	mu    sync.Mutex
	cache map[common.Hash]EnvelopeState

	wg   sync.WaitGroup
	quit chan struct{}
}

// Start processing events.
func (t *tracker) Start() {
	t.quit = make(chan struct{})
	t.wg.Add(1)
	go func() {
		t.handleEnvelopeEvents()
		t.wg.Done()
	}()
}

// Stop process events.
func (t *tracker) Stop() {
	close(t.quit)
	t.wg.Wait()
}

// Add hash to a tracker.
func (t *tracker) Add(hash common.Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache[hash] = EnvelopePosted
}

// Add request hash to a tracker.
func (t *tracker) AddRequest(hash common.Hash, timerC <-chan time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache[hash] = MailServerRequestSent
	go t.expireRequest(hash, timerC)
}

func (t *tracker) expireRequest(hash common.Hash, timerC <-chan time.Time) {
	select {
	case <-t.quit:
		return
	case <-timerC:
		t.handleEvent(whisper.EnvelopeEvent{
			Event: whisper.EventMailServerRequestExpired,
			Hash:  hash,
		})
	}
}

// handleEnvelopeEvents processes whisper envelope events
func (t *tracker) handleEnvelopeEvents() {
	events := make(chan whisper.EnvelopeEvent, 100) // must be buffered to prevent blocking whisper
	sub := t.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	for {
		select {
		case <-t.quit:
			return
		case event := <-events:
			t.handleEvent(event)
		}
	}
}

// handleEvent based on type of the event either triggers
// confirmation handler or removes hash from tracker
func (t *tracker) handleEvent(event whisper.EnvelopeEvent) {
	handlers := map[whisper.EventType]func(whisper.EnvelopeEvent){
		whisper.EventEnvelopeSent:               t.handleEventEnvelopeSent,
		whisper.EventEnvelopeExpired:            t.handleEventEnvelopeExpired,
		whisper.EventMailServerRequestCompleted: t.handleEventMailServerRequestCompleted,
		whisper.EventMailServerRequestExpired:   t.handleEventMailServerRequestExpired,
	}

	if handler, ok := handlers[event.Event]; ok {
		handler(event)
	}
}

func (t *tracker) handleEventEnvelopeSent(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, ok := t.cache[event.Hash]
	// if we didn't send a message using extension - skip it
	// if message was already confirmed - skip it
	if !ok || state == EnvelopeSent {
		return
	}
	log.Debug("envelope is sent", "hash", event.Hash, "peer", event.Peer)
	t.cache[event.Hash] = EnvelopeSent
	if t.handler != nil {
		t.handler.EnvelopeSent(event.Hash)
	}
}

func (t *tracker) handleEventEnvelopeExpired(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if state, ok := t.cache[event.Hash]; ok {
		log.Debug("envelope expired", "hash", event.Hash, "state", state)
		delete(t.cache, event.Hash)
		if state == EnvelopeSent {
			return
		}
		if t.handler != nil {
			t.handler.EnvelopeExpired(event.Hash)
		}
	}
}

func (t *tracker) handleEventMailServerRequestCompleted(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, ok := t.cache[event.Hash]
	if !ok || state != MailServerRequestSent {
		return
	}
	log.Debug("mailserver response received", "hash", event.Hash)
	delete(t.cache, event.Hash)
	if t.handler != nil {
		if resp, ok := event.Data.(*whisper.MailServerResponse); ok {
			t.handler.MailServerRequestCompleted(event.Hash, resp.LastEnvelopeHash, resp.Cursor, resp.Error)
		}
	}
}

func (t *tracker) handleEventMailServerRequestExpired(event whisper.EnvelopeEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, ok := t.cache[event.Hash]
	if !ok || state != MailServerRequestSent {
		return
	}
	log.Debug("mailserver response expired", "hash", event.Hash)
	delete(t.cache, event.Hash)
	if t.handler != nil {
		t.handler.MailServerRequestExpired(event.Hash)
	}
}
