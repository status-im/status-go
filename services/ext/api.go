package ext

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/mailserver"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
	"github.com/status-im/status-go/protocol/urls"
	"github.com/status-im/status-go/services/ext/mailservers"
)

const (
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
	Topic types.TopicType `json:"topic"`

	// Topics is a list of Whisper topics.
	Topics []types.TopicType `json:"topics"`

	// SymKeyID is an ID of a symmetric key to authenticate to MailServer.
	// It's derived from MailServer password.
	SymKeyID string `json:"symKeyID"`

	// Timeout is the time to live of the request specified in seconds.
	// Default is 10 seconds
	Timeout time.Duration `json:"timeout"`

	// Force ensures that requests will bypass enforced delay.
	Force bool `json:"force"`
}

func (r *MessagesRequest) SetDefaults(now time.Time) {
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

// MessagesResponse is a response for requestMessages2 method.
type MessagesResponse struct {
	// Cursor from the response can be used to retrieve more messages
	// for the previous request.
	Cursor string `json:"cursor"`

	// Error indicates that something wrong happened when sending messages
	// to the requester.
	Error error `json:"error"`
}

// -----
// PUBLIC API
// -----

// PublicAPI extends whisper public API.
type PublicAPI struct {
	service  *Service
	eventSub mailservers.EnvelopeEventSubscriber
	log      log.Logger
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service, eventSub mailservers.EnvelopeEventSubscriber) *PublicAPI {
	return &PublicAPI{
		service:  s,
		eventSub: eventSub,
		log:      log.New("package", "status-go/services/sshext.PublicAPI"),
	}
}

// RetryConfig specifies configuration for retries with timeout and max amount of retries.
type RetryConfig struct {
	BaseTimeout time.Duration
	// StepTimeout defines duration increase per each retry.
	StepTimeout time.Duration
	MaxRetries  int
}

func WaitForExpiredOrCompleted(requestID types.Hash, events chan types.EnvelopeEvent, timeout time.Duration) (*types.MailServerResponse, error) {
	expired := fmt.Errorf("request %x expired", requestID)
	after := time.NewTimer(timeout)
	defer after.Stop()
	for {
		var ev types.EnvelopeEvent
		select {
		case ev = <-events:
		case <-after.C:
			return nil, expired
		}
		if ev.Hash != requestID {
			continue
		}
		switch ev.Event {
		case types.EventMailServerRequestCompleted:
			data, ok := ev.Data.(*types.MailServerResponse)
			if ok {
				return data, nil
			}
			return nil, errors.New("invalid event data type")
		case types.EventMailServerRequestExpired:
			return nil, expired
		}
	}
}

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

func (api *PublicAPI) Leave(chat protocol.Chat) error {
	return api.service.messenger.Leave(chat)
}

func (api *PublicAPI) LeaveGroupChat(ctx Context, chatID string, remove bool) (*protocol.MessengerResponse, error) {
	return api.service.messenger.LeaveGroupChat(ctx, chatID, remove)
}

func (api *PublicAPI) CreateGroupChatWithMembers(ctx Context, name string, members []string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.CreateGroupChatWithMembers(ctx, name, members)
}

func (api *PublicAPI) CreateGroupChatFromInvitation(name string, chatID string, adminPK string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.CreateGroupChatFromInvitation(name, chatID, adminPK)
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

func (api *PublicAPI) ChangeGroupChatName(ctx Context, chatID string, name string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.ChangeGroupChatName(ctx, chatID, name)
}

func (api *PublicAPI) SendGroupChatInvitationRequest(ctx Context, chatID string, adminPK string, message string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendGroupChatInvitationRequest(ctx, chatID, adminPK, message)
}

func (api *PublicAPI) GetGroupChatInvitations() ([]*protocol.GroupChatInvitation, error) {
	return api.service.messenger.GetGroupChatInvitations()
}

func (api *PublicAPI) SendGroupChatInvitationRejection(ctx Context, invitationRequestID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendGroupChatInvitationRejection(ctx, invitationRequestID)
}

func (api *PublicAPI) LoadFilters(parent context.Context, chats []*transport.Filter) ([]*transport.Filter, error) {
	return api.service.messenger.LoadFilters(chats)
}

func (api *PublicAPI) SaveChat(parent context.Context, chat *protocol.Chat) error {
	return api.service.messenger.SaveChat(chat)
}

func (api *PublicAPI) CreateOneToOneChat(parent context.Context, request *requests.CreateOneToOneChat) (*protocol.MessengerResponse, error) {
	return api.service.messenger.CreateOneToOneChat(request)
}

func (api *PublicAPI) Chats(parent context.Context) []*protocol.Chat {
	return api.service.messenger.Chats()
}

func (api *PublicAPI) DeleteChat(parent context.Context, chatID string) error {
	return api.service.messenger.DeleteChat(chatID)
}

func (api *PublicAPI) MuteChat(parent context.Context, chatID string) error {
	return api.service.messenger.MuteChat(chatID)
}

func (api *PublicAPI) UnmuteChat(parent context.Context, chatID string) error {
	return api.service.messenger.UnmuteChat(chatID)
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

func (api *PublicAPI) GetContactByID(parent context.Context, id string) *protocol.Contact {
	return api.service.messenger.GetContactByID(id)
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

// Communities returns a list of communities that are stored
func (api *PublicAPI) Communities(parent context.Context) ([]*communities.Community, error) {
	return api.service.messenger.Communities()
}

// JoinedCommunities returns a list of communities that the user has joined
func (api *PublicAPI) JoinedCommunities(parent context.Context) ([]*communities.Community, error) {
	return api.service.messenger.JoinedCommunities()
}

// JoinCommunity joins a community with the given ID
func (api *PublicAPI) JoinCommunity(parent context.Context, communityID types.HexBytes) (*protocol.MessengerResponse, error) {
	return api.service.messenger.JoinCommunity(communityID)
}

// LeaveCommunity leaves a commuity with the given ID
func (api *PublicAPI) LeaveCommunity(parent context.Context, communityID types.HexBytes) (*protocol.MessengerResponse, error) {
	return api.service.messenger.LeaveCommunity(communityID)
}

// CreateCommunity creates a new community with the provided description
func (api *PublicAPI) CreateCommunity(request *requests.CreateCommunity) (*protocol.MessengerResponse, error) {
	return api.service.messenger.CreateCommunity(request)

}

// ExportCommunity exports the private key of the community with given ID
func (api *PublicAPI) ExportCommunity(id types.HexBytes) (types.HexBytes, error) {
	key, err := api.service.messenger.ExportCommunity(id)
	if err != nil {
		return nil, err
	}
	return crypto.FromECDSA(key), nil
}

// ImportCommunity imports a community with the given private key in hex
func (api *PublicAPI) ImportCommunity(hexPrivateKey string) (*protocol.MessengerResponse, error) {
	// Strip the 0x from the beginning
	privateKey, err := crypto.HexToECDSA(hexPrivateKey[2:])
	if err != nil {
		return nil, err
	}
	return api.service.messenger.ImportCommunity(privateKey)

}

// CreateCommunityChat creates a community chat in the given community
func (api *PublicAPI) CreateCommunityChat(communityID types.HexBytes, c *protobuf.CommunityChat) (*protocol.MessengerResponse, error) {
	return api.service.messenger.CreateCommunityChat(communityID, c)
}

// InviteUsersToCommunity invites the users with pks to the community with ID
func (api *PublicAPI) InviteUsersToCommunity(request *requests.InviteUsersToCommunity) (*protocol.MessengerResponse, error) {
	return api.service.messenger.InviteUsersToCommunity(request)
}

// ShareCommunity share the community with a set of users
func (api *PublicAPI) ShareCommunity(request *requests.ShareCommunity) (*protocol.MessengerResponse, error) {
	return api.service.messenger.ShareCommunity(request)
}

// RemoveUserFromCommunity removes the user with pk from the community with ID
func (api *PublicAPI) RemoveUserFromCommunity(communityID types.HexBytes, userPublicKey string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RemoveUserFromCommunity(communityID, userPublicKey)
}

// MyPendingRequestsToJoin returns the pending requests for the logged in user
func (api *PublicAPI) MyPendingRequestsToJoin() ([]*communities.RequestToJoin, error) {
	return api.service.messenger.MyPendingRequestsToJoin()
}

// PendingRequestsToJoinForCommunity returns the pending requests to join for a given community
func (api *PublicAPI) PendingRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return api.service.messenger.PendingRequestsToJoinForCommunity(id)
}

// AcceptRequestToJoinCommunity accepts a pending request to join a community
func (api *PublicAPI) AcceptRequestToJoinCommunity(request *requests.AcceptRequestToJoinCommunity) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AcceptRequestToJoinCommunity(request)
}

// DeclineRequestToJoinCommunity accepts a pending request to join a community
func (api *PublicAPI) DeclineRequestToJoinCommunity(request *requests.DeclineRequestToJoinCommunity) error {
	return api.service.messenger.DeclineRequestToJoinCommunity(request)
}

// RequestToJoinCommunity requests to join a particular community
func (api *PublicAPI) RequestToJoinCommunity(request *requests.RequestToJoinCommunity) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RequestToJoinCommunity(request)
}

type ApplicationMessagesResponse struct {
	Messages []*common.Message `json:"messages"`
	Cursor   string            `json:"cursor"`
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

func (api *PublicAPI) StartMessenger() (*protocol.MessengerResponse, error) {
	return api.service.StartMessenger()
}

func (api *PublicAPI) DeleteMessage(id string) error {
	return api.service.messenger.DeleteMessage(id)
}

func (api *PublicAPI) DeleteMessagesByChatID(id string) error {
	return api.service.messenger.DeleteMessagesByChatID(id)
}

func (api *PublicAPI) MarkMessagesSeen(chatID string, ids []string) (uint64, error) {
	return api.service.messenger.MarkMessagesSeen(chatID, ids)
}

func (api *PublicAPI) MarkAllRead(chatID string) error {
	return api.service.messenger.MarkAllRead(chatID)
}

func (api *PublicAPI) AddContact(ctx context.Context, pubKey string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AddContact(ctx, pubKey)
}

func (api *PublicAPI) RemoveContact(ctx context.Context, pubKey string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RemoveContact(ctx, pubKey)
}

func (api *PublicAPI) ClearHistory(chatID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.ClearHistory(chatID)
}

func (api *PublicAPI) DeactivateChat(chatID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.DeactivateChat(chatID)
}

func (api *PublicAPI) UpdateMessageOutgoingStatus(id, newOutgoingStatus string) error {
	return api.service.messenger.UpdateMessageOutgoingStatus(id, newOutgoingStatus)
}

func (api *PublicAPI) SendChatMessage(ctx context.Context, message *common.Message) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendChatMessage(ctx, message)
}

func (api *PublicAPI) ReSendChatMessage(ctx context.Context, messageID string) error {
	return api.service.messenger.ReSendChatMessage(ctx, messageID)
}

func (api *PublicAPI) SendChatMessages(ctx context.Context, messages []*common.Message) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendChatMessages(ctx, messages)
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

func (api *PublicAPI) SignMessageWithChatKey(ctx context.Context, message string) (types.HexBytes, error) {
	return api.service.messenger.SignMessage(message)
}

func (api *PublicAPI) UpdateMailservers(enodes []string) error {
	nodes := make([]*enode.Node, len(enodes))
	for i, rawurl := range enodes {
		node, err := enode.ParseV4(rawurl)
		if err != nil {
			return err
		}
		nodes[i] = node
	}
	return api.service.UpdateMailservers(nodes)
}

// PushNotifications server endpoints

func (api *PublicAPI) StartPushNotificationsServer() error {
	err := api.service.accountsDB.SaveSetting("push-notifications-server-enabled?", true)
	if err != nil {
		return err
	}

	return api.service.messenger.StartPushNotificationsServer()
}

func (api *PublicAPI) StopPushNotificationsServer() error {
	err := api.service.accountsDB.SaveSetting("push-notifications-server-enabled?", false)
	if err != nil {
		return err
	}

	return api.service.messenger.StopPushNotificationsServer()
}

// PushNotification client endpoints

func (api *PublicAPI) RegisterForPushNotifications(ctx context.Context, deviceToken string, apnTopic string, tokenType protobuf.PushNotificationRegistration_TokenType) error {
	// We set both for now as they are equivalent
	err := api.service.accountsDB.SaveSetting("remote-push-notifications-enabled?", true)
	if err != nil {
		return err
	}
	err = api.service.accountsDB.SaveSetting("notifications-enabled?", true)
	if err != nil {
		return err
	}

	return api.service.messenger.RegisterForPushNotifications(ctx, deviceToken, apnTopic, tokenType)
}

func (api *PublicAPI) UnregisterFromPushNotifications(ctx context.Context) error {
	err := api.service.accountsDB.SaveSetting("remote-push-notifications-enabled?", false)
	if err != nil {
		return err
	}
	err = api.service.accountsDB.SaveSetting("notifications-enabled?", false)
	if err != nil {
		return err
	}

	return api.service.messenger.UnregisterFromPushNotifications(ctx)
}

func (api *PublicAPI) DisableSendingNotifications(ctx context.Context) error {
	err := api.service.accountsDB.SaveSetting("send-push-notifications?", false)
	if err != nil {
		return err
	}

	return api.service.messenger.DisableSendingPushNotifications()
}

func (api *PublicAPI) EnableSendingNotifications(ctx context.Context) error {
	err := api.service.accountsDB.SaveSetting("send-push-notifications?", true)
	if err != nil {
		return err
	}
	return api.service.messenger.EnableSendingPushNotifications()
}

func (api *PublicAPI) EnablePushNotificationsFromContactsOnly(ctx context.Context) error {
	err := api.service.accountsDB.SaveSetting("push-notifications-from-contacts-only?", true)
	if err != nil {
		return err
	}
	return api.service.messenger.EnablePushNotificationsFromContactsOnly()
}

func (api *PublicAPI) DisablePushNotificationsFromContactsOnly(ctx context.Context) error {
	err := api.service.accountsDB.SaveSetting("push-notifications-from-contacts-only?", false)
	if err != nil {
		return err
	}
	return api.service.messenger.DisablePushNotificationsFromContactsOnly()
}

func (api *PublicAPI) EnablePushNotificationsBlockMentions(ctx context.Context) error {
	err := api.service.accountsDB.SaveSetting("push-notifications-block-mentions?", true)
	if err != nil {
		return err
	}
	return api.service.messenger.EnablePushNotificationsBlockMentions()
}

func (api *PublicAPI) DisablePushNotificationsBlockMentions(ctx context.Context) error {
	err := api.service.accountsDB.SaveSetting("push-notifications-block-mentions?", false)
	if err != nil {
		return err
	}
	return api.service.messenger.DisablePushNotificationsBlockMentions()
}

func (api *PublicAPI) AddPushNotificationsServer(ctx context.Context, publicKeyBytes types.HexBytes) error {
	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return err
	}

	// this is coming from a user, so it has to be a custom server
	return api.service.messenger.AddPushNotificationsServer(ctx, publicKey, pushnotificationclient.ServerTypeCustom)
}

