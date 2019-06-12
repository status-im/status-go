package adapters

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/status-im/status-console-client/protocol/v1"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/whisper/shhclient"

	whisper "github.com/status-im/whisper/whisperv6"
)

type whisperClientKeysManager struct {
	client     *shhclient.Client
	privateKey *ecdsa.PrivateKey

	passToSymMutex    sync.RWMutex
	passToSymKeyCache map[string]string
}

func (m *whisperClientKeysManager) PrivateKey() *ecdsa.PrivateKey {
	return m.privateKey
}

func (m *whisperClientKeysManager) AddOrGetKeyPair(priv *ecdsa.PrivateKey) (string, error) {
	// caching is handled in Whisper
	return m.client.AddPrivateKey(context.Background(), crypto.FromECDSA(priv))
}

func (m *whisperClientKeysManager) AddOrGetSymKeyFromPassword(password string) (string, error) {
	m.passToSymMutex.Lock()
	defer m.passToSymMutex.Unlock()

	if val, ok := m.passToSymKeyCache[password]; ok {
		return val, nil
	}

	id, err := m.client.GenerateSymmetricKeyFromPassword(context.Background(), password)
	if err != nil {
		return "", err
	}

	m.passToSymKeyCache[password] = id

	return id, nil
}

func (m *whisperClientKeysManager) GetRawSymKey(id string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

// WhisperClientAdapter is an adapter for Whisper client
// which implements Protocol interface. It requires an RPC client
// which can use various transports like HTTP, IPC or in-proc.
type WhisperClientAdapter struct {
	rpcClient   *rpc.Client
	shhClient   *shhclient.Client
	keysManager *whisperClientKeysManager

	mailServerEnodes        []string
	selectedMailServerEnode string
}

// WhisperClientAdapter must implement Protocol interface.
var _ protocol.Protocol = (*WhisperClientAdapter)(nil)

// NewWhisperClientAdapter returns a new WhisperClientAdapter.
func NewWhisperClientAdapter(c *rpc.Client, privateKey *ecdsa.PrivateKey, mailServers []string) *WhisperClientAdapter {
	client := shhclient.NewClient(c)

	return &WhisperClientAdapter{
		rpcClient:        c,
		shhClient:        client,
		mailServerEnodes: mailServers,
		keysManager: &whisperClientKeysManager{
			client:     client,
			privateKey: privateKey,
		},
	}
}

// Subscribe subscribes to a public channel.
// in channel is used to receive messages.
func (a *WhisperClientAdapter) Subscribe(
	ctx context.Context,
	in chan<- *protocol.Message,
	options protocol.SubscribeOptions,
) (*protocol.Subscription, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	criteria := newCriteria(a.keysManager)
	if err := updateCriteriaFromSubscribeOptions(criteria, options); err != nil {
		return nil, err
	}

	return a.subscribeMessages(ctx, criteria.ToWhisper(), in)
}

func (a *WhisperClientAdapter) subscribeMessages(
	ctx context.Context,
	crit whisper.Criteria,
	in chan<- *protocol.Message,
) (*protocol.Subscription, error) {
	messages := make(chan *whisper.Message)
	shhSub, err := a.shhClient.SubscribeMessages(ctx, crit, messages)
	if err != nil {
		return nil, err
	}

	sub := protocol.NewSubscription()

	go func() {
		defer shhSub.Unsubscribe()

		for {
			select {
			case raw := <-messages:
				m, err := protocol.DecodeMessage(raw.Payload)
				if err != nil {
					log.Printf("failed to decode message: %v", err)
					break
				}

				sigPubKey, err := crypto.UnmarshalPubkey(raw.Sig)
				if err != nil {
					log.Printf("failed to get a signature: %v", err)
					break
				}
				m.SigPubKey = sigPubKey

				in <- &m
			case err := <-shhSub.Err():
				sub.Cancel(err)
				return
			case <-sub.Done():
				return
			}
		}
	}()

	return sub, nil
}

// Send sends a new message to a public chat.
// Identity is required to sign a message as only signed messages
// are accepted and displayed.
func (a *WhisperClientAdapter) Send(
	ctx context.Context,
	data []byte,
	options protocol.SendOptions,
) ([]byte, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	newMessage, err := newNewMessage(a.keysManager, data)
	if err != nil {
		return nil, err
	}
	if err := updateNewMessageFromSendOptions(newMessage, options); err != nil {
		return nil, err
	}

	hash, err := a.shhClient.Post(ctx, newMessage.ToWhisper())
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(hash)
}

// Request sends a request to MailServer for historic messages.
func (a *WhisperClientAdapter) Request(ctx context.Context, params protocol.RequestOptions) error {
	enode, err := a.selectAndAddMailServer(ctx)
	if err != nil {
		return err
	}
	return a.requestMessages(ctx, enode, params)
}

func (a *WhisperClientAdapter) selectAndAddMailServer(ctx context.Context) (string, error) {
	if a.selectedMailServerEnode != "" {
		return a.selectedMailServerEnode, nil
	}

	enode := randomItem(a.mailServerEnodes)

	if err := a.rpcClient.CallContext(ctx, nil, "admin_addPeer", enode); err != nil {
		return "", err
	}

	// Adding peer is asynchronous operation so we need to retry a few times.
	retries := 0
	for {
		err := a.shhClient.MarkTrustedPeer(ctx, enode)
		if ctx.Err() == context.Canceled {
			log.Printf("requesting public messages canceled")
			return "", err
		}
		if err == nil {
			break
		}
		if retries < 3 {
			retries++
			<-time.After(time.Second)
		} else {
			return "", fmt.Errorf("failed to mark peer as trusted: %v", err)
		}
	}

	a.selectedMailServerEnode = enode

	return enode, nil
}

func (a *WhisperClientAdapter) requestMessages(ctx context.Context, enode string, options protocol.RequestOptions) error {
	log.Printf("requesting messages from node %s", enode)

	mailSymKeyID, err := a.keysManager.AddOrGetSymKeyFromPassword(MailServerPassword)
	if err != nil {
		return err
	}

	arg, err := createShhextRequestMessagesParam(enode, mailSymKeyID, options)
	if err != nil {
		return err
	}

	return a.rpcClient.CallContext(ctx, nil, "shhext_requestMessages", arg)
}

type criteria struct {
	whisper.Criteria
	keys keysManager
}

func newCriteria(keys keysManager) *criteria {
	return &criteria{
		Criteria: whisper.Criteria{
			MinPow:   WhisperPoW,
			AllowP2P: true, // messages from mail server are direct p2p messages
		},
		keys: keys,
	}
}

func (c *criteria) ToWhisper() whisper.Criteria {
	return c.Criteria
}

func (c *criteria) updateForPublicGroup(name string) error {
	topic, err := ToTopic(name)
	if err != nil {
		return err
	}
	c.Topics = append(c.Topics, topic)

	symKeyID, err := c.keys.AddOrGetSymKeyFromPassword(name)
	if err != nil {
		return err
	}
	c.SymKeyID = symKeyID

	return nil
}

func (c *criteria) updateForPrivate(name string, recipient *ecdsa.PublicKey) error {
	topic, err := ToTopic(name)
	if err != nil {
		return err
	}
	c.Topics = append(c.Topics, topic)

	keyID, err := c.keys.AddOrGetKeyPair(c.keys.PrivateKey())
	if err != nil {
		return err
	}
	c.PrivateKeyID = keyID

	return nil
}

func updateCriteriaFromSubscribeOptions(c *criteria, options protocol.SubscribeOptions) error {
	if options.Recipient != nil && options.ChatName != "" {
		return c.updateForPrivate(options.ChatName, options.Recipient)
	} else if options.ChatName != "" {
		return c.updateForPublicGroup(options.ChatName)
	} else {
		return errors.New("unrecognized options")
	}
}
