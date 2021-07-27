package signal

const (

	// EventMesssageDelivered triggered when we got acknowledge from datasync level, that means peer got message
	EventMesssageDelivered = "message.delivered"

	// EventCommunityFound triggered when user requested info about some community and messenger successfully
	// retrieved it from mailserver
	EventCommunityInfoFound = "community.found"
)

// MessageDeliveredSignal specifies chat and message that was delivered
type MessageDeliveredSignal struct {
	ChatID    string `json:"chatID"`
	MessageID string `json:"messageID"`
}

// MessageDeliveredSignal specifies chat and message that was delivered
type CommunityInfoFoundSignal struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	MembersCount int    `json:"membersCount"`
	Verified     bool   `json:"verified"`
}

// SendMessageDelivered notifies about delivered message
func SendMessageDelivered(chatID string, messageID string) {
	send(EventMesssageDelivered, MessageDeliveredSignal{ChatID: chatID, MessageID: messageID})
}

// SendMessageDelivered notifies about delivered message
func SendCommunityInfoFound(community interface{}) {
	send(EventCommunityInfoFound, community)
}
