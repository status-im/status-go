package adapters

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/pkg/errors"
	"github.com/status-im/status-console-client/protocol/v1"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/shhext/chat"
	whisper "github.com/status-im/whisper/whisperv6"
)

type whisperServiceKeysManager struct {
	shh *whisper.Whisper

	// Identity of the current user.
	privateKey *ecdsa.PrivateKey

	passToSymKeyMutex sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *whisperServiceKeysManager) PrivateKey() *ecdsa.PrivateKey {
	return m.privateKey
}

func (m *whisperServiceKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in Whisper
	return m.shh.AddKeyPair(priv)
}

func (m *whisperServiceKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
	m.passToSymKeyMutex.Lock()
	defer m.passToSymKeyMutex.Unlock()

	if val, ok := m.passToSymKeyCache[password]; ok {
		return val, nil
	}

	id, err := m.shh.AddSymKeyFromPassword(password)
	if err != nil {
		return id, err
	}

	m.passToSymKeyCache[password] = id

	return id, nil
}

func (m *whisperServiceKeysManager) GetRawSymKey(id string) ([]byte, error) {
	return m.shh.GetSymKey(id)
}

// WhisperServiceAdapter is an adapter for Whisper service
// the implements Protocol interface.
type WhisperServiceAdapter struct {
	node        *node.StatusNode // TODO: replace with an interface
	shh         *whisper.Whisper
	keysManager *whisperServiceKeysManager

	pfs *chat.ProtocolService

	selectedMailServerEnode string
}

// WhisperServiceAdapter must implement Protocol interface.
var _ protocol.Protocol = (*WhisperServiceAdapter)(nil)

// NewWhisperServiceAdapter returns a new WhisperServiceAdapter.
func NewWhisperServiceAdapter(node *node.StatusNode, shh *whisper.Whisper, privateKey *ecdsa.PrivateKey) *WhisperServiceAdapter {
	return &WhisperServiceAdapter{
		node: node,
		shh:  shh,
		keysManager: &whisperServiceKeysManager{
			shh:               shh,
			privateKey:        privateKey,
			passToSymKeyCache: make(map[string]string),
		},
	}
}

// InitPFS adds support for PFS messages.
func (a *WhisperServiceAdapter) InitPFS(baseDir string) error {
	const (
		// TODO: manage these values properly
		dbFileName    = "pfs_v1.db"
		sqlSecretKey  = "enc-key-abc"
		instalationID = "instalation-1"
	)

	dbPath := filepath.Join(baseDir, dbFileName)
	persistence, err := chat.NewSQLLitePersistence(dbPath, sqlSecretKey)
	if err != nil {
		return err
	}

	addBundlesHandler := func(addedBundles []chat.IdentityAndIDPair) {
		log.Printf("added bundles: %v", addedBundles)
	}

	pfs := chat.NewProtocolService(
		chat.NewEncryptionService(
			persistence,
			chat.DefaultEncryptionServiceConfig(instalationID),
		),
		addBundlesHandler,
	)

	a.SetPFS(pfs)

	return nil
}

// SetPFS sets the PFS service.
func (a *WhisperServiceAdapter) SetPFS(pfs *chat.ProtocolService) {
	a.pfs = pfs
}

// Subscribe subscribes to a public chat using the Whisper service.
func (a *WhisperServiceAdapter) Subscribe(
	ctx context.Context,
	in chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*protocol.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	filter := newFilter(a.keysManager)
	if err := updateFilterFromSubscribeOptions(filter, options); err != nil {
		return nil, err
	}

	filterID, err := a.shh.Subscribe(filter.ToWhisper())
	if err != nil {
		return nil, err
	}

	subWhisper := newWhisperSubscription(a.shh, filterID)
	sub := protocol.NewSubscription()

	go func() {
		defer subWhisper.Unsubscribe() // nolint: errcheck

		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				received, err := subWhisper.Messages()
				if err != nil {
					sub.Cancel(err)
					return
				}

				messages := a.handleMessages(received)
				for _, m := range messages {
					in <- m
				}
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}

func (a *WhisperServiceAdapter) handleMessages(received []*whisper.ReceivedMessage) []*protocol.Message {
	var messages []*protocol.Message

	for _, item := range received {
		message, err := a.decodeMessage(item)
		if err != nil {
			log.Printf("failed to decode message %#+x: %v", item.EnvelopeHash.Bytes(), err)
			continue
		}
		messages = append(messages, message)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Clock < messages[j].Clock
	})

	return messages
}

