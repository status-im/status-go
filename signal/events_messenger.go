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

	// EventCuratedCommunitiesRefreshStarted triggered when a curated communities refresh begins
	EventCuratedCommunitiesRefreshStarted = "curated.communities.refresh.started"

	// EventCuratedCommunityResolved triggered per curated community once its description resolution completes
	EventCuratedCommunityResolved = "curated.communities.refresh.resolved"

	// EventCuratedCommunitiesRefreshFinished triggered when a curated communities refresh ends
	EventCuratedCommunitiesRefreshFinished = "curated.communities.refresh.finished"
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
	send(EventMesssageDelivered, MessageDeliveredSignal{ChatID: chatID, MessageID: messageID})
}

// SendMediaServerStarted notifies about restarts of the media server
func SendMediaServerStarted(port int) {
	send(EventMediaServerStarted, MediaServerStarted{Port: port})
}

// SendMessageDelivered notifies about delivered message
func SendCommunityInfoFound(community interface{}) {
	send(EventCommunityInfoFound, community)
}

func SendStatusUpdatesTimedOut(statusUpdates interface{}) {
	send(EventStatusUpdatesTimedOut, statusUpdates)
}

// CuratedCommunityResolvedSignal reports a curated community whose description resolution finished
type CuratedCommunityResolvedSignal struct {
	CommunityID string `json:"communityID"`
	Stored      bool   `json:"stored"`
}

// CuratedCommunitiesRefreshFinishedSignal reports the outcome counts of a curated communities refresh
type CuratedCommunitiesRefreshFinishedSignal struct {
	Resolved   int  `json:"resolved"`
	Unresolved int  `json:"unresolved"`
	Cancelled  bool `json:"cancelled"`
}

func SendCuratedCommunitiesRefreshStarted() {
	send(EventCuratedCommunitiesRefreshStarted, nil)
}

func SendCuratedCommunityResolved(communityID string, stored bool) {
	send(EventCuratedCommunityResolved, CuratedCommunityResolvedSignal{CommunityID: communityID, Stored: stored})
}

func SendCuratedCommunitiesRefreshFinished(resolved, unresolved int, cancelled bool) {
	send(EventCuratedCommunitiesRefreshFinished, CuratedCommunitiesRefreshFinishedSignal{
		Resolved:   resolved,
		Unresolved: unresolved,
		Cancelled:  cancelled,
	})
}
