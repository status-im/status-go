package signal

import (
	"github.com/status-im/status-go/protocol/discord"
)

const (

	// EventDiscordCategoriesAndChannelsExtracted triggered when categories and
	// channels for exported discord files have been successfully extracted
	EventDiscordCategoriesAndChannelsExtracted = "community.discordCategoriesAndChannelsExtracted"

	// EventExtractDiscordCategoriesAndChannelsFailed triggered when extraction of
	// categories and channels from exported discord files have failed
	EventExtractDiscordCategoriesAndChannelsFailed = "community.extractDiscordCategoriesAndChannelsFailed"
)

type DiscordCategoriesAndChannelsExtractedSignal struct {
	Categories             []*discord.Category `json:"discordCategories"`
	Channels               []*discord.Channel  `json:"discordChannels"`
	OldestMessageTimestamp int64               `json:"oldestMessageTimestamp"`
}

type ExtractDiscordCategoriesAndChannelsFailedSignal struct{}

func SendDiscordCategoriesAndChannelsExtracted(categories []*discord.Category, channels []*discord.Channel, oldestMessageTimestamp int64) {
	send(EventDiscordCategoriesAndChannelsExtracted, DiscordCategoriesAndChannelsExtractedSignal{
		Categories:             categories,
		Channels:               channels,
		OldestMessageTimestamp: oldestMessageTimestamp,
	})
}

func SendExtractDiscordCategoriesAndChannelsFailed() {
	send(EventExtractDiscordCategoriesAndChannelsFailed, ExtractDiscordCategoriesAndChannelsFailedSignal{})
}
