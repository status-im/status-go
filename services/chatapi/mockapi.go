package chatapi

import (
	"fmt"

	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/signal"
)

type ChatsResponse struct {
	UnreadMessagesCount int                 `json:"unviewed-messages-count"`
	Chats               map[string]ChatView `json:"chats"`
}

type ChatView struct {
	ID                     string            `json:"chat-id"`
	Name                   string            `json:"name"`
	ColorHex               string            `json:"color"`
	LastMessageContent     map[string]string `json:"last-message-content"`
	LastMessageContentType string            `json:"last-message-content-type"`
	UnreadMessagesCount    int               `json:"unviewed-messages-count"`
	IsActive               bool              `json:"is-active"`
	IsGroupChat            bool              `json:"group-chat"`
	IsPublic               bool              `json:"public?"`
}

type API struct {
	cs map[string]ChatView
}

func NewMockAPI(node *node.StatusNode) *API {
	return &API{
		cs: map[string]ChatView{
			"status": {
				ID:                     "status",
				Name:                   "status",
				ColorHex:               "#000000",
				IsActive:               true,
				LastMessageContentType: "text/plain",
				LastMessageContent:     map[string]string{"text": "still fake but real ID"},
				UnreadMessagesCount:    1,
				IsGroupChat:            true,
				IsPublic:               true,
			},
			"status-fake": {
				ID:                     "status-fake",
				Name:                   "status-fake",
				ColorHex:               "#51d0f0",
				IsActive:               true,
				LastMessageContentType: "text/plain",
				LastMessageContent:     map[string]string{"text": "well, hello there!"},
				UnreadMessagesCount:    20,
				IsGroupChat:            true,
				IsPublic:               true,
			},
			"status-fake-group": {
				ID:                     "status-fake-group",
				Name:                   "status-fake-group",
				ColorHex:               "#51d0f0",
				IsActive:               true,
				LastMessageContentType: "text/plain",
				LastMessageContent:     map[string]string{"text": "private-group-chat!"},
				UnreadMessagesCount:    9,
				IsGroupChat:            true,
				IsPublic:               false,
			},
			"blah-one-on-one": {
				ID:                     "blah-one-on-one",
				Name:                   "One Single Imitation",
				ColorHex:               "#51d0f0",
				IsActive:               true,
				LastMessageContentType: "text/plain",
				LastMessageContent:     map[string]string{"text": "one-on-one!"},
				UnreadMessagesCount:    1,
				IsGroupChat:            false,
				IsPublic:               false,
			},
		},
	}
}

func (api *API) Chats() (ChatsResponse, error) {
	return ChatsResponse{
		UnreadMessagesCount: 30,
		Chats:               api.cs,
	}, nil
}

func (api *API) JoinPublicChat(name string) error {
	api.cs[name] = ChatView{
		ID:                     name,
		Name:                   name,
		ColorHex:               "#abcabc",
		IsActive:               true,
		LastMessageContentType: "text/plain",
		LastMessageContent:     map[string]string{"text": fmt.Sprintf("you created %s!", name)},
		UnreadMessagesCount:    1,
		IsGroupChat:            true,
		IsPublic:               true,
	}
	api.sendChatsUpdatedSignal(name)

	return nil
}

func (api *API) JoinPrivateGroupChat(name string, participants []string) error {
	api.cs[name] = ChatView{
		ID:                     name,
		Name:                   name,
		ColorHex:               "#abcabc",
		IsActive:               true,
		LastMessageContentType: "text/plain",
		LastMessageContent:     map[string]string{"text": fmt.Sprintf("%s -> %d participants!", name, len(participants))},
		UnreadMessagesCount:    1,
		IsGroupChat:            true,
		IsPublic:               false,
	}
	api.sendChatsUpdatedSignal(name)

	return nil
}

func (api *API) StartOneOnOneChat(recipient string) error {
	api.cs[recipient] = ChatView{
		ID:                     recipient,
		Name:                   recipient,
		ColorHex:               "#abcabc",
		IsActive:               true,
		LastMessageContentType: "text/plain",
		LastMessageContent:     map[string]string{"text": fmt.Sprintf("you created %s!", recipient)},
		UnreadMessagesCount:    1,
		IsGroupChat:            false,
		IsPublic:               false,
	}
	api.sendChatsUpdatedSignal(recipient)

	return nil
}

func (api *API) sendChatsUpdatedSignal(name string) {
	signal.SendChatsDidChangeEvent(name)
}

// TODO: a signal

/*

{"status" {:updated-at nil, :tags #{}, :referenced-messages {}, :color "#51d0f0", :contacts #{}, :last-clock-value 156035060894207, :admins #{}, :members-joined #{}, :name "status", :removed-from-at nil, :membership-updates (), :unviewed-messages-count 1, :last-message-content-type "text/plain", :is-active true, :last-message-content {:chat-id "status", :text "Please take a look at this Youtube playlist to learn how to use Status\n\nhttps://www.youtube.com/watch?v=fnuqRV37JmE&list=PLbrz7IuP1hrinFXb47zmukKPbsB-1_Uhx", :response-to "0xe72e84ddb55d4c499f1412aa997b865a0bd6303610ee7dd00254181f0e14f06e", :response-to-v2 "0xb0366c4ef191e12be89030b299694ec5c89c77efa2ef93de13c40c356a7b2a74", :metadata {:link ([72 155])}, :render-recipe [["Please take a look at this Youtube playlist to learn how to use Status\n\n" :text] ["https://www.youtube.com/watch?v=fnuqRV37JmE&list=PLbrz7IuP1hrinFXb47zmukKPbsB-1_Uhx" :link]]}, :messages #status-im.utils.prio<…>

*/
