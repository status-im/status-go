//go:build nimbus
// +build nimbus

package shhext

import (
	"context"

	"github.com/status-im/status-go/protocol/common"

	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/transport"
)

// -----
// PUBLIC API
// -----

// NimbusPublicAPI extends whisper public API.
type NimbusPublicAPI struct {
	service   *NimbusService
	publicAPI types.PublicWhisperAPI
}

// NewPublicAPI returns instance of the public API.
func NewNimbusPublicAPI(s *NimbusService) *NimbusPublicAPI {
	return &NimbusPublicAPI{
		service:   s,
		publicAPI: s.w.PublicWhisperAPI(),
	}
}

// ConfirmMessagesProcessedByID is a method to confirm that messages was consumed by
// the client side.
// TODO: this is broken now as it requires dedup ID while a message hash should be used.
func (api *NimbusPublicAPI) ConfirmMessagesProcessedByID(messageConfirmations []*Metadata) error {
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
// It's important to call NimbusPublicAPI.afterSend() so that the client receives a signal
// with confirmation that the message left the device.
func (api *NimbusPublicAPI) Post(ctx context.Context, newMessage types.NewMessage) (types.HexBytes, error) {
	return api.publicAPI.Post(ctx, newMessage)
}

func (api *NimbusPublicAPI) Join(chat protocol.Chat) error {
	return api.service.messenger.Join(chat)
}

func (api *NimbusPublicAPI) Leave(chat protocol.Chat) error {
	return api.service.messenger.Leave(chat)
}

func (api *NimbusPublicAPI) LeaveGroupChat(ctx Context, chatID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.LeaveGroupChat(ctx, chatID)
}

func (api *NimbusPublicAPI) CreateGroupChatWithMembers(ctx Context, name string, members []string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.CreateGroupChatWithMembers(ctx, name, members)
}

func (api *NimbusPublicAPI) AddMembersToGroupChat(ctx Context, chatID string, members []string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AddMembersToGroupChat(ctx, chatID, members)
}

func (api *NimbusPublicAPI) RemoveMemberFromGroupChat(ctx Context, chatID string, member string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RemoveMembersFromGroupChat(ctx, chatID, []string{member})
}

func (api *NimbusPublicAPI) AddAdminsToGroupChat(ctx Context, chatID string, members []string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AddAdminsToGroupChat(ctx, chatID, members)
}

func (api *NimbusPublicAPI) ConfirmJoiningGroup(ctx context.Context, chatID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.ConfirmJoiningGroup(ctx, chatID)
}

func (api *NimbusPublicAPI) LoadFilters(parent context.Context, chats []*transport.Filter) ([]*transport.Filter, error) {
	return api.service.messenger.LoadFilters(chats)
}

func (api *NimbusPublicAPI) SaveChat(parent context.Context, chat *protocol.Chat) error {
	return api.service.messenger.SaveChat(chat)
}

func (api *NimbusPublicAPI) Chats(parent context.Context) []*protocol.Chat {
	return api.service.messenger.Chats()
}

func (api *NimbusPublicAPI) DeleteChat(parent context.Context, chatID string) error {
	return api.service.messenger.DeleteChat(chatID)
}

func (api *NimbusPublicAPI) SaveContact(parent context.Context, contact *protocol.Contact) error {
	return api.service.messenger.SaveContact(contact)
}

func (api *NimbusPublicAPI) BlockContact(parent context.Context, contact *protocol.Contact) (*protocol.MessengerResponse, error) {
	return api.service.messenger.BlockContact(contact)
}

func (api *NimbusPublicAPI) Contacts(parent context.Context) []*protocol.Contact {
	return api.service.messenger.Contacts()
}

func (api *NimbusPublicAPI) RemoveFilters(parent context.Context, chats []*transport.Filter) error {
	return api.service.messenger.RemoveFilters(chats)
}

// EnableInstallation enables an installation for multi-device sync.
func (api *NimbusPublicAPI) EnableInstallation(installationID string) error {
	return api.service.messenger.EnableInstallation(installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (api *NimbusPublicAPI) DisableInstallation(installationID string) error {
	return api.service.messenger.DisableInstallation(installationID)
}

// GetOurInstallations returns all the installations available given an identity
func (api *NimbusPublicAPI) GetOurInstallations() []*multidevice.Installation {
	return api.service.messenger.Installations()
}

// SetInstallationMetadata sets the metadata for our own installation
func (api *NimbusPublicAPI) SetInstallationMetadata(installationID string, data *multidevice.InstallationMetadata) error {
	return api.service.messenger.SetInstallationMetadata(installationID, data)
}

// VerifyENSNames takes a list of ensdetails and returns whether they match the public key specified
func (api *NimbusPublicAPI) VerifyENSNames(details []enstypes.ENSDetails) (map[string]enstypes.ENSResponse, error) {
	return api.service.messenger.VerifyENSNames(api.service.config.VerifyENSURL, ensContractAddress, details)
}

type ApplicationMessagesResponse struct {
	Messages []*common.Message `json:"messages"`
	Cursor   string            `json:"cursor"`
}

func (api *NimbusPublicAPI) ChatMessages(chatID, cursor string, limit int) (*ApplicationMessagesResponse, error) {
	messages, cursor, err := api.service.messenger.MessageByChatID(chatID, cursor, limit)
	if err != nil {
		return nil, err
	}

	return &ApplicationMessagesResponse{
		Messages: messages,
		Cursor:   cursor,
	}, nil
}

func (api *NimbusPublicAPI) DeleteMessage(id string) error {
	return api.service.messenger.DeleteMessage(id)
}

func (api *NimbusPublicAPI) DeleteMessagesByChatID(id string) error {
	return api.service.messenger.DeleteMessagesByChatID(id)
}

func (api *NimbusPublicAPI) MarkMessagesSeen(chatID string, ids []string) error {
	return api.service.messenger.MarkMessagesSeen(chatID, ids)
}

func (api *NimbusPublicAPI) UpdateMessageOutgoingStatus(id, newOutgoingStatus string) error {
	return api.service.messenger.UpdateMessageOutgoingStatus(id, newOutgoingStatus)
}

func (api *PublicAPI) StartMessenger() error {
	return api.service.StartMessenger()
}

func (api *NimbusPublicAPI) SendChatMessage(ctx context.Context, message *common.Message) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendChatMessage(ctx, message)
}

func (api *NimbusPublicAPI) SendPinMessage(ctx context.Context, message *common.PinMessage) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendPinMessage(ctx, message)
}

func (api *NimbusPublicAPI) ReSendChatMessage(ctx context.Context, messageID string) error {
	return api.service.messenger.ReSendChatMessage(ctx, messageID)
}

func (api *NimbusPublicAPI) RequestTransaction(ctx context.Context, chatID, value, contract, address string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RequestTransaction(ctx, chatID, value, contract, address)
}

func (api *NimbusPublicAPI) RequestAddressForTransaction(ctx context.Context, chatID, from, value, contract string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.RequestAddressForTransaction(ctx, chatID, from, value, contract)
}

func (api *NimbusPublicAPI) DeclineRequestAddressForTransaction(ctx context.Context, messageID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.DeclineRequestAddressForTransaction(ctx, messageID)
}

func (api *NimbusPublicAPI) DeclineRequestTransaction(ctx context.Context, messageID string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.DeclineRequestTransaction(ctx, messageID)
}

func (api *NimbusPublicAPI) AcceptRequestAddressForTransaction(ctx context.Context, messageID, address string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AcceptRequestAddressForTransaction(ctx, messageID, address)
}

func (api *NimbusPublicAPI) SendTransaction(ctx context.Context, chatID, value, contract, transactionHash string, signature types.HexBytes) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendTransaction(ctx, chatID, value, contract, transactionHash, signature)
}

func (api *NimbusPublicAPI) AcceptRequestTransaction(ctx context.Context, transactionHash, messageID string, signature types.HexBytes) (*protocol.MessengerResponse, error) {
	return api.service.messenger.AcceptRequestTransaction(ctx, transactionHash, messageID, signature)
}

func (api *NimbusPublicAPI) SendContactUpdates(ctx context.Context, name, picture string) error {
	return api.service.messenger.SendContactUpdates(ctx, name, picture)
}

func (api *NimbusPublicAPI) SendContactUpdate(ctx context.Context, contactID, name, picture string) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendContactUpdate(ctx, contactID, name, picture)
}

func (api *NimbusPublicAPI) SendPairInstallation(ctx context.Context) (*protocol.MessengerResponse, error) {
	return api.service.messenger.SendPairInstallation(ctx)
}

func (api *NimbusPublicAPI) SyncDevices(ctx context.Context, name, picture string) error {
	return api.service.messenger.SyncDevices(ctx, name, picture)
}
