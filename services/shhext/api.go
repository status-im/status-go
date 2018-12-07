package shhext

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/services/shhext/chat"
	"github.com/status-im/status-go/services/shhext/mailservers"
	whisper "github.com/status-im/whisper/whisperv6"
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
	// ErrPFSNotEnabled is returned when an endpoint PFS only is called but
	// PFS is disabled
	ErrPFSNotEnabled = errors.New("pfs not enabled")
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
	// DEPRECATED
	Topic whisper.TopicType `json:"topic"`

	// Topics is a list of Whisper topics.
	Topics []whisper.TopicType `json:"topics"`

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

// MessagesRequestPayload is a payload sent to the Mail Server.
type MessagesRequestPayload struct {
	// Lower is a lower bound of time range for which messages are requested.
	Lower uint32
	// Upper is a lower bound of time range for which messages are requested.
	Upper uint32
	// Bloom is a bloom filter to filter envelopes.
	Bloom []byte
	// Limit is the max number of envelopes to return.
	Limit uint32
	// Cursor is used for pagination of the results.
	Cursor []byte
	// Batch set to true indicates that the client supports batched response.
	Batch bool
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

func (api *PublicAPI) getPeer(rawurl string) (*enode.Node, error) {
	if len(rawurl) == 0 {
		return mailservers.GetFirstConnected(api.service.server, api.service.peerStore)
	}
	return enode.ParseV4(rawurl)
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

	mailServerNode, err := api.getPeer(r.MailServerPeer)
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
		publicKey = mailServerNode.Pubkey()
	}

	payload, err := makePayload(r)
	if err != nil {
		return nil, err
	}

	envelope, err := makeEnvelop(
		payload,
		symKey,
		publicKey,
		api.service.nodeID,
		shh.MinPow(),
		now,
	)
	if err != nil {
		return nil, err
	}

	if err := shh.RequestHistoricMessagesWithTimeout(mailServerNode.ID().Bytes(), envelope, r.Timeout*time.Second); err != nil {
		return nil, err
	}
	hash := envelope.Hash()
	return hash[:], nil
}

// GetNewFilterMessages is a prototype method with deduplication
func (api *PublicAPI) GetNewFilterMessages(filterID string) ([]*whisper.Message, error) {
	msgs, err := api.publicAPI.GetFilterMessages(filterID)
	if err != nil {
		return nil, err
	}

	dedupMessages := api.service.deduplicator.Deduplicate(msgs)

	if api.service.pfsEnabled {
		// Attempt to decrypt message, otherwise leave unchanged
		for _, msg := range dedupMessages {

			if err := api.processPFSMessage(msg); err != nil {
				return nil, err
			}
		}
	}

	return dedupMessages, nil
}

// ConfirmMessagesProcessed is a method to confirm that messages was consumed by
// the client side.
func (api *PublicAPI) ConfirmMessagesProcessed(messages []*whisper.Message) error {
	return api.service.deduplicator.AddMessages(messages)
}

// SendPublicMessage sends a public chat message to the underlying transport
func (api *PublicAPI) SendPublicMessage(ctx context.Context, msg chat.SendPublicMessageRPC) (hexutil.Bytes, error) {
	privateKey, err := api.service.w.GetPrivateKey(msg.Sig)
	if err != nil {
		return nil, err
	}

	// This is transport layer agnostic
	protocolMessage, err := api.service.protocol.BuildPublicMessage(privateKey, msg.Payload)
	if err != nil {
		return nil, err
	}

	symKeyID, err := api.service.w.AddSymKeyFromPassword(msg.Chat)
	if err != nil {
		return nil, err
	}

	// Enrich with transport layer info
	whisperMessage := chat.PublicMessageToWhisper(msg, protocolMessage)
	whisperMessage.SymKeyID = symKeyID

	// And dispatch
	return api.Post(ctx, whisperMessage)
}

// SendDirectMessage sends a 1:1 chat message to the underlying transport
func (api *PublicAPI) SendDirectMessage(ctx context.Context, msg chat.SendDirectMessageRPC) ([]hexutil.Bytes, error) {
	if !api.service.pfsEnabled {
		return nil, ErrPFSNotEnabled
	}
	// To be completely agnostic from whisper we should not be using whisper to store the key
	privateKey, err := api.service.w.GetPrivateKey(msg.Sig)
	if err != nil {
		return nil, err
	}

	publicKey, err := crypto.UnmarshalPubkey(msg.PubKey)
	if err != nil {
		return nil, err
	}

	// This is transport layer-agnostic
	protocolMessages, err := api.service.protocol.BuildDirectMessage(privateKey, msg.Payload, publicKey)
	if err != nil {
		return nil, err
	}

	var response []hexutil.Bytes

	for key, message := range protocolMessages {
		msg.PubKey = crypto.FromECDSAPub(key)
		// Enrich with transport layer info
		whisperMessage := chat.DirectMessageToWhisper(msg, message)

		// And dispatch
		hash, err := api.Post(ctx, whisperMessage)
		if err != nil {
			return nil, err
		}
		response = append(response, hash)

	}
	return response, nil
}