func (api *PublicAPI) RemovePushNotificationServer(ctx context.Context, publicKeyBytes types.HexBytes) error {
	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return err
	}

	return api.service.messenger.RemovePushNotificationServer(ctx, publicKey)
}

func (api *PublicAPI) GetPushNotificationsServers() ([]*pushnotificationclient.PushNotificationServer, error) {
	return api.service.messenger.GetPushNotificationsServers()
}

func (api *PublicAPI) RegisteredForPushNotifications() (bool, error) {
	return api.service.messenger.RegisteredForPushNotifications()
}

// Emoji

func (api *PublicAPI) SendEmojiReaction(ctx context.Context, chatID, messageID string, emojiID protobuf.EmojiReaction_Type) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendEmojiReaction(ctx, chatID, messageID, emojiID)
}

func (api *PublicAPI) SendEmojiReactionRetraction(ctx context.Context, emojiReactionID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendEmojiReactionRetraction(ctx, emojiReactionID)
}

func (api *PublicAPI) EmojiReactionsByChatID(chatID string, cursor string, limit int) ([]*protocol.EmojiReaction, error) {
	return api.service.messenger.EmojiReactionsByChatID(chatID, cursor, limit)
}

// Urls

func (api *PublicAPI) GetLinkPreviewWhitelist() []urls.Site {
	return urls.LinkPreviewWhitelist()
}

