package shhext

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/mailserver"
	"github.com/status-im/status-go/services/shhext/mailservers"
	"github.com/status-im/status-go/whisper/v6"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/ext"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
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
<<<<<<< HEAD
=======

type Author struct {
	PublicKey types.HexBytes `json:"publicKey"`
	Alias     string         `json:"alias"`
	Identicon string         `json:"identicon"`
}

type Metadata struct {
	DedupID      []byte         `json:"dedupId"`
	EncryptionID types.HexBytes `json:"encryptionId"`
	MessageID    types.HexBytes `json:"messageId"`
	Author       Author         `json:"author"`
}

// ConfirmMessagesProcessedByID is a method to confirm that messages was consumed by
// the client side.
// TODO: this is broken now as it requires dedup ID while a message hash should be used.
func (api *PublicAPI) ConfirmMessagesProcessedByID(messageConfirmations []*Metadata) error {
	confirmationCount := len(messageConfirmations)
	dedupIDs := make([][]byte, confirmationCount)
	encryptionIDs := make([][]byte, confirmationCount)
	for i, confirmation := range messageConfirmations {
		dedupIDs[i] = confirmation.DedupID
		encryptionIDs[i] = confirmation.EncryptionID
	}
	return api.service.ConfirmMessagesProcessed(encryptionIDs)
}

// Post is used to send one-to-one for those who did not enabled device-to-device sync,
// in other words don't use PFS-enabled messages. Otherwise, SendDirectMessage is used.
// It's important to call PublicAPI.afterSend() so that the client receives a signal
// with confirmation that the message left the device.
func (api *PublicAPI) Post(ctx context.Context, newMessage types.NewMessage) (types.HexBytes, error) {
	return api.publicAPI.Post(ctx, newMessage)
}

// SendPublicMessage sends a public chat message to the underlying transport.
// Message's payload is a transit encoded message.
// It's important to call PublicAPI.afterSend() so that the client receives a signal
// with confirmation that the message left the device.
func (api *PublicAPI) SendPublicMessage(ctx context.Context, msg SendPublicMessageRPC) (types.HexBytes, error) {
	chat := protocol.Chat{
		Name: msg.Chat,
	}
	return api.service.messenger.SendRaw(ctx, chat, msg.Payload)
}

// SendDirectMessage sends a 1:1 chat message to the underlying transport
// Message's payload is a transit encoded message.
// It's important to call PublicAPI.afterSend() so that the client receives a signal
// with confirmation that the message left the device.
func (api *PublicAPI) SendDirectMessage(ctx context.Context, msg SendDirectMessageRPC) (types.HexBytes, error) {
	chat := protocol.Chat{
		ChatType: protocol.ChatTypeOneToOne,
		ID:       types.EncodeHex(msg.PubKey),
	}

	return api.service.messenger.SendRaw(ctx, chat, msg.Payload)
}

func (api *PublicAPI) Join(chat protocol.Chat) error {
	return api.service.messenger.Join(chat)
}

func (api *PublicAPI) Leave(chat protocol.Chat) error {
	return api.service.messenger.Leave(chat)
}

func (api *PublicAPI) LeaveGroupChat(ctx Context, chatID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.LeaveGroupChat(ctx, chatID)
}

func (api *PublicAPI) CreateGroupChatWithMembers(ctx Context, name string, members []string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.CreateGroupChatWithMembers(ctx, name, members)
}

func (api *PublicAPI) AddMembersToGroupChat(ctx Context, chatID string, members []string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AddMembersToGroupChat(ctx, chatID, members)
}

func (api *PublicAPI) RemoveMemberFromGroupChat(ctx Context, chatID string, member string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RemoveMemberFromGroupChat(ctx, chatID, member)
}

func (api *PublicAPI) AddAdminsToGroupChat(ctx Context, chatID string, members []string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AddAdminsToGroupChat(ctx, chatID, members)
}

func (api *PublicAPI) ConfirmJoiningGroup(ctx context.Context, chatID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.ConfirmJoiningGroup(ctx, chatID)
}

