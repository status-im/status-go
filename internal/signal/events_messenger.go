package signal

const (
	// EventMediaServerStarted triggers when the media server successfully binds a new port
	EventMediaServerStarted = "mediaserver.started"

	// EventMesssageDelivered triggered when we got acknowledge from datasync level, that means peer got message
	EventMesssageDelivered = "message.delivered"

	// EventCommunityInfoFound triggered when user requested info about some community and messenger successfully
	// retrieved it from mailserver
	EventCommunityInfoFound = "community.found"

	// EventStatusUpdatesTimedOut Event Automatic Status Updates Timed out
	EventStatusUpdatesTimedOut = "status.updates.timedout"

	// EventCuratedCommunitiesUpdate triggered when it is time to refresh the list of curated communities
	EventCuratedCommunitiesUpdate = "curated.communities.update"
)

// MessageDeliveredSignal specifies chat and message that was delivered
type MessageDeliveredSignal struct {
	ChatID    string `json:"chatID"`
	MessageID string `json:"messageID"`
}

// MediaServerStarted specifies chat and message that was delivered
type MediaServerStarted struct {
	Port int `json:"port"`
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
	CurrentSession().SendMessageDelivered(chatID, messageID)
}

// SendMessageDelivered sends in the session s.
func (s Session) SendMessageDelivered(chatID string, messageID string) {
	s.send(EventMesssageDelivered, MessageDeliveredSignal{ChatID: chatID, MessageID: messageID})
}

// SendMediaServerStarted notifies about restarts of the media server
func SendMediaServerStarted(port int) {
	CurrentSession().SendMediaServerStarted(port)
}

// SendMediaServerStarted sends in the session s.
func (s Session) SendMediaServerStarted(port int) {
	s.send(EventMediaServerStarted, MediaServerStarted{Port: port})
}

// SendMessageDelivered notifies about delivered message
func SendCommunityInfoFound(community interface{}) {
	CurrentSession().SendCommunityInfoFound(community)
}

// SendCommunityInfoFound sends in the session s.
func (s Session) SendCommunityInfoFound(community interface{}) {
	s.send(EventCommunityInfoFound, community)
}

func SendStatusUpdatesTimedOut(statusUpdates interface{}) {
	CurrentSession().SendStatusUpdatesTimedOut(statusUpdates)
}

// SendStatusUpdatesTimedOut sends in the session s.
func (s Session) SendStatusUpdatesTimedOut(statusUpdates interface{}) {
	s.send(EventStatusUpdatesTimedOut, statusUpdates)
}

func SendCuratedCommunitiesUpdate(curatedCommunitiesUpdate interface{}) {
	CurrentSession().SendCuratedCommunitiesUpdate(curatedCommunitiesUpdate)
}

// SendCuratedCommunitiesUpdate sends in the session s.
func (s Session) SendCuratedCommunitiesUpdate(curatedCommunitiesUpdate interface{}) {
	s.send(EventCuratedCommunitiesUpdate, curatedCommunitiesUpdate)
}