func (api *PublicAPI) GetLinkPreviewData(link string) (previewData urls.LinkPreviewData, err error) {
	return urls.GetLinkPreviewData(link)
}

func (api *PublicAPI) EnsVerified(pk, ensName string) error {
	return api.service.messenger.ENSVerified(pk, ensName)
}

// Echo is a method for testing purposes.
func (api *PublicAPI) Echo(ctx context.Context, message string) (string, error) {
	return message, nil
}

// -----
// HELPER
// -----

// MakeMessagesRequestPayload makes a specific payload for MailServer
// to request historic messages.
// DEPRECATED
func MakeMessagesRequestPayload(r MessagesRequest) ([]byte, error) {
	cursor, err := hex.DecodeString(r.Cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %v", err)
	}

	if len(cursor) > 0 && len(cursor) != mailserver.CursorLength {
		return nil, fmt.Errorf("invalid cursor size: expected %d but got %d", mailserver.CursorLength, len(cursor))
	}

	payload := mailserver.MessagesRequestPayload{
		Lower: r.From,
		Upper: r.To,
		// We need to pass bloom filter for
		// backward compatibility
		Bloom:  createBloomFilter(r),
		Topics: topicsToByteArray(r.Topics),
		Limit:  r.Limit,
		Cursor: cursor,
		// Client must tell the MailServer if it supports batch responses.
		// This can be removed in the future.
		Batch: true,
	}

	return rlp.EncodeToBytes(payload)
}

func topicsToByteArray(topics []types.TopicType) [][]byte {

	var response [][]byte
	for idx := range topics {
		response = append(response, topics[idx][:])
	}

	return response
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
		i.Or(i, new(big.Int).SetBytes(bloom))
	}

	combined := make([]byte, types.BloomFilterSize)
	data := i.Bytes()
	copy(combined[types.BloomFilterSize-len(data):], data)

	return combined
}

// TopicsToBloom squashes all topics into a single bloom filter.
func TopicsToBloom(topics ...types.TopicType) []byte {
	return topicsToBloom(topics...)
}
