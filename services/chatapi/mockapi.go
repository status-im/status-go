package chatapi

import (
	"fmt"

	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/signal"
)

/*
* This API part consists of a few API methods and one signal
* signal is called every time when a list of chat is updated, no matter
* if a chat is removed or added or changed.
 */

type ChatsResponse struct {
	// TOTAL unviewed message count (for "home" tab)
	UnreadMessagesCount int `json:"unviewed-messages-count"`
	// Dict of chats for now (in the future maybe it should be replaces with something like ordered dict)
	Chats map[string]ChatView `json:"chats"`
}

// A single chat representation as status-react expects
type ChatView struct {
	// chat ID is the chat topic in case of public chats or
	// the recipient key if that is 1-1 chat. no group chats are supported now.
	ID string `json:"chat-id"`
	// something like "status" for #status and empty for 1-1 chat (clojure generates 3-word name)
	Name string `json:"name"` // empty for 1-1 chats
	// chat color (generated based on ID)
	ColorHex string `json:"color"`
	// properties of the last received message in the chatroom

	// content is a dict, because extensions and emoji are treated separately
	LastMessageContent map[string]string `json:"last-message-content"`

	// text or emoji or sticker or etc
	LastMessageContentType string `json:"last-message-content-type"`

	// just a clock value
	LastMessageClock int `json:"last-clock-value"` // 0 if no messages for this chat

	// number of unread messages in this chat
	UnreadMessagesCount int `json:"unviewed-messages-count"`

	// can always be "true" for go-based chat
	IsActive bool `json:"is-active"`

	// "true" for public and group chats
	IsGroupChat bool `json:"group-chat"`

	// "true" for public chats
	IsPublic bool `json:"public?"`

	// when the chat was last updated (new message came) (or created), local time!
	Timestamp int `json:"timestamp"`
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
				Name:                   "",
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

func (api *API) JoinPrivateGroupChat(id string, name string, admin string, participants []string) error {
	api.cs[name] = ChatView{
		ID:                     id,
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
		Name:                   "",
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

func (api *API) RemoveChat(id string) error {
	delete(api.cs, id)
	api.sendChatsUpdatedSignal(id)
	return nil
}

func (api *API) sendChatsUpdatedSignal(name string) {
	signal.SendChatsDidChangeEvent(name)
}
