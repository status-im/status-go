package signal

import (
	"github.com/status-im/status-go/protocol/discord"
)

const (

	// EventDiscordCategoriesAndChannelsExtracted triggered when categories and
	// channels for exported discord files have been successfully extracted
	EventDiscordCategoriesAndChannelsExtracted = "community.discordCategoriesAndChannelsExtracted"
)

type DiscordCategoriesAndChannelsExtractedSignal struct {
	Categories             []*discord.Category             `json:"discordCategories"`
	Channels               []*discord.Channel              `json:"discordChannels"`
	OldestMessageTimestamp int64                           `json:"oldestMessageTimestamp"`
	Errors                 map[string]*discord.ImportError `json:"errors"`
}

func SendDiscordCategoriesAndChannelsExtracted(categories []*discord.Category, channels []*discord.Channel, oldestMessageTimestamp int64, errors map[string]*discord.ImportError) {
	send(EventDiscordCategoriesAndChannelsExtracted, DiscordCategoriesAndChannelsExtractedSignal{
		Categories:             categories,
		Channels:               channels,
		OldestMessageTimestamp: oldestMessageTimestamp,
		Errors:                 errors,
	})
}
