package signal

const (

	// EventHistoryArchivesEnabled triggered when the community history archive protocol
	// was enabled via the RPC API
	EventHistoryArchivesProtocolEnabled = "community.historyArchivesProtocolEnabled"
	// EventHistoryArchivesDisabled triggered when the community history archive protocol
	// was disabled via the RPC API
	EventHistoryArchivesProtocolDisabled = "community.historyArchivesProtocolDisabled"
	// EventCreatingHistoryArchives is triggered when the community owner node
	// starts to create archives torrents
	EventCreatingHistoryArchives = "community.creatingHistoryArchives"
	// EventHistoryArchivesCreated is triggered when the community owner node
	// has finished to create archives torrents
	EventHistoryArchivesCreated = "community.historyArchivesCreated"
	// EventNoHistoryArchivesCreated is triggered when the community owner node
	// tried to create archives but haven't because there were no new messages
	// to archive
	EventNoHistoryArchivesCreated = "community.noHistoryArchivesCreated"
	// EventHistoryArchivesSeeding is triggered when the community owner node
	// started seeding archives torrents
	EventHistoryArchivesSeeding = "community.historyArchivesSeeding"
	// EventHistoryArchivesUnseeded is triggered when the community owner node
	// drops a torrent for a particular community
	EventHistoryArchivesUnseeded = "community.historyArchivesUnseeded"
	// EventDownloadingHistoryArchivesFinished is triggered when the community member node
	// has downloaded all archives
	EventDownloadingHistoryArchivesStarted = "community.downloadingHistoryArchivesStarted"
	// EventHistoryArchiveDownloaded is triggered when the community member node
	// has downloaded an individual community archive
	EventHistoryArchiveDownloaded = "community.historyArchiveDownloaded"
	// EventImportingHistoryArchiveMessages is triggered when the community member node
	// has starts importing downloaded archive messages into the database
	EventImportingHistoryArchiveMessages = "community.importingHistoryArchiveMessages"
	// EventDownloadingHistoryArchivesFinished is triggered when the community member node
	// has downloaded all archives
	EventDownloadingHistoryArchivesFinished = "community.downloadingHistoryArchivesFinished"
	// EventIndexDownloadCompleted is triggered when the community member node
	// has completed downloading the archive index file
	EventIndexDownloadCompleted = "community.indexDownloadCompleted"
)

type CreatingHistoryArchivesSignal struct {
	CommunityID string `json:"communityId"`
}

type NoHistoryArchivesCreatedSignal struct {
	CommunityID string `json:"communityId"`
	From        int    `json:"from"`
	To          int    `json:"to"`
}

type HistoryArchivesCreatedSignal struct {
	CommunityID string `json:"communityId"`
	From        int    `json:"from"`
	To          int    `json:"to"`
}

type HistoryArchivesSeedingSignal struct {
	CommunityID string `json:"communityId"`
}

type HistoryArchivesUnseededSignal struct {
	CommunityID string `json:"communityId"`
}

type HistoryArchiveDownloadedSignal struct {
	CommunityID string `json:"communityId"`
	From        int    `json:"from"`
	To          int    `json:"to"`
}

type DownloadingHistoryArchivesStartedSignal struct {
	CommunityID string `json:"communityId"`
}

type ImportingHistoryArchiveMessagesSignal struct {
	CommunityID string `json:"communityId"`
}

type DownloadingHistoryArchivesFinishedSignal struct {
	CommunityID string `json:"communityId"`
}

type IndexDownloadCompletedSignal struct {
	CommunityID string `json:"communityId"`
	IndexCid    string `json:"indexCid"`
}

func SendHistoryArchivesProtocolEnabled() {
	CurrentSession().SendHistoryArchivesProtocolEnabled()
}

// SendHistoryArchivesProtocolEnabled sends in the session s.
func (s Session) SendHistoryArchivesProtocolEnabled() {
	s.send(EventHistoryArchivesProtocolEnabled, nil)
}

func SendHistoryArchivesProtocolDisabled() {
	CurrentSession().SendHistoryArchivesProtocolDisabled()
}

// SendHistoryArchivesProtocolDisabled sends in the session s.
func (s Session) SendHistoryArchivesProtocolDisabled() {
	s.send(EventHistoryArchivesProtocolDisabled, nil)
}