func (api *PublicAPI) requestMessagesUsingPayload(request db.HistoryRequest, peer, symkeyID string, payload []byte, force bool, timeout time.Duration, topics []types.TopicType) (hash types.Hash, err error) {
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

	err = request.Replace(hash)
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
func (api *PublicAPI) InitiateHistoryRequests(parent context.Context, request InitiateHistoryRequestParams) (rst []types.HexBytes, err error) {
	tx := api.service.storage.NewTx()
	defer func() {
		if err == nil {
			err = tx.Commit()
		}
	}()
	ctx := NewContextFromService(parent, api.service, tx)
	requests, err := api.service.historyUpdates.CreateRequests(ctx, request.Requests)
	if err != nil {
		return nil, err
	}
	var (
		payload []byte
		hash    types.Hash
	)
	for i := range requests {
		req := requests[i]
		options := CreateTopicOptionsFromRequest(req)
		bloom := options.ToBloomFilterOption()
		payload, err = bloom.ToMessagesRequestPayload()
		if err != nil {
			return rst, err
		}
		hash, err = api.requestMessagesUsingPayload(req, request.Peer, request.SymKeyID, payload, request.Force, request.Timeout, options.Topics())
		if err != nil {
			return rst, err
		}
		rst = append(rst, hash.Bytes())
	}
	return rst, err
}

// CompleteRequest client must mark request completed when all envelopes were processed.
func (api *PublicAPI) CompleteRequest(parent context.Context, hex string) (err error) {
	tx := api.service.storage.NewTx()
	ctx := NewContextFromService(parent, api.service, tx)
	err = api.service.historyUpdates.UpdateFinishedRequest(ctx, types.HexToHash(hex))
	if err == nil {
		return tx.Commit()
	}
	return err
}

func (api *PublicAPI) LoadFilters(parent context.Context, chats []*transport.Filter) ([]*transport.Filter, error) {
	return api.service.messenger.LoadFilters(chats)
}

func (api *PublicAPI) SaveChat(parent context.Context, chat *protocol.Chat) error {
	api.log.Info("saving chat", "chat", chat)
	return api.service.messenger.SaveChat(chat)
}

func (api *PublicAPI) Chats(parent context.Context) []*protocol.Chat {
	return api.service.messenger.Chats()
}

func (api *PublicAPI) DeleteChat(parent context.Context, chatID string) error {
	return api.service.messenger.DeleteChat(chatID)
}

func (api *PublicAPI) SaveContact(parent context.Context, contact *protocol.Contact) error {
	return api.service.messenger.SaveContact(contact)
}

func (api *PublicAPI) BlockContact(parent context.Context, contact *protocol.Contact) ([]*protocol.Chat, error) {
	api.log.Info("blocking contact", "contact", contact.ID)
	return api.service.messenger.BlockContact(contact)
}

func (api *PublicAPI) Contacts(parent context.Context) []*protocol.Contact {
	return api.service.messenger.Contacts()
}

func (api *PublicAPI) RemoveFilters(parent context.Context, chats []*transport.Filter) error {
	return api.service.messenger.RemoveFilters(chats)
}

// EnableInstallation enables an installation for multi-device sync.
func (api *PublicAPI) EnableInstallation(installationID string) error {
	return api.service.messenger.EnableInstallation(installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (api *PublicAPI) DisableInstallation(installationID string) error {
	return api.service.messenger.DisableInstallation(installationID)
}

// GetOurInstallations returns all the installations available given an identity
func (api *PublicAPI) GetOurInstallations() []*multidevice.Installation {
	return api.service.messenger.Installations()
}

// SetInstallationMetadata sets the metadata for our own installation
func (api *PublicAPI) SetInstallationMetadata(installationID string, data *multidevice.InstallationMetadata) error {
	return api.service.messenger.SetInstallationMetadata(installationID, data)
}

// VerifyENSNames takes a list of ensdetails and returns whether they match the public key specified
func (api *PublicAPI) VerifyENSNames(details []enstypes.ENSDetails) (map[string]enstypes.ENSResponse, error) {
	return api.service.messenger.VerifyENSNames(api.service.config.VerifyENSURL, ensContractAddress, details)
}

type ApplicationMessagesResponse struct {
	Messages []*protocol.Message `json:"messages"`
	Cursor   string              `json:"cursor"`
}

func (api *PublicAPI) ChatMessages(chatID, cursor string, limit int) (*ApplicationMessagesResponse, error) {
	messages, cursor, err := api.service.messenger.MessageByChatID(chatID, cursor, limit)
	if err != nil {
		return nil, err
	}

	return &ApplicationMessagesResponse{
		Messages: messages,
		Cursor:   cursor,
	}, nil
}

func (api *PublicAPI) DeleteMessage(id string) error {
	return api.service.messenger.DeleteMessage(id)
}

func (api *PublicAPI) DeleteMessagesByChatID(id string) error {
	return api.service.messenger.DeleteMessagesByChatID(id)
}

func (api *PublicAPI) MarkMessagesSeen(chatID string, ids []string) error {
	return api.service.messenger.MarkMessagesSeen(chatID, ids)
}

func (api *PublicAPI) UpdateMessageOutgoingStatus(id, newOutgoingStatus string) error {
	return api.service.messenger.UpdateMessageOutgoingStatus(id, newOutgoingStatus)
}

func (api *PublicAPI) SendChatMessage(ctx context.Context, message *protocol.Message) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendChatMessage(ctx, message)
}

func (api *PublicAPI) ReSendChatMessage(ctx context.Context, messageID string) error {
	return api.service.messenger.ReSendChatMessage(ctx, messageID)
}

func (api *PublicAPI) RequestTransaction(ctx context.Context, chatID, value, contract, address string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RequestTransaction(ctx, chatID, value, contract, address)
}

func (api *PublicAPI) RequestAddressForTransaction(ctx context.Context, chatID, from, value, contract string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RequestAddressForTransaction(ctx, chatID, from, value, contract)
}

func (api *PublicAPI) DeclineRequestAddressForTransaction(ctx context.Context, messageID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.DeclineRequestAddressForTransaction(ctx, messageID)
}

func (api *PublicAPI) DeclineRequestTransaction(ctx context.Context, messageID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.DeclineRequestTransaction(ctx, messageID)
}

func (api *PublicAPI) AcceptRequestAddressForTransaction(ctx context.Context, messageID, address string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AcceptRequestAddressForTransaction(ctx, messageID, address)
}

func (api *PublicAPI) SendTransaction(ctx context.Context, chatID, value, contract, transactionHash string, signature types.HexBytes) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendTransaction(ctx, chatID, value, contract, transactionHash, signature)
}

func (api *PublicAPI) AcceptRequestTransaction(ctx context.Context, transactionHash, messageID string, signature types.HexBytes) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AcceptRequestTransaction(ctx, transactionHash, messageID, signature)
}

