// +build !nimbus

package shhext

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/whisper/v6"
)

const (
	// defaultWorkTime is a work time reported in messages sent to MailServer nodes.
	defaultWorkTime = 5
)

// PublicAPI extends whisper public API.
type PublicAPI struct {
	*ext.PublicAPI

	service   *Service
	publicAPI types.PublicWhisperAPI
	log       log.Logger
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		PublicAPI: ext.NewPublicAPI(s.Service, s.w),
		service:   s,
		publicAPI: s.w.PublicWhisperAPI(),
		log:       log.New("package", "status-go/services/sshext.PublicAPI"),
	}
}

// makeEnvelop makes an envelop for a historic messages request.
// Symmetric key is used to authenticate to MailServer.
// PK is the current node ID.
// DEPRECATED
func makeEnvelop(
	payload []byte,
	symKey []byte,
	publicKey *ecdsa.PublicKey,
	nodeID *ecdsa.PrivateKey,
	pow float64,
	now time.Time,
) (types.Envelope, error) {
	// TODO: replace with an types.Envelope creator passed to the API struct
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
	envelope, err := message.Wrap(&params, now)
	if err != nil {
		return nil, err
	}
	return gethbridge.NewWhisperEnvelope(envelope), nil
}

// RequestMessages sends a request for historic messages to a MailServer.
func (api *PublicAPI) RequestMessages(_ context.Context, r ext.MessagesRequest) (types.HexBytes, error) {
	api.log.Info("RequestMessages", "request", r)

	now := api.service.w.GetCurrentTime()
	r.SetDefaults(now)

	if r.From > r.To {
		return nil, fmt.Errorf("Query range is invalid: from > to (%d > %d)", r.From, r.To)
	}

	mailServerNode, err := api.service.GetPeer(r.MailServerPeer)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", ext.ErrInvalidMailServerPeer, err)
	}

	var (
		symKey    []byte
		publicKey *ecdsa.PublicKey
	)

	if r.SymKeyID != "" {
		symKey, err = api.service.w.GetSymKey(r.SymKeyID)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", ext.ErrInvalidSymKeyID, err)
		}
	} else {
		publicKey = mailServerNode.Pubkey()
	}

	payload, err := ext.MakeMessagesRequestPayload(r)
	if err != nil {
		return nil, err
	}

	envelope, err := makeEnvelop(
		payload,
		symKey,
		publicKey,
		api.service.NodeID(),
		api.service.w.MinPow(),
		now,
	)
	if err != nil {
		return nil, err
	}
	hash := envelope.Hash()

	if !r.Force {
		err = api.service.RequestsRegistry().Register(hash, r.Topics)
		if err != nil {
			return nil, err
		}
	}

	if err := api.service.w.RequestHistoricMessagesWithTimeout(mailServerNode.ID().Bytes(), envelope, r.Timeout*time.Second); err != nil {
		if !r.Force {
			api.service.RequestsRegistry().Unregister(hash)
		}
		return nil, err
	}

	return hash[:], nil
}

// RequestMessagesSync repeats MessagesRequest using configuration in retry conf.
func (api *PublicAPI) RequestMessagesSync(conf ext.RetryConfig, r ext.MessagesRequest) (ext.MessagesResponse, error) {
	var resp ext.MessagesResponse

	events := make(chan types.EnvelopeEvent, 10)
	var (
		requestID types.HexBytes
		err       error
		retries   int
	)
	for retries <= conf.MaxRetries {
		sub := api.service.w.SubscribeEnvelopeEvents(events)
		r.Timeout = conf.BaseTimeout + conf.StepTimeout*time.Duration(retries)
		timeout := r.Timeout
		// FIXME this weird conversion is required because MessagesRequest expects seconds but defines time.Duration
		r.Timeout = time.Duration(int(r.Timeout.Seconds()))
		requestID, err = api.RequestMessages(context.Background(), r)
		if err != nil {
			sub.Unsubscribe()
			return resp, err
		}
		mailServerResp, err := ext.WaitForExpiredOrCompleted(types.BytesToHash(requestID), events, timeout)
		sub.Unsubscribe()
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

	// FollowCursor if true loads messages until cursor is empty.
	FollowCursor bool `json:"followCursor"`

	// Topics is a list of Whisper topics.
	// If empty, a full bloom filter will be used.
	Topics []types.TopicType `json:"topics"`
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

// createSyncMailRequest creates SyncMailRequest. It uses a full bloom filter
// if no topics are given.
func createSyncMailRequest(r SyncMessagesRequest) (types.SyncMailRequest, error) {
	var bloom []byte
	if len(r.Topics) > 0 {
		bloom = ext.TopicsToBloom(r.Topics...)
	} else {
		bloom = types.MakeFullNodeBloom()
	}

	cursor, err := hex.DecodeString(r.Cursor)
	if err != nil {
		return types.SyncMailRequest{}, err
	}

	return types.SyncMailRequest{
		Lower:  r.From,
		Upper:  r.To,
		Bloom:  bloom,
		Limit:  r.Limit,
		Cursor: cursor,
	}, nil
}

func createSyncMessagesResponse(r types.SyncEventResponse) SyncMessagesResponse {
	return SyncMessagesResponse{
		Cursor: hex.EncodeToString(r.Cursor),
		Error:  r.Error,
	}
}

// SyncMessages sends a request to a given MailServerPeer to sync historic messages.
// MailServerPeers needs to be added as a trusted peer first.
func (api *PublicAPI) SyncMessages(ctx context.Context, r SyncMessagesRequest) (SyncMessagesResponse, error) {
	log.Info("SyncMessages start", "request", r)

	var response SyncMessagesResponse

	mailServerEnode, err := enode.ParseV4(r.MailServerPeer)
	if err != nil {
		return response, fmt.Errorf("invalid MailServerPeer: %v", err)
	}
	mailServerID := mailServerEnode.ID().Bytes()

	request, err := createSyncMailRequest(r)
	if err != nil {
		return response, fmt.Errorf("failed to create a sync mail request: %v", err)
	}

	for {
		log.Info("Sending a request to sync messages", "request", request)

		resp, err := api.service.SyncMessages(ctx, mailServerID, request)
		if err != nil {
			return response, err
		}

		log.Info("Syncing messages response", "error", resp.Error, "cursor", fmt.Sprintf("%#x", resp.Cursor))

		if resp.Error != "" || len(resp.Cursor) == 0 || !r.FollowCursor {
			return createSyncMessagesResponse(resp), nil
		}

		request.Cursor = resp.Cursor
	}
}