func (a *WhisperServiceAdapter) decodeMessage(message *whisper.ReceivedMessage) (*protocol.Message, error) {
	payload := message.Payload
	publicKey := message.SigToPubKey()
	hash := message.EnvelopeHash.Bytes()

	if a.pfs != nil {
		decryptedPayload, err := a.pfs.HandleMessage(
			a.keysManager.PrivateKey(),
			publicKey,
			payload,
			hash,
		)
		if err != nil {
			log.Printf("failed to handle message %#+x by PFS: %v", hash, err)
		} else {
			payload = decryptedPayload
		}
	}

	decoded, err := protocol.DecodeMessage(payload)
	if err != nil {
		return nil, err
	}
	decoded.ID = hash
	decoded.SigPubKey = publicKey

	return &decoded, nil
}

// Send sends a new message using the Whisper service.
func (a *WhisperServiceAdapter) Send(
	ctx context.Context,
	data []byte,
	options protocol.SendOptions,
) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if a.pfs != nil {
		var (
			encryptedData []byte
			err           error
		)

		if options.Recipient != nil {
			encryptedData, err = a.pfs.BuildDirectMessage(
				a.keysManager.PrivateKey(),
				options.Recipient,
				data,
			)
		} else {
			encryptedData, err = a.pfs.BuildPublicMessage(a.keysManager.PrivateKey(), data)
		}

		if err != nil {
			return nil, err
		}
		data = encryptedData
	}

	newMessage, err := newNewMessage(a.keysManager, data)
	if err != nil {
		return nil, err
	}
	if err := updateNewMessageFromSendOptions(newMessage, options); err != nil {
		return nil, err
	}

	// Only public Whisper API implements logic to send messages.
	shhAPI := whisper.NewPublicWhisperAPI(a.shh)
	return shhAPI.Post(ctx, newMessage.ToWhisper())
}

// Request requests messages from mail servers.
func (a *WhisperServiceAdapter) Request(ctx context.Context, options protocol.RequestOptions) error {
	if err := options.Validate(); err != nil {
		return err
	}

	// TODO: remove from here. MailServerEnode must be provided in the params.
	enode, err := a.selectAndAddMailServer()
	if err != nil {
		return err
	}

	keyID, err := a.keysManager.AddOrGetSymKeyFromPassword(MailServerPassword)
	if err != nil {
		return err
	}

	req, err := createShhextRequestMessagesParam(enode, keyID, options)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = a.requestMessages(ctx, req, true)

	log.Printf("[WhisperServiceAdapter::Request] took %s", time.Since(now))

	return err
}

func (a *WhisperServiceAdapter) selectAndAddMailServer() (string, error) {
	if a.selectedMailServerEnode != "" {
		return a.selectedMailServerEnode, nil
	}

	config := a.node.Config()
	enode := randomItem(config.ClusterConfig.TrustedMailServers)
	errCh := waitForPeerAsync(
		a.node.GethNode().Server(),
		enode,
		p2p.PeerEventTypeAdd,
		time.Second*5,
	)

	log.Printf("[WhisperServiceAdapter::selectAndAddMailServer] randomly selected %s node", enode)

	if err := a.node.AddPeer(enode); err != nil {
		return "", err
	}

	err := <-errCh
	if err != nil {
		err = fmt.Errorf("failed to add mail server %s: %v", enode, err)
	} else {
		a.selectedMailServerEnode = enode
	}

	return enode, err
}