func SendCreatingHistoryArchives(communityID string) {
	CurrentSession().SendCreatingHistoryArchives(communityID)
}

// SendCreatingHistoryArchives sends in the session s.
func (s Session) SendCreatingHistoryArchives(communityID string) {
	s.send(EventCreatingHistoryArchives, CreatingHistoryArchivesSignal{CommunityID: communityID})
}

func SendNoHistoryArchivesCreated(communityID string, from int, to int) {
	CurrentSession().SendNoHistoryArchivesCreated(communityID, from, to)
}

// SendNoHistoryArchivesCreated sends in the session s.
func (s Session) SendNoHistoryArchivesCreated(communityID string, from int, to int) {
	s.send(EventNoHistoryArchivesCreated, NoHistoryArchivesCreatedSignal{
		CommunityID: communityID,
		From:        from,
		To:          to,
	})
}

func SendHistoryArchivesCreated(communityID string, from int, to int) {
	CurrentSession().SendHistoryArchivesCreated(communityID, from, to)
}

// SendHistoryArchivesCreated sends in the session s.
func (s Session) SendHistoryArchivesCreated(communityID string, from int, to int) {
	s.send(EventHistoryArchivesCreated, HistoryArchivesCreatedSignal{
		CommunityID: communityID,
		From:        from,
		To:          to,
	})
}

func SendHistoryArchivesSeeding(communityID string) {
	CurrentSession().SendHistoryArchivesSeeding(communityID)
}

// SendHistoryArchivesSeeding sends in the session s.
func (s Session) SendHistoryArchivesSeeding(communityID string) {
	s.send(EventHistoryArchivesSeeding, HistoryArchivesSeedingSignal{CommunityID: communityID})
}

func SendHistoryArchivesUnseeded(communityID string) {
	CurrentSession().SendHistoryArchivesUnseeded(communityID)
}

// SendHistoryArchivesUnseeded sends in the session s.
func (s Session) SendHistoryArchivesUnseeded(communityID string) {
	s.send(EventHistoryArchivesUnseeded, HistoryArchivesUnseededSignal{CommunityID: communityID})
}

func SendHistoryArchiveDownloaded(communityID string, from int, to int) {
	CurrentSession().SendHistoryArchiveDownloaded(communityID, from, to)
}

// SendHistoryArchiveDownloaded sends in the session s.
func (s Session) SendHistoryArchiveDownloaded(communityID string, from int, to int) {
	s.send(EventHistoryArchiveDownloaded, HistoryArchiveDownloadedSignal{
		CommunityID: communityID,
		From:        from,
		To:          to,
	})
}

func SendDownloadingHistoryArchivesStarted(communityID string) {
	CurrentSession().SendDownloadingHistoryArchivesStarted(communityID)
}

// SendDownloadingHistoryArchivesStarted sends in the session s.
func (s Session) SendDownloadingHistoryArchivesStarted(communityID string) {
	s.send(EventDownloadingHistoryArchivesStarted, DownloadingHistoryArchivesStartedSignal{
		CommunityID: communityID,
	})
}

func SendImportingHistoryArchiveMessages(communityID string) {
	CurrentSession().SendImportingHistoryArchiveMessages(communityID)
}

// SendImportingHistoryArchiveMessages sends in the session s.
func (s Session) SendImportingHistoryArchiveMessages(communityID string) {
	s.send(EventImportingHistoryArchiveMessages, ImportingHistoryArchiveMessagesSignal{
		CommunityID: communityID,
	})
}

func SendDownloadingHistoryArchivesFinished(communityID string) {
	CurrentSession().SendDownloadingHistoryArchivesFinished(communityID)
}

// SendDownloadingHistoryArchivesFinished sends in the session s.
func (s Session) SendDownloadingHistoryArchivesFinished(communityID string) {
	s.send(EventDownloadingHistoryArchivesFinished, DownloadingHistoryArchivesFinishedSignal{
		CommunityID: communityID,
	})
}

func SendIndexDownloadCompleted(communityID string, indexCid string) {
	CurrentSession().SendIndexDownloadCompleted(communityID, indexCid)
}

// SendIndexDownloadCompleted sends in the session s.
func (s Session) SendIndexDownloadCompleted(communityID string, indexCid string) {
	s.send(EventIndexDownloadCompleted, IndexDownloadCompletedSignal{
		CommunityID: communityID,
		IndexCid:    indexCid,
	})
}
