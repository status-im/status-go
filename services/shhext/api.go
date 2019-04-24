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
	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/mailserver"
	"github.com/status-im/status-go/services/shhext/chat"
	"github.com/status-im/status-go/services/shhext/dedup"
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

// MessagesRequest is a RequestMessages() request payload.
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
	SymKeyID string `json:"symKeyID"`

	// Timeout is the time to live of the request specified in seconds.
	// Default is 10 seconds
	Timeout time.Duration `json:"timeout"`

	// Force ensures that requests will bypass enforced delay.
	Force bool `json:"force"`
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

// MessagesResponse is a response for shhext_requestMessages2 method.
type MessagesResponse struct {
	// Cursor from the response can be used to retrieve more messages
	// for the previous request.
	Cursor string `json:"cursor"`

	// Error indicates that something wrong happened when sending messages
	// to the requester.
	Error error `json:"error"`
}

// SyncMessagesRequest is a SyncMessages() request payload.
type SyncMessagesRequest struct {
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

	// Topics is a list of Whisper topics.
	// If empty, a full bloom filter will be used.
	Topics []whisper.TopicType `json:"topics"`
}

// SyncMessagesResponse is a response from the mail server
// to which SyncMessagesRequest was sent.
type SyncMessagesResponse struct {
	// Cursor from the response can be used to retrieve more messages
	// for the previous request.
	Cursor string `json:"cursor"`

	// Error indicates that something wrong happened when sending messages
	// to the requester.
	Error string `json:"error"`
}

