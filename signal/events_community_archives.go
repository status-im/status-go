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
	// EventHistoryArchiveDownloaded is triggered when the community member node
	// has downloaded an individual community archive
	EventHistoryArchiveDownloaded = "community.historyArchiveDownloaded"
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

func SendHistoryArchivesProtocolEnabled() {
	send(EventHistoryArchivesProtocolEnabled, nil)
}

func SendHistoryArchivesProtocolDisabled() {
	send(EventHistoryArchivesProtocolDisabled, nil)
}

func SendCreatingHistoryArchives(communityID string) {
	send(EventCreatingHistoryArchives, CreatingHistoryArchivesSignal{CommunityID: communityID})
}

func SendNoHistoryArchivesCreated(communityID string, from int, to int) {
	send(EventNoHistoryArchivesCreated, NoHistoryArchivesCreatedSignal{
		CommunityID: communityID,
		From:        from,
		To:          to,
	})
}

func SendHistoryArchivesCreated(communityID string, from int, to int) {
	send(EventHistoryArchivesCreated, HistoryArchivesCreatedSignal{
		CommunityID: communityID,
		From:        from,
		To:          to,
	})
}

func SendHistoryArchivesSeeding(communityID string) {
	send(EventHistoryArchivesSeeding, HistoryArchivesSeedingSignal{CommunityID: communityID})
}

func SendHistoryArchivesUnseeded(communityID string) {
	send(EventHistoryArchivesUnseeded, HistoryArchivesUnseededSignal{CommunityID: communityID})
}

func SendHistoryArchiveDownloaded(communityID string, from int, to int) {
	send(EventHistoryArchiveDownloaded, HistoryArchiveDownloadedSignal{
		CommunityID: communityID,
		From:        from,
		To:          to,
	})
}
