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

	// EventCuratedCommunitiesRefreshStarted triggered when a curated communities refresh has its directory
	// and begins to fetch its entries. It tells whether the directory changed and whether updating it failed
	EventCuratedCommunitiesRefreshStarted = "curated.communities.refresh.started"

	// EventCuratedCommunitiesRefreshFinished triggered when a curated communities refresh ends.
	// It counts the outcomes of the entries the refresh had to fetch
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

// CuratedCommunitiesRefreshStartedSignal reports that a curated communities refresh has its directory.
// Changed tells whether the directory read from the contract differs from the one stored before the refresh.
// DirectoryFailed tells that the contract could not be read or its directory could not be stored:
// the directory stored earlier is used, if it can be read, and Changed is false.
// RunID is the ID of the run, as returned by the refresh call that started or joined it.
type CuratedCommunitiesRefreshStartedSignal struct {
	RunID           uint64 `json:"runId"`
	Changed         bool   `json:"changed"`
	DirectoryFailed bool   `json:"directoryFailed"`
}

// CuratedCommunitiesRefreshFinishedSignal reports the outcome counts of a curated communities refresh.
// Only the communities the refresh had to fetch are counted, each in one count at most.
// Resolved: fetched and now stored. NotFound: store nodes were asked without error and it is still
// not stored, or its ID is malformed. Failed: the fetch returned an error, also when store nodes
// may not be asked on this connection. An entry interrupted by the cancellation is in none of them.
// RunID is the ID of the run, the same as in its started signal.
type CuratedCommunitiesRefreshFinishedSignal struct {
	RunID     uint64 `json:"runId"`
	Resolved  int    `json:"resolved"`
	NotFound  int    `json:"notFound"`
	Failed    int    `json:"failed"`
	Cancelled bool   `json:"cancelled"`
}

func SendCuratedCommunitiesRefreshStarted(runID uint64, changed, directoryFailed bool) {
	CurrentSession().SendCuratedCommunitiesRefreshStarted(runID, changed, directoryFailed)
}

// SendCuratedCommunitiesRefreshStarted sends in the session s.
func (s Session) SendCuratedCommunitiesRefreshStarted(runID uint64, changed, directoryFailed bool) {
	s.send(EventCuratedCommunitiesRefreshStarted, CuratedCommunitiesRefreshStartedSignal{
		RunID:           runID,
		Changed:         changed,
		DirectoryFailed: directoryFailed,
	})
}

func SendCuratedCommunitiesRefreshFinished(runID uint64, resolved, notFound, failed int, cancelled bool) {
	CurrentSession().SendCuratedCommunitiesRefreshFinished(runID, resolved, notFound, failed, cancelled)
}

// SendCuratedCommunitiesRefreshFinished sends in the session s.
func (s Session) SendCuratedCommunitiesRefreshFinished(runID uint64, resolved, notFound, failed int, cancelled bool) {
	s.send(EventCuratedCommunitiesRefreshFinished, CuratedCommunitiesRefreshFinishedSignal{
		RunID:     runID,
		Resolved:  resolved,
		NotFound:  notFound,
		Failed:    failed,
		Cancelled: cancelled,
	})
}