// InitiateHistoryRequestParams type for initiating history requests from a peer.
type InitiateHistoryRequestParams struct {
	Peer     string
	SymKeyID string
	Requests []TopicRequest
	Force    bool
	Timeout  time.Duration
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
func (api *PublicAPI) Post(ctx context.Context, req whisper.NewMessage) (hexutil.Bytes, error) {
	hexID, err := api.publicAPI.Post(ctx, req)
	if err == nil {
		api.service.envelopesMonitor.Add(common.BytesToHash(hexID), req)
	} else {
		return nil, err
	}
	mID := messageID(req)
	return mID[:], err
}

func (api *PublicAPI) getPeer(rawurl string) (*enode.Node, error) {
	if len(rawurl) == 0 {
		return mailservers.GetFirstConnected(api.service.server, api.service.peerStore)
	}
	return enode.ParseV4(rawurl)
}

// RetryConfig specifies configuration for retries with timeout and max amount of retries.
type RetryConfig struct {
	BaseTimeout time.Duration
	// StepTimeout defines duration increase per each retry.
	StepTimeout time.Duration
	MaxRetries  int
}

// RequestMessagesSync repeats MessagesRequest using configuration in retry conf.
func (api *PublicAPI) RequestMessagesSync(conf RetryConfig, r MessagesRequest) (MessagesResponse, error) {
	var resp MessagesResponse

	shh := api.service.w
	events := make(chan whisper.EnvelopeEvent, 10)
	sub := shh.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()
	var (
		requestID hexutil.Bytes
		err       error
		retries   int
	)
	for retries <= conf.MaxRetries {
		r.Timeout = conf.BaseTimeout + conf.StepTimeout*time.Duration(retries)
		// FIXME this weird conversion is required because MessagesRequest expects seconds but defines time.Duration
		r.Timeout = time.Duration(int(r.Timeout.Seconds()))
		requestID, err = api.RequestMessages(context.Background(), r)
		if err != nil {
			return resp, err
		}
		mailServerResp, err := waitForExpiredOrCompleted(common.BytesToHash(requestID), events)
		if err == nil {
			resp.Cursor = hex.EncodeToString(mailServerResp.Cursor)
			resp.Error = mailServerResp.Error
			return resp, nil
		}
		retries++
		api.log.Error("[RequestMessagesSync] failed", "err", err, "retries", retries)
	}
	return resp, fmt.Errorf("failed to request messages after %d retries", retries)
}

func waitForExpiredOrCompleted(requestID common.Hash, events chan whisper.EnvelopeEvent) (*whisper.MailServerResponse, error) {
	for {
		ev := <-events
		if ev.Hash != requestID {
			continue
		}
		switch ev.Event {
		case whisper.EventMailServerRequestCompleted:
			data, ok := ev.Data.(*whisper.MailServerResponse)
			if ok {
				return data, nil
			}
			return nil, errors.New("invalid event data type")
		case whisper.EventMailServerRequestExpired:
			return nil, errors.New("request expired")
		}
	}
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

	payload, err := makeMessagesRequestPayload(r)
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
	hash := envelope.Hash()

	if !r.Force {
		err = api.service.requestsRegistry.Register(hash, r.Topics)
		if err != nil {
			return nil, err
		}
	}

	if err := shh.RequestHistoricMessagesWithTimeout(mailServerNode.ID().Bytes(), envelope, r.Timeout*time.Second); err != nil {
		if !r.Force {
			api.service.requestsRegistry.Unregister(hash)
		}
		return nil, err
	}

	return hash[:], nil
}

// createSyncMailRequest creates SyncMailRequest. It uses a full bloom filter
// if no topics are given.
func createSyncMailRequest(r SyncMessagesRequest) (whisper.SyncMailRequest, error) {
	var bloom []byte
	if len(r.Topics) > 0 {
		bloom = topicsToBloom(r.Topics...)
	} else {
		bloom = whisper.MakeFullNodeBloom()
	}

	cursor, err := hex.DecodeString(r.Cursor)
	if err != nil {
		return whisper.SyncMailRequest{}, err
	}

	return whisper.SyncMailRequest{
		Lower:  r.From,
		Upper:  r.To,
		Bloom:  bloom,
		Limit:  r.Limit,
		Cursor: cursor,
	}, nil
}

func createSyncMessagesResponse(r whisper.SyncEventResponse) SyncMessagesResponse {
	return SyncMessagesResponse{
		Cursor: hex.EncodeToString(r.Cursor),
		Error:  r.Error,
	}
}

// SyncMessages sends a request to a given MailServerPeer to sync historic messages.
// MailServerPeers needs to be added as a trusted peer first.
func (api *PublicAPI) SyncMessages(ctx context.Context, r SyncMessagesRequest) (SyncMessagesResponse, error) {
	var response SyncMessagesResponse

	mailServerEnode, err := enode.ParseV4(r.MailServerPeer)
	if err != nil {
		return response, fmt.Errorf("invalid MailServerPeer: %v", err)
	}

	request, err := createSyncMailRequest(r)
	if err != nil {
		return response, fmt.Errorf("failed to create a sync mail request: %v", err)
	}

	if err := api.service.w.SyncMessages(mailServerEnode.ID().Bytes(), request); err != nil {
		return response, fmt.Errorf("failed to send a sync request: %v", err)
	}

	// Wait for the response which is received asynchronously as a p2p packet.
	// This packet handler will send an event which contains the response payload.
	events := make(chan whisper.EnvelopeEvent)
	sub := api.service.w.SubscribeEnvelopeEvents(events)
	defer sub.Unsubscribe()

	for {
		select {
		case event := <-events:
			if event.Event != whisper.EventMailServerSyncFinished {
				continue
			}

			log.Info("received EventMailServerSyncFinished event", "data", event.Data)

			if resp, ok := event.Data.(whisper.SyncEventResponse); ok {
				return createSyncMessagesResponse(resp), nil
			}
			return response, fmt.Errorf("did not understand the response event data")
		case <-ctx.Done():
			return response, ctx.Err()
		}
	}
}

// GetNewFilterMessages is a prototype method with deduplication
func (api *PublicAPI) GetNewFilterMessages(filterID string) ([]dedup.DeduplicateMessage, error) {
	msgs, err := api.publicAPI.GetFilterMessages(filterID)
	if err != nil {
		return nil, err
	}

	dedupMessages := api.service.deduplicator.Deduplicate(msgs)

	if api.service.pfsEnabled {
		// Attempt to decrypt message, otherwise leave unchanged
		for _, dedupMessage := range dedupMessages {

			if err := api.processPFSMessage(dedupMessage); err != nil {
				return nil, err
			}
		}
	}

	return dedupMessages, nil
}

// ConfirmMessagesProcessed is a method to confirm that messages was consumed by
// the client side.
func (api *PublicAPI) ConfirmMessagesProcessed(messages []*whisper.Message) error {
	for _, msg := range messages {
		err := api.service.historyUpdates.UpdateTopicHistory(msg.Topic, time.Unix(int64(msg.Timestamp), 0))
		if err != nil {
			return err
		}
	}
	return api.service.deduplicator.AddMessages(messages)
}

// ConfirmMessagesProcessedByID is a method to confirm that messages was consumed by
// the client side.
func (api *PublicAPI) ConfirmMessagesProcessedByID(messageIDs [][]byte) error {
	if err := api.service.protocol.ConfirmMessagesProcessed(messageIDs); err != nil {
		return err
	}

	return api.service.deduplicator.AddMessageByID(messageIDs)
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
func (api *PublicAPI) SendDirectMessage(ctx context.Context, msg chat.SendDirectMessageRPC) (hexutil.Bytes, error) {
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
	var protocolMessage []byte

	if msg.DH {
		protocolMessage, err = api.service.protocol.BuildDHMessage(privateKey, &privateKey.PublicKey, msg.Payload)
	} else {
		protocolMessage, err = api.service.protocol.BuildDirectMessage(privateKey, publicKey, msg.Payload)
	}

	if err != nil {
		return nil, err
	}

	// Enrich with transport layer info
	whisperMessage := chat.DirectMessageToWhisper(msg, protocolMessage)

	// And dispatch
	return api.Post(ctx, whisperMessage)
}

func (api *PublicAPI) requestMessagesUsingPayload(request db.HistoryRequest, peer, symkeyID string, payload []byte, force bool, timeout time.Duration, topics []whisper.TopicType) (hash common.Hash, err error) {
	shh := api.service.w
	now := api.service.w.GetCurrentTime()

	mailServerNode, err := api.getPeer(peer)
	if err != nil {
		return hash, fmt.Errorf("%v: %v", ErrInvalidMailServerPeer, err)
	}

	var (
		symKey    []byte
		publicKey *ecdsa.PublicKey
	)

	if symkeyID != "" {
		symKey, err = shh.GetSymKey(symkeyID)
		if err != nil {
			return hash, fmt.Errorf("%v: %v", ErrInvalidSymKeyID, err)
		}
	} else {
		publicKey = mailServerNode.Pubkey()
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
		return hash, err
	}
	hash = envelope.Hash()

	request.ID = hash
	err = request.Save()
	if err != nil {
		return hash, err
	}

	if !force {
		err = api.service.requestsRegistry.Register(hash, topics)
		if err != nil {
			return hash, err
		}
	}

	if err := shh.RequestHistoricMessagesWithTimeout(mailServerNode.ID().Bytes(), envelope, timeout); err != nil {
		err = request.Delete()
		if err != nil {
			return hash, err
		}
		if !force {
			api.service.requestsRegistry.Unregister(hash)
		}
		return hash, err
	}

	return hash, nil

}

// InitiateHistoryRequests is a stateful API for initiating history request for each topic.
// Caller of this method needs to define only two parameters per each TopicRequest:
// - Topic
// - Duration in nanoseconds. Will be used to determine starting time for history request.
// After that status-go will guarantee that request for this topic and date will be performed.
func (api *PublicAPI) InitiateHistoryRequests(request InitiateHistoryRequestParams) ([]hexutil.Bytes, error) {
	rst := []hexutil.Bytes{}
	requests, err := api.service.historyUpdates.CreateRequests(request.Requests)
	if err != nil {
		return nil, err
	}
	for i := range requests {
		req := requests[i]
		options := CreateTopicOptionsFromRequest(req)
		bloom := options.ToBloomFilterOption()
		payload, err := bloom.ToMessagesRequestPayload()
		if err != nil {
			return rst, err
		}
		hash, err := api.requestMessagesUsingPayload(req, request.Peer, request.SymKeyID, payload, request.Force, request.Timeout, options.Topics())
		if err != nil {
			return rst, err
		}
		rst = append(rst, hash[:])
	}
	return rst, nil
}

// CompleteRequest client must mark request completed when all envelopes were processed.
func (api *PublicAPI) CompleteRequest(ctx context.Context, hex string) error {
	return api.service.historyUpdates.UpdateFinishedRequest(common.HexToHash(hex))
}

// DEPRECATED: use SendDirectMessage with DH flag
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

	protocolMessage, err := api.service.protocol.BuildDHMessage(privateKey, &privateKey.PublicKey, msg.Payload)
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

func (api *PublicAPI) processPFSMessage(dedupMessage dedup.DeduplicateMessage) error {
	msg := dedupMessage.Message

	privateKeyID := api.service.w.SelectedKeyPairID()
	if privateKeyID == "" {
		return errors.New("no key selected")
	}

	privateKey, err := api.service.w.GetPrivateKey(privateKeyID)
	if err != nil {
		return err
	}

	publicKey, err := crypto.UnmarshalPubkey(msg.Sig)
	if err != nil {
		return err
	}

	response, err := api.service.protocol.HandleMessage(privateKey, publicKey, msg.Payload, dedupMessage.DedupID)

	switch err {
	case nil:
		// Set the decrypted payload
		msg.Payload = response
	case chat.ErrDeviceNotFound:
		// Notify that someone tried to contact us using an invalid bundle
		if privateKey.PublicKey != *publicKey {
			api.log.Warn("Device not found, sending signal", "err", err)
			keyString := fmt.Sprintf("0x%x", crypto.FromECDSAPub(publicKey))
			handler := EnvelopeSignalHandler{}
			handler.DecryptMessageFailed(keyString)
		}
	case chat.ErrNotProtocolMessage:
		// Not using encryption, pass directly to the layer below
		api.log.Debug("Not a protocol message", "err", err)
	default:
		// Log and pass to the client, even if failed to decrypt
		api.log.Error("Failed handling message with error", "err", err)

	}

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

// makeMessagesRequestPayload makes a specific payload for MailServer
// to request historic messages.
func makeMessagesRequestPayload(r MessagesRequest) ([]byte, error) {
	cursor, err := hex.DecodeString(r.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %v", err)
	}
	if len(cursor) > 0 && len(cursor) != mailserver.DBKeyLength {
		return nil, fmt.Errorf("invalid cursor size: expected %d but got %d", mailserver.DBKeyLength, len(cursor))
	}

	payload := mailserver.MessagesRequestPayload{
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
