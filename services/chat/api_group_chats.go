package chat

import (
	"context"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/requests"
)

type GroupChatResponse struct {
	Chat     *Chat             `json:"chat"`
	Messages []*common.Message `json:"messages"`
}

type GroupChatResponseWithInvitations struct {
	Chat        *Chat                           `json:"chat"`
	Messages    []*common.Message               `json:"messages"`
	Invitations []*protocol.GroupChatInvitation `json:"invitations"`
}

type CreateOneToOneChatResponse struct {
	Chat    *Chat             `json:"chat,omitempty"`
	Contact *protocol.Contact `json:"contact,omitempty"`
}

type StartGroupChatResponse struct {
	Chat     *Chat               `json:"chat,omitempty"`
	Contacts []*protocol.Contact `json:"contacts"`
	Messages []*common.Message   `json:"messages,omitempty"`
}

func (api *API) CreateOneToOneChat(ctx context.Context, ID types.HexBytes, ensName string) (*CreateOneToOneChatResponse, error) {
	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))
	response, err := api.s.messenger.CreateOneToOneChat(&requests.CreateOneToOneChat{ID: ID, ENSName: ensName})
	if err != nil {
		return nil, err
	}

	protocolChat := response.Chats()[0]
	pinnedMessages, cursor, err := api.s.messenger.PinnedMessageByChatID(protocolChat.ID, "", -1)
	if err != nil {
		return nil, err
	}
	chat, err := toAPIChat(protocolChat, nil, pubKey, pinnedMessages, cursor)
	if err != nil {
		return nil, err
	}

	var contact *protocol.Contact
	if ensName != "" {
		contact = response.Contacts[0]
	}

	return &CreateOneToOneChatResponse{
		Chat:    chat,
		Contact: contact,
	}, nil
}

func (api *API) CreateGroupChat(ctx context.Context, name string, members []string) (*GroupChatResponse, error) {
	return api.execAndGetGroupChatResponse(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.CreateGroupChatWithMembers(ctx, name, members)
	})
}

func (api *API) CreateGroupChatFromInvitation(name string, chatID string, adminPK string) (*GroupChatResponse, error) {
	return api.execAndGetGroupChatResponse(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.CreateGroupChatFromInvitation(name, chatID, adminPK)
	})
}

func (api *API) LeaveGroupChat(ctx context.Context, chatID string, remove bool) (*GroupChatResponse, error) {
	return api.execAndGetGroupChatResponse(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.LeaveGroupChat(ctx, chatID, remove)
	})
}

func (api *API) AddMembersToGroupChat(ctx context.Context, chatID string, members []string) (*GroupChatResponseWithInvitations, error) {
	return api.execAndGetGroupChatResponseWithInvitations(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.AddMembersToGroupChat(ctx, chatID, members)
	})
}

func (api *API) RemoveMemberFromGroupChat(ctx context.Context, chatID string, member string) (*GroupChatResponse, error) {
	return api.execAndGetGroupChatResponse(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.RemoveMemberFromGroupChat(ctx, chatID, member)
	})
}

func (api *API) AddAdminsToGroupChat(ctx context.Context, chatID string, members []string) (*GroupChatResponse, error) {
	return api.execAndGetGroupChatResponse(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.AddAdminsToGroupChat(ctx, chatID, members)
	})
}

func (api *API) ConfirmJoiningGroup(ctx context.Context, chatID string) (*GroupChatResponse, error) {
	return api.execAndGetGroupChatResponse(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.ConfirmJoiningGroup(ctx, chatID)
	})
}

func (api *API) ChangeGroupChatName(ctx context.Context, chatID string, name string) (*GroupChatResponse, error) {
	return api.execAndGetGroupChatResponse(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.ChangeGroupChatName(ctx, chatID, name)
	})
}

func (api *API) SendGroupChatInvitationRequest(ctx context.Context, chatID string, adminPK string, message string) (*GroupChatResponseWithInvitations, error) {
	return api.execAndGetGroupChatResponseWithInvitations(func() (*protocol.MessengerResponse, error) {
		return api.s.messenger.SendGroupChatInvitationRequest(ctx, chatID, adminPK, message)
	})
}

func (api *API) GetGroupChatInvitations() ([]*protocol.GroupChatInvitation, error) {
	return api.s.messenger.GetGroupChatInvitations()
}

func (api *API) SendGroupChatInvitationRejection(ctx context.Context, invitationRequestID string) ([]*protocol.GroupChatInvitation, error) {
	response, err := api.s.messenger.SendGroupChatInvitationRejection(ctx, invitationRequestID)
	if err != nil {
		return nil, err
	}
	return response.Invitations, nil
}

func (api *API) StartGroupChat(ctx context.Context, name string, members []string) (*StartGroupChatResponse, error) {
	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))

	var response *protocol.MessengerResponse
	var err error
	if len(members) == 1 {
		memberPk, err := common.HexToPubkey(members[0])
		if err != nil {
			return nil, err
		}
		response, err = api.s.messenger.CreateOneToOneChat(&requests.CreateOneToOneChat{
			ID: types.HexBytes(crypto.FromECDSAPub(memberPk)),
		})
		if err != nil {
			return nil, err
		}
	} else {
		response, err = api.s.messenger.CreateGroupChatWithMembers(ctx, name, members)
		if err != nil {
			return nil, err
		}
	}

	chat, err := toAPIChat(response.Chats()[0], nil, pubKey, nil, "")
	if err != nil {
		return nil, err
	}

	return &StartGroupChatResponse{
		Chat:     chat,
		Contacts: response.Contacts,
		Messages: response.Messages(),
	}, nil
}

func toGroupChatResponse(pubKey string, response *protocol.MessengerResponse) (*GroupChatResponse, error) {
	chat, err := toAPIChat(response.Chats()[0], nil, pubKey, nil, "")
	if err != nil {
		return nil, err
	}

	return &GroupChatResponse{
		Chat:     chat,
		Messages: response.Messages(),
	}, nil
}

func toGroupChatResponseWithInvitations(pubKey string, response *protocol.MessengerResponse) (*GroupChatResponseWithInvitations, error) {
	g, err := toGroupChatResponse(pubKey, response)
	if err != nil {
		return nil, err
	}

	return &GroupChatResponseWithInvitations{
		Chat:        g.Chat,
		Messages:    g.Messages,
		Invitations: response.Invitations,
	}, nil
}

func (api *API) execAndGetGroupChatResponse(fn func() (*protocol.MessengerResponse, error)) (*GroupChatResponse, error) {
	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))
	response, err := fn()
	if err != nil {
		return nil, err
	}
	return toGroupChatResponse(pubKey, response)
}

func (api *API) execAndGetGroupChatResponseWithInvitations(fn func() (*protocol.MessengerResponse, error)) (*GroupChatResponseWithInvitations, error) {
	pubKey := types.EncodeHex(crypto.FromECDSAPub(api.s.messenger.IdentityPublicKey()))

	response, err := fn()
	if err != nil {
		return nil, err
	}

	return toGroupChatResponseWithInvitations(pubKey, response)
}
