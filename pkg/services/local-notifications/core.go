package localnotifications

import (
	"database/sql"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/signal"
)

type PushCategory string

type NotificationType string

type NotificationBody interface {
	json.Marshaler
}

type Notification struct {
	ID       common.Hash
	Platform float32
	Body     NotificationBody
	BodyType NotificationType
	Title    string
	Message  string
	// DisplayTitle and DisplayMessage are privacy-filtered for lock screen/OS display.
	// When set, clients should use these for OS notifications; Title/Message remain full for in-app.
	DisplayTitle        string
	DisplayMessage      string
	Category            PushCategory
	Deeplink            string
	Image               string
	IsScheduled         bool
	ScheduledTime       string
	IsConversation      bool
	IsGroupConversation bool
	ConversationID      string
	ThreadID            string
	Timestamp           uint64
	Author              NotificationAuthor
	// CommunityIcon is the community avatar (data URI) for community chat/request notifications.
	CommunityIcon string
	// ChatIcon is the chat/group avatar (data URI) for group chat notifications.
	ChatIcon string
	Deleted  bool
	// IsFromMe indicates the message was sent by the current user (outgoing).
	IsFromMe bool
}

type NotificationAuthor struct {
	ID   string `json:"id"`
	Icon string `json:"icon"`
	Name string `json:"name"`
}

// notificationAlias is an interim struct used for json un/marshalling
type notificationAlias struct {
	ID                  common.Hash        `json:"id"`
	Platform            float32            `json:"platform,omitempty"`
	Body                json.RawMessage    `json:"body"`
	BodyType            NotificationType   `json:"bodyType"`
	Title               string             `json:"title,omitempty"`
	Message             string             `json:"message,omitempty"`
	DisplayTitle        string             `json:"displayTitle,omitempty"`
	DisplayMessage      string             `json:"displayMessage,omitempty"`
	Category            PushCategory       `json:"category,omitempty"`
	Deeplink            string             `json:"deepLink,omitempty"`
	Image               string             `json:"imageUrl,omitempty"`
	IsScheduled         bool               `json:"isScheduled,omitempty"`
	ScheduledTime       string             `json:"scheduleTime,omitempty"`
	IsConversation      bool               `json:"isConversation,omitempty"`
	IsGroupConversation bool               `json:"isGroupConversation,omitempty"`
	ConversationID      string             `json:"conversationId,omitempty"`
	ThreadID            string             `json:"threadId,omitempty"`
	Timestamp           uint64             `json:"timestamp,omitempty"`
	Author              NotificationAuthor `json:"notificationAuthor,omitempty"`
	CommunityIcon       string             `json:"communityIcon,omitempty"`
	ChatIcon            string             `json:"chatIcon,omitempty"`
	Deleted             bool               `json:"deleted,omitempty"`
	IsFromMe            bool               `json:"isFromMe,omitempty"`
}

// MessageEvent - structure used to pass messages from chat to bus
type MessageEvent struct{}

// CustomEvent - structure used to pass custom user set messages to bus
type CustomEvent struct{}

// Service keeps the state of message bus
type Service struct {
	started bool
	db      *Database
}

func NewService(appDB *sql.DB) (*Service, error) {
	db := NewDB(appDB)

	return &Service{
		db: db,
	}, nil
}

func (n *Notification) MarshalJSON() ([]byte, error) {

	var body json.RawMessage
	if n.Body != nil {
		encodedBody, err := n.Body.MarshalJSON()
		if err != nil {
			return nil, err
		}
		body = encodedBody
	}

	alias := notificationAlias{
		ID:                  n.ID,
		Platform:            n.Platform,
		Body:                body,
		BodyType:            n.BodyType,
		Category:            n.Category,
		Title:               n.Title,
		Message:             n.Message,
		DisplayTitle:        n.DisplayTitle,
		DisplayMessage:      n.DisplayMessage,
		Deeplink:            n.Deeplink,
		Image:               n.Image,
		IsScheduled:         n.IsScheduled,
		ScheduledTime:       n.ScheduledTime,
		IsConversation:      n.IsConversation,
		IsGroupConversation: n.IsGroupConversation,
		ConversationID:      n.ConversationID,
		ThreadID:            n.ThreadID,
		Timestamp:           n.Timestamp,
		Author:              n.Author,
		CommunityIcon:       n.CommunityIcon,
		ChatIcon:            n.ChatIcon,
		Deleted:             n.Deleted,
		IsFromMe:            n.IsFromMe,
	}

	return json.Marshal(alias)
}

func PushMessages(ns []*Notification) {
	for _, n := range ns {
		pushMessage(n)
	}
}

func pushMessage(notification *Notification) {
	logutils.ZapLogger().Debug("Pushing a new push notification")
	signal.SendLocalNotifications(notification)
}

// Start Worker which processes all incoming messages
func (s *Service) Start() error {
	s.started = true
	logutils.ZapLogger().Info("Successful start")
	return nil
}

// Stop worker
func (s *Service) Stop() error {
	s.started = false
	return nil
}

// APIs returns list of available RPC APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "localnotifications",
			Version:   "0.1.0",
			Service:   NewAPI(s),
		},
	}
}

func (s *Service) IsStarted() bool {
	return s.started
}
