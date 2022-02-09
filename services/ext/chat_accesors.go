package ext

import (
	"context"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

var (
	ErrChatNotFound = errors.New("can't find chat")
)

type ChannelGroupType string

const Personal ChannelGroupType = "personal"
const Community ChannelGroupType = "community"

type ChatPinnedMessages struct {
	Cursor         string
	PinnedMessages []*common.PinnedMessage
}

type ChatMember struct {
	// Community Roles
	Roles []protobuf.CommunityMember_Roles `json:"roles,omitempty"`
	// Admin indicates if the member is an admin of the group chat
	Admin bool `json:"admin"`
	// Joined indicates if the member has joined the group chat
	Joined bool `json:"joined"`
}

type Chat struct {
	ID                       string                             `json:"id"`
	Name                     string                             `json:"name"`
	Description              string                             `json:"description"`
	Color                    string                             `json:"color"`
	Emoji                    string                             `json:"emoji"`
	Active                   bool                               `json:"active"`
	ChatType                 protocol.ChatType                  `json:"chatType"`
	Timestamp                int64                              `json:"timestamp"`
	LastClockValue           uint64                             `json:"lastClockValue"`
	DeletedAtClockValue      uint64                             `json:"deletedAtClockValue"`
	ReadMessagesAtClockValue uint64                             `json:"readMessagesAtClockValue"`
	UnviewedMessagesCount    uint                               `json:"unviewedMessagesCount"`
	UnviewedMentionsCount    uint                               `json:"unviewedMentionsCount"`
	LastMessage              *common.Message                    `json:"lastMessage"`
	Members                  map[string]ChatMember              `json:"members,omitempty"`
	MembershipUpdates        []v1protocol.MembershipUpdateEvent `json:"membershipUpdateEvents"`
	Alias                    string                             `json:"alias,omitempty"`
	Identicon                string                             `json:"identicon"`
	Muted                    bool                               `json:"muted"`
	InvitationAdmin          string                             `json:"invitationAdmin,omitempty"`
	ReceivedInvitationAdmin  string                             `json:"receivedInvitationAdmin,omitempty"`
	Profile                  string                             `json:"profile,omitempty"`
	CommunityID              string                             `json:"communityId,omitempty"`
	CategoryID               string                             `json:"categoryId"`
	Position                 int32                              `json:"position,omitempty"`
	Permissions              *protobuf.CommunityPermissions     `json:"permissions,omitempty"`
	Joined                   int64                              `json:"joined,omitempty"`
	SyncedTo                 uint32                             `json:"syncedTo,omitempty"`
	SyncedFrom               uint32                             `json:"syncedFrom,omitempty"`
	Highlight                bool                               `json:"highlight,omitempty"`
	PinnedMessages           *ChatPinnedMessages                `json:"pinnedMessages,omitempty"`
	CanPost                  bool                               `json:"canPost"`
}

type ChannelGroup struct {
	Type       ChannelGroupType                         `json:"channelGroupType"`
	Name       string                                   `json:"name"`
	Images     map[string]images.IdentityImage          `json:"images"`
	Color      string                                   `json:"color"`
	Chats      map[string]*Chat                         `json:"chats"`
	Categories map[string]communities.CommunityCategory `json:"categories"`
	EnsName    string                                   `json:"ensName"`
}

func (api *PublicAPI) GetChats(parent context.Context) (map[string]ChannelGroup, error) {
	joinedCommunities, err := api.service.messenger.JoinedCommunities()
	if err != nil {
		return nil, err
	}

	channels := api.service.messenger.Chats()

	pubKey, err := api.service.accountsDB.GetPublicKey()
	if err != nil {
		return nil, err
	}

	result := make(map[string]ChannelGroup)

	result[pubKey] = ChannelGroup{
		Type:       Personal,
		Name:       "",
		Images:     make(map[string]images.IdentityImage),
		Color:      "",
		Chats:      make(map[string]*Chat),
		Categories: make(map[string]communities.CommunityCategory),
		EnsName:    "", // Not implemented yet in communities
	}

	for _, chat := range channels {
		if !chat.Active || (!chat.OneToOne() && !chat.PrivateGroupChat() && !chat.Public()) || chat.CommunityID != "" {
			continue
		}

		pinnedMessages, cursor, err := api.service.messenger.PinnedMessageByChatID(chat.ID, "", -1)
		if err != nil {
			return nil, err
		}

		c, err := toAPIChat(chat, nil, pubKey, pinnedMessages, cursor)
		if err != nil {
			return nil, err
		}
		result[pubKey].Chats[chat.ID] = c
	}

	for _, community := range joinedCommunities {
		chGrp := ChannelGroup{
			Type:       Community,
			Name:       community.Name(),
			Color:      community.Description().Identity.Color,
			Images:     make(map[string]images.IdentityImage),
			Chats:      make(map[string]*Chat),
			Categories: make(map[string]communities.CommunityCategory),
		}

		for t, i := range community.Description().Identity.Images {
			chGrp.Images[t] = images.IdentityImage{Name: t, Payload: i.Payload}
		}

		for _, cat := range community.Description().Categories {
			chGrp.Categories[cat.CategoryId] = communities.CommunityCategory{
				ID:       cat.CategoryId,
				Name:     cat.Name,
				Position: int(cat.Position),
			}
		}

		for _, chat := range channels {
			if chat.CommunityID == community.IDString() {
				pinnedMessages, cursor, err := api.service.messenger.PinnedMessageByChatID(chat.ID, "", -1)
				if err != nil {
					return nil, err
				}

				c, err := toAPIChat(chat, community, pubKey, pinnedMessages, cursor)
				if err != nil {
					return nil, err
				}

				chGrp.Chats[c.ID] = c
			}
		}

		result[community.IDString()] = chGrp
	}

	return result, nil
}

func (api *PublicAPI) GetChat(parent context.Context, communityID types.HexBytes, chatID string) (*Chat, error) {
	fullChatID := chatID

	if len(communityID) != 0 {
		fullChatID = string(communityID.Bytes()) + chatID
	}

	messengerChat := api.service.messenger.Chat(fullChatID)
	if messengerChat == nil {
		return nil, ErrChatNotFound
	}

	pubKey, err := api.service.accountsDB.GetPublicKey()
	if err != nil {
		return nil, err
	}

	var community *communities.Community
	if messengerChat.CommunityID != "" {
		communityID, err := hexutil.Decode(messengerChat.CommunityID)
		if err != nil {
			return nil, err
		}

		community, err = api.service.messenger.GetCommunityByID(communityID)
		if err != nil {
			return nil, err
		}
	}

	pinnedMessages, cursor, err := api.service.messenger.PinnedMessageByChatID(messengerChat.ID, "", -1)
	if err != nil {
		return nil, err
	}

	result, err := toAPIChat(messengerChat, community, pubKey, pinnedMessages, cursor)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func toAPIChat(protocolChat *protocol.Chat, community *communities.Community, pubKey string, pinnedMessages []*common.PinnedMessage, cursor string) (*Chat, error) {
	chat := &Chat{
		ID:                       strings.TrimPrefix(protocolChat.ID, protocolChat.CommunityID),
		Name:                     protocolChat.Name,
		Description:              protocolChat.Description,
		Color:                    protocolChat.Color,
		Emoji:                    protocolChat.Emoji,
		Active:                   protocolChat.Active,
		ChatType:                 protocolChat.ChatType,
		Timestamp:                protocolChat.Timestamp,
		LastClockValue:           protocolChat.LastClockValue,
		DeletedAtClockValue:      protocolChat.DeletedAtClockValue,
		ReadMessagesAtClockValue: protocolChat.ReadMessagesAtClockValue,
		UnviewedMessagesCount:    protocolChat.UnviewedMessagesCount,
		UnviewedMentionsCount:    protocolChat.UnviewedMentionsCount,
		LastMessage:              protocolChat.LastMessage,
		Members:                  make(map[string]ChatMember),
		MembershipUpdates:        protocolChat.MembershipUpdates,
		Alias:                    protocolChat.Alias,
		Identicon:                protocolChat.Identicon,
		Muted:                    protocolChat.Muted,
		InvitationAdmin:          protocolChat.InvitationAdmin,
		ReceivedInvitationAdmin:  protocolChat.ReceivedInvitationAdmin,
		Profile:                  protocolChat.Profile,
		CommunityID:              protocolChat.CommunityID,
		CategoryID:               protocolChat.CategoryID,
		Joined:                   protocolChat.Joined,
		SyncedTo:                 protocolChat.SyncedTo,
		SyncedFrom:               protocolChat.SyncedFrom,
		Highlight:                protocolChat.Highlight,
	}

	if protocolChat.OneToOne() {
		chat.Name = "" // Emptying since it contains non useful data
	}

	if len(pinnedMessages) != 0 {
		chat.PinnedMessages = &ChatPinnedMessages{
			Cursor:         cursor,
			PinnedMessages: pinnedMessages,
		}
	}

	err := chat.populateCommunityFields(community)
	if err != nil {
		return nil, err
	}

	chat.setChatMembers(protocolChat, community, pubKey)

	return chat, nil
}

func (chat *Chat) setChatMembers(sourceChat *protocol.Chat, community *communities.Community, userPubKey string) {
	if sourceChat.ChatType == protocol.ChatTypePrivateGroupChat && len(sourceChat.Members) > 0 {
		for _, m := range sourceChat.Members {
			chat.Members[m.ID] = ChatMember{
				Admin:  m.Admin,
				Joined: m.Joined,
			}
		}
		return
	}

	if sourceChat.ChatType == protocol.ChatTypeOneToOne {
		chat.Members[sourceChat.ID] = ChatMember{
			Joined: true,
		}
		chat.Members[userPubKey] = ChatMember{
			Joined: true,
		}
		return
	}

	if community != nil {
		for pubKey, m := range community.Description().Members {
			if pubKey == userPubKey {
				chat.Members[pubKey] = ChatMember{
					Roles:  m.Roles,
					Joined: true,
				}
			} else {
				chat.Members[pubKey] = ChatMember{
					Roles:  m.Roles,
					Joined: community.Joined(),
				}
			}
		}
		return
	}
}

func (chat *Chat) populateCommunityFields(community *communities.Community) error {
	commChat, exists := community.Chats()[chat.ID]
	if !exists {
		return ErrChatNotFound
	}

	canPost, err := community.CanMemberIdentityPost(chat.ID)
	if err != nil {
		return err
	}

	chat.CategoryID = commChat.CategoryId
	chat.Position = commChat.Position
	chat.Permissions = commChat.Permissions
	chat.Emoji = commChat.Identity.Emoji
	chat.Name = commChat.Identity.DisplayName
	chat.Color = commChat.Identity.Color
	chat.Description = commChat.Identity.Description
	chat.CanPost = canPost

	return nil
}
