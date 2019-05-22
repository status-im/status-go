package chatapi

import (
	"github.com/status-im/status-go/node"
)

type Type string

const (
	TypePublic       = Type("public")
	TypePrivateGroup = Type("private_group")
	TypeOneOnOne     = Type("one_on_one")
)

type ChatsResponse struct {
	UnreadMessagesCount int        `json:"unread_messages"`
	Chats               []ChatView `json:"chats"`
}

type ChatView struct {
	ID                  string `json:"id"`
	Type                Type   `json:"type"`
	Name                string `json:"name"`
	LastMessageContent  string `json:"last_message"`
	LastMessageSender   string `json:"last_message_sender"`
	UnreadMessagesCount int    `json:"unread_messages"`
}

type API struct {
}

func NewMockAPI(node *node.StatusNode) *API {
	return &API{}
}

func (api *API) Chats(method string, args []interface{}) (ChatsResponse, error) {
	return ChatsResponse{
		UnreadMessagesCount: 30,
		Chats: []ChatView{
			{
				ID:                  "blah1",
				Type:                TypePublic,
				Name:                "#status-fake",
				LastMessageContent:  "well, hello there!",
				LastMessageSender:   "Unreal Fake Imitation",
				UnreadMessagesCount: 20,
			},
			{
				ID:                  "blah-private-group1",
				Type:                TypePrivateGroup,
				Name:                "#status-fake-group",
				LastMessageContent:  "group, hello there!",
				LastMessageSender:   "Unreal Private Group",
				UnreadMessagesCount: 9,
			},
			{
				ID:                  "blah-one-on-one",
				Type:                TypeOneOnOne,
				Name:                "One Single Imitation",
				LastMessageContent:  "Hi there!",
				LastMessageSender:   "One Single Imitation",
				UnreadMessagesCount: 1,
			},
		},
	}, nil
}
