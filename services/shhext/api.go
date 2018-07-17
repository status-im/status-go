package shhext

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/services/shhext/chat"
)

const (
	// defaultWorkTime is a work time reported in messages sent to MailServer nodes.
	defaultWorkTime = 5
	// defaultRequestTimeout is the default request timeout in seconds
	defaultRequestTimeout = 10
)

var (
	// ErrInvalidMailServerPeer is returned when it fails to parse enode from params.
	ErrInvalidMailServerPeer = errors.New("invalid mailServerPeer value")
	// ErrInvalidSymKeyID is returned when it fails to get a symmetric key.
	ErrInvalidSymKeyID = errors.New("invalid symKeyID value")
	// ErrInvalidPublicKey is returned when public key can't be extracted
	// from MailServer's nodeID.
	ErrInvalidPublicKey = errors.New("can't extract public key")
)

// -----
// PAYLOADS
// -----

// MessagesRequest is a payload send to a MailServer to get messages.
type MessagesRequest struct {
	// MailServerPeer is MailServer's enode address.
	MailServerPeer string `json:"mailServerPeer"`

	// From is a lower bound of time range (optional).
	// Default is 24 hours back from now.
	From uint32 `json:"from"`

	// To is a upper bound of time range (optional).
	// Default is now.
	To uint32 `json:"to"`

	// Limit determines the number of messages sent by the mail server
	// for the current paginated request
	Limit uint32 `json:"limit"`

	// Cursor is used as starting point for paginated requests
	Cursor string `json:"cursor"`

	// Topic is a regular Whisper topic.
	Topic whisper.TopicType `json:"topic"`

	// SymKeyID is an ID of a symmetric key to authenticate to MailServer.
	// It's derived from MailServer password.
	//
	// It's also possible to authenticate request with MailServerPeer
	// public key.
	SymKeyID string `json:"symKeyID"`

	// Timeout is the time to live of the request specified in seconds.
	// Default is 10 seconds
	Timeout time.Duration `json:"timeout"`
}

func (r *MessagesRequest) setDefaults(now time.Time) {
	// set From and To defaults
	if r.To == 0 {
		r.To = uint32(now.UTC().Unix())
	}

	if r.From == 0 {
		oneDay := uint32(86400) // -24 hours
		if r.To < oneDay {
			r.From = 0
		} else {
			r.From = r.To - oneDay
		}
	}

	if r.Timeout == 0 {
		r.Timeout = defaultRequestTimeout
	}
}

// -----
// PUBLIC API
// -----

// PublicAPI extends whisper public API.
type PublicAPI struct {
	service   *Service
	publicAPI *whisper.PublicWhisperAPI
	log       log.Logger
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		service:   s,
		publicAPI: whisper.NewPublicWhisperAPI(s.w),
		log:       log.New("package", "status-go/services/sshext.PublicAPI"),
	}
}

// Post shamelessly copied from whisper codebase with slight modifications.
func (api *PublicAPI) Post(ctx context.Context, req whisper.NewMessage) (hash hexutil.Bytes, err error) {
	hash, err = api.publicAPI.Post(ctx, req)
	if err == nil {
		var envHash common.Hash
		copy(envHash[:], hash[:]) // slice can't be used as key
		api.service.tracker.Add(envHash)
	}
	return hash, err
}

// RequestMessages sends a request for historic messages to a MailServer.
func (api *PublicAPI) RequestMessages(_ context.Context, r MessagesRequest) (hexutil.Bytes, error) {
	api.log.Info("RequestMessages", "request", r)
	shh := api.service.w
	now := api.service.w.GetCurrentTime()
	r.setDefaults(now)

	if r.From > r.To {
		return nil, fmt.Errorf("Query range is invalid: from > to (%d > %d)", r.From, r.To)
	}

	var err error

	mailServerNode, err := discover.ParseNode(r.MailServerPeer)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", ErrInvalidMailServerPeer, err)
	}

	var (
		symKey    []byte
		publicKey *ecdsa.PublicKey
	)

	if r.SymKeyID != "" {
		symKey, err = shh.GetSymKey(r.SymKeyID)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", ErrInvalidSymKeyID, err)
		}
	} else {
		publicKey, err = mailServerNode.ID.Pubkey()
		if err != nil {
			return nil, fmt.Errorf("%v: %v", ErrInvalidPublicKey, err)
		}
	}

	envelope, err := makeEnvelop(
		makePayload(r),
		symKey,
		publicKey,
		api.service.nodeID,
		shh.MinPow(),
		now,
	)
	if err != nil {
		return nil, err
	}

	hash := envelope.Hash()
	if err := shh.RequestHistoricMessages(mailServerNode.ID[:], envelope); err != nil {
		return nil, err
	}

	api.service.tracker.AddRequest(hash, time.After(r.Timeout*time.Second))

	return hash[:], nil
}

