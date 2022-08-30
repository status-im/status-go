package signal

import (
	"github.com/status-im/status-go/protocol/discord"
)

const (

	// EventDiscordCategoriesAndChannelsExtracted triggered when categories and
	// channels for exported discord files have been successfully extracted
	EventDiscordCategoriesAndChannelsExtracted = "community.discordCategoriesAndChannelsExtracted"

	// EventDiscordCommunityImportProgress is triggered during the import
	// of a discord community as it progresses
	EventDiscordCommunityImportProgress = "community.discordCommunityImportProgress"

	// EventDiscordCommunityImportFinished triggered when importing
	// the discord community into status was successful
	EventDiscordCommunityImportFinished = "community.discordCommunityImportFinished"

	// EventDiscordCommunityImportCancelled triggered when importing
	// the discord community was cancelled
	EventDiscordCommunityImportCancelled = "community.discordCommunityImportCancelled"
)

type DiscordCategoriesAndChannelsExtractedSignal struct {
	Categories             []*discord.Category             `json:"discordCategories"`
	Channels               []*discord.Channel              `json:"discordChannels"`
	OldestMessageTimestamp int64                           `json:"oldestMessageTimestamp"`
	Errors                 map[string]*discord.ImportError `json:"errors"`
}

type DiscordCommunityImportProgressSignal struct {
	ImportProgress *discord.ImportProgress `json:"importProgress"`
}

type DiscordCommunityImportFinishedSignal struct {
	CommunityID string `json:"communityId"`
}

type DiscordCommunityImportCancelledSignal struct {
	CommunityID string `json:"communityId"`
}

func SendDiscordCategoriesAndChannelsExtracted(categories []*discord.Category, channels []*discord.Channel, oldestMessageTimestamp int64, errors map[string]*discord.ImportError) {
	send(EventDiscordCategoriesAndChannelsExtracted, DiscordCategoriesAndChannelsExtractedSignal{
		Categories:             categories,
		Channels:               channels,
		OldestMessageTimestamp: oldestMessageTimestamp,
		Errors:                 errors,
	})
}

func SendDiscordCommunityImportProgress(importProgress *discord.ImportProgress) {
	send(EventDiscordCommunityImportProgress, DiscordCommunityImportProgressSignal{
		ImportProgress: importProgress,
	})
}

func SendDiscordCommunityImportFinished(communityID string) {
	send(EventDiscordCommunityImportFinished, DiscordCommunityImportFinishedSignal{
		CommunityID: communityID,
	})
}

func SendDiscordCommunityImportCancelled(communityID string) {
	send(EventDiscordCommunityImportCancelled, DiscordCommunityImportCancelledSignal{
		CommunityID: communityID,
	})
}