func (api *PublicAPI) SendContactUpdates(ctx context.Context, name, picture string) error {
	return api.service.messenger.SendContactUpdates(ctx, name, picture)
}

func (api *PublicAPI) SendContactUpdate(ctx context.Context, contactID, name, picture string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendContactUpdate(ctx, contactID, name, picture)
}

func (api *PublicAPI) SendPairInstallation(ctx context.Context) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendPairInstallation(ctx)
}

func (api *PublicAPI) SyncDevices(ctx context.Context, name, picture string) error {
	return api.service.messenger.SyncDevices(ctx, name, picture)
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

// makeMessagesRequestPayload makes a specific payload for MailServer
// to request historic messages.
func makeMessagesRequestPayload(r MessagesRequest) ([]byte, error) {
	cursor, err := hex.DecodeString(r.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %v", err)
	}

	if len(cursor) > 0 && len(cursor) != mailserver.CursorLength {
		return nil, fmt.Errorf("invalid cursor size: expected %d but got %d", mailserver.CursorLength, len(cursor))
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

	return types.TopicToBloom(r.Topic)
}

func topicsToBloom(topics ...types.TopicType) []byte {
	i := new(big.Int)
	for _, topic := range topics {
		bloom := types.TopicToBloom(topic)
		i.Or(i, new(big.Int).SetBytes(bloom[:]))
	}

	combined := make([]byte, types.BloomFilterSize)
	data := i.Bytes()
	copy(combined[types.BloomFilterSize-len(data):], data[:])

	return combined
}
>>>>>>> 25d46c6d82fbf3e4b6659cb96cf1789bac87f12a