// GetNewFilterMessages is a prototype method with deduplication
func (api *PublicAPI) GetNewFilterMessages(filterID string) ([]*whisper.Message, error) {
	msgs, err := api.publicAPI.GetFilterMessages(filterID)
	if err != nil {
		return nil, err
	}

	dedupMessages := api.service.deduplicator.Deduplicate(msgs)

	// Attempt to decrypt message, otherwise leave unchanged
	for _, msg := range dedupMessages {
		var privateKey *ecdsa.PrivateKey
		var publicKey *ecdsa.PublicKey

		// Msg.Dst is empty is a public message, nothing to do
		if msg.Dst != nil {
			// There's probably a better way to do this
			keyBytes, err := hexutil.Bytes(msg.Dst).MarshalText()
			if err != nil {
				return nil, err
			}

			privateKey, err = api.service.w.GetPrivateKey(string(keyBytes))
			if err != nil {
				return nil, err
			}

			// This needs to be pushed down in the protocol message
			publicKey, err = crypto.UnmarshalPubkey(msg.Sig)
			if err != nil {
				return nil, err
			}
		}

		payload, err := api.service.protocol.HandleMessage(privateKey, publicKey, msg.Payload)

		// Ignore errors for now
		if err == nil {
			msg.Payload = payload
		}

	}

	return dedupMessages, nil
}

// ConfirmMessagesProcessed is a method to confirm that messages was consumed by
// the client side.
func (api *PublicAPI) ConfirmMessagesProcessed(messages []*whisper.Message) error {
	return api.service.deduplicator.AddMessages(messages)
}

func (api *PublicAPI) SendPublicMessage(ctx context.Context, msg chat.SendPublicMessageRPC) (hexutil.Bytes, error) {
	fmt.Printf("%s\n", msg.Sig)
	privateKey, err := api.service.w.GetPrivateKey(msg.Sig)
	fmt.Printf("%s\n", privateKey)
	if err != nil {
		return nil, err
	}

	// This is transport layer agnostic
	protocolMessage, err := api.service.protocol.BuildPublicMessage(privateKey, msg.Payload)
	if err != nil {
		return nil, err
	}

	symKeyID, err := api.service.w.AddSymKeyFromPassword(msg.Chat)

	// Enrich with transport layer info
	whisperMessage := chat.PublicMessageToWhisper(&msg, protocolMessage)
	if err != nil {
		return nil, err
	}

	whisperMessage.SymKeyID = symKeyID

	// And dispatch
	return api.Post(ctx, *whisperMessage)
}

func (api *PublicAPI) SendDirectMessage(ctx context.Context, msg chat.SendDirectMessageRPC) (hexutil.Bytes, error) {

	// To be completely agnostic from whisper we should not be using whisper to store the key
	privateKey, err := api.service.w.GetPrivateKey(msg.Sig)
	if err != nil {
		return nil, err
	}

	publicKey, err := crypto.UnmarshalPubkey(msg.PubKey)
	if err != nil {
		return nil, err
	}

	// This is transport layer agnostic
	protocolMessage, err := api.service.protocol.BuildDirectMessage(privateKey, publicKey, msg.Payload)
	if err != nil {
		return nil, err
	}

	// Enrich with transport layer info
	whisperMessage := chat.DirectMessageToWhisper(&msg, protocolMessage)

	// And dispatch
	return api.Post(ctx, *whisperMessage)
}

// -----
// HELPER
// -----

// makeEnvelop makes an envelop for a historic messages request.
// Symmetric key is used to authenticate to MailServer.
// PK is the current node ID.
func makeEnvelop(
	payload []byte,
	symKey []byte,
	publicKey *ecdsa.PublicKey,
	nodeID *ecdsa.PrivateKey,
	pow float64,
	now time.Time,
) (*whisper.Envelope, error) {
	params := whisper.MessageParams{
		PoW:      pow,
		Payload:  payload,
		WorkTime: defaultWorkTime,
		Src:      nodeID,
	}
	// Either symKey or public key is required.
	// This condition is verified in `message.Wrap()` method.
	if len(symKey) > 0 {
		params.KeySym = symKey
	} else if publicKey != nil {
		params.Dst = publicKey
	}
	message, err := whisper.NewSentMessage(&params)
	if err != nil {
		return nil, err
	}
	return message.Wrap(&params, now)
}

// makePayload makes a specific payload for MailServer to request historic messages.
func makePayload(r MessagesRequest) []byte {
	// Payload format:
	// 4  bytes for lower
	// 4  bytes for upper
	// 64 bytes for the bloom filter
	// 4  bytes for limit
	// 36 bytes for the cursor. optional.
	data := make([]byte, 12+whisper.BloomFilterSize)

	// from
	binary.BigEndian.PutUint32(data, r.From)
	// to
	binary.BigEndian.PutUint32(data[4:], r.To)
	// bloom
	copy(data[8:], whisper.TopicToBloom(r.Topic))
	// limit
	binary.BigEndian.PutUint32(data[8+whisper.BloomFilterSize:], r.Limit)

	// cursor is the key of an envelope in leveldb.
	// it's 36 bytes. 4 bytes for the timestamp + 32 bytes for the envelope hash
	expectedCursorSize := common.HashLength + 4
	cursorBytes, err := hex.DecodeString(r.Cursor)
	if err != nil || len(cursorBytes) != expectedCursorSize {
		return data
	}

	return append(data, cursorBytes...)
}