// SendPairingMessage sends a 1:1 chat message to our own devices to initiate a pairing session
func (api *PublicAPI) SendPairingMessage(ctx context.Context, msg chat.SendDirectMessageRPC) ([]hexutil.Bytes, error) {
	if !api.service.pfsEnabled {
		return nil, ErrPFSNotEnabled
	}
	// To be completely agnostic from whisper we should not be using whisper to store the key
	privateKey, err := api.service.w.GetPrivateKey(msg.Sig)
	if err != nil {
		return nil, err
	}

	msg.PubKey = crypto.FromECDSAPub(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	protocolMessage, err := api.service.protocol.BuildPairingMessage(privateKey, msg.Payload)
	if err != nil {
		return nil, err
	}

	var response []hexutil.Bytes

	// Enrich with transport layer info
	whisperMessage := chat.DirectMessageToWhisper(msg, protocolMessage)

	// And dispatch
	hash, err := api.Post(ctx, whisperMessage)
	if err != nil {
		return nil, err
	}
	response = append(response, hash)

	return response, nil
}

// SendGroupMessage sends a group messag chat message to the underlying transport
func (api *PublicAPI) SendGroupMessage(ctx context.Context, msg chat.SendGroupMessageRPC) ([]hexutil.Bytes, error) {
	if !api.service.pfsEnabled {
		return nil, ErrPFSNotEnabled
	}

	// To be completely agnostic from whisper we should not be using whisper to store the key
	privateKey, err := api.service.w.GetPrivateKey(msg.Sig)
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
	protocolMessages, err := api.service.protocol.BuildDirectMessage(privateKey, msg.Payload, keys...)
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
		hash, err := api.Post(ctx, whisperMessage)
		if err != nil {
			return nil, err
		}
		response = append(response, hash)

	}
	return response, nil
}

func (api *PublicAPI) processPFSMessage(msg *whisper.Message) error {
	var privateKey *ecdsa.PrivateKey
	var publicKey *ecdsa.PublicKey

	// Msg.Dst is empty is a public message, nothing to do
	if msg.Dst != nil {
		// There's probably a better way to do this
		keyBytes, err := hexutil.Bytes(msg.Dst).MarshalText()
		if err != nil {
			return err
		}

		privateKey, err = api.service.w.GetPrivateKey(string(keyBytes))
		if err != nil {
			return err
		}

		// This needs to be pushed down in the protocol message
		publicKey, err = crypto.UnmarshalPubkey(msg.Sig)
		if err != nil {
			return err
		}
	}

	response, err := api.service.protocol.HandleMessage(privateKey, publicKey, msg.Payload)

	// Notify that someone tried to contact us using an invalid bundle
	if err == chat.ErrDeviceNotFound && privateKey.PublicKey != *publicKey {
		api.log.Warn("Device not found, sending signal", "err", err)
		keyString := fmt.Sprintf("0x%x", crypto.FromECDSAPub(publicKey))
		handler := EnvelopeSignalHandler{}
		handler.DecryptMessageFailed(keyString)
		return nil
	} else if err != nil {
		// Ignore errors for now as those might be non-pfs messages
		api.log.Error("Failed handling message with error", "err", err)
		return nil
	}

	// Add unencrypted payload
	msg.Payload = response

	return nil
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
func makePayload(r MessagesRequest) ([]byte, error) {
	expectedCursorSize := common.HashLength + 4
	cursor, err := hex.DecodeString(r.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %v", err)
	}
	if len(cursor) != expectedCursorSize {
		return nil, fmt.Errorf("invalid cursor size: expected %d but got %d", expectedCursorSize, len(cursor))
	}

	payload := MessagesRequestPayload{
		Lower:  r.From,
		Upper:  r.To,
		Bloom:  createBloomFilter(r),
		Limit:  r.Limit,
		Cursor: cursor,
		// Client must tell the MailServer if it supports batch responses.
		// This can be removed in the future.
		Batch: true,
	}

	return rlp.EncodeToBytes(payload)
}

func createBloomFilter(r MessagesRequest) []byte {
	if len(r.Topics) > 0 {
		return topicsToBloom(r.Topics...)
	}

	return whisper.TopicToBloom(r.Topic)
}

func topicsToBloom(topics ...whisper.TopicType) []byte {
	i := new(big.Int)
	for _, topic := range topics {
		bloom := whisper.TopicToBloom(topic)
		i.Or(i, new(big.Int).SetBytes(bloom[:]))
	}

	combined := make([]byte, whisper.BloomFilterSize)
	data := i.Bytes()
	copy(combined[whisper.BloomFilterSize-len(data):], data[:])

	return combined
}