func (a *WhisperServiceAdapter) requestMessages(ctx context.Context, req shhext.MessagesRequest, followCursor bool) (resp shhext.MessagesResponse, err error) {
	shhextService, err := a.node.ShhExtService()
	if err != nil {
		return
	}
	shhextAPI := shhext.NewPublicAPI(shhextService)

	log.Printf("[WhisperServiceAdapter::requestMessages] request for a chunk with %d messages", req.Limit)

	start := time.Now()
	resp, err = shhextAPI.RequestMessagesSync(shhext.RetryConfig{
		BaseTimeout: time.Second * 10,
		StepTimeout: time.Second,
		MaxRetries:  3,
	}, req)
	if err != nil {
		log.Printf("[WhisperServiceAdapter::requestMessages] failed with err: %v", err)
		return
	}

	log.Printf("[WhisperServiceAdapter::requestMessages] delivery of %d message took %v", req.Limit, time.Since(start))
	log.Printf("[WhisperServiceAdapter::requestMessages] response: %+v", resp)

	if resp.Error != nil {
		err = resp.Error
		return
	}
	if !followCursor || resp.Cursor == "" {
		return
	}

	req.Cursor = resp.Cursor
	log.Printf("[WhisperServiceAdapter::requestMessages] request messages with cursor %v", req.Cursor)
	return a.requestMessages(ctx, req, true)
}

// whisperSubscription encapsulates a Whisper filter.
type whisperSubscription struct {
	shh      *whisper.Whisper
	filterID string
}

// newWhisperSubscription returns a new whisperSubscription.
func newWhisperSubscription(shh *whisper.Whisper, filterID string) *whisperSubscription {
	return &whisperSubscription{
		shh:      shh,
		filterID: filterID,
	}
}

// Messages retrieves a list of messages for a given filter.
func (s whisperSubscription) Messages() ([]*whisper.ReceivedMessage, error) {
	f := s.shh.GetFilter(s.filterID)
	if f == nil {
		return nil, errors.New("filter does not exist")
	}
	messages := f.Retrieve()
	return messages, nil
}

// Unsubscribe removes the subscription.
func (s whisperSubscription) Unsubscribe() error {
	return s.shh.Unsubscribe(s.filterID)
}

type filter struct {
	*whisper.Filter
	keys keysManager
}

func newFilter(keys keysManager) *filter {
	return &filter{
		Filter: &whisper.Filter{
			PoW:      0,
			AllowP2P: true,
			Messages: whisper.NewMemoryMessageStore(),
		},
		keys: keys,
	}
}

func (f *filter) ToWhisper() *whisper.Filter {
	return f.Filter
}

func (f *filter) updateForPublicGroup(name string) error {
	topic, err := ToTopic(name)
	if err != nil {
		return err
	}
	f.Topics = append(f.Topics, topic[:])

	symKeyID, err := f.keys.AddOrGetSymKeyFromPassword(name)
	if err != nil {
		return err
	}
	symKey, err := f.keys.GetRawSymKey(symKeyID)
	if err != nil {
		return err
	}
	f.KeySym = symKey

	return nil
}

func (f *filter) updateForPrivate(name string, recipient *ecdsa.PublicKey) error {
	topic, err := ToTopic(name)
	if err != nil {
		return err
	}
	f.Topics = append(f.Topics, topic[:])

	f.KeyAsym = f.keys.PrivateKey()

	return nil
}

func updateFilterFromSubscribeOptions(f *filter, options protocol.SubscribeOptions) error {
	if options.Recipient != nil && options.ChatName != "" {
		return f.updateForPrivate(options.ChatName, options.Recipient)
	} else if options.ChatName != "" {
		return f.updateForPublicGroup(options.ChatName)
	} else {
		return errors.New("unrecognized options")
	}
}
