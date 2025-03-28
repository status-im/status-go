package protocol

import (
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/signal"
)

func (m *Messenger) HandleFeedItem(feedItem *gofeed.Item) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	imageURL := ""
	if feedItem.Image != nil {
		imageURL = feedItem.Image.URL
	}
	newsLinkLabel := ""
	if feedItem.Custom != nil {
		newsLinkLabel = feedItem.Custom["linkLabel"]
	}

	id, err := uuid.NewRandom()
	if err != nil {
		m.logger.Error("HandleFeedItem: failed to generate a UUID", zap.Error(err))
		return nil, err
	}
	notification := &ActivityCenterNotification{
		ID:              types.FromHex(id.String()),
		Type:            ActivityCenterNotificationTypeNews,
		Timestamp:       uint64(feedItem.PublishedParsed.UnixMilli()),
		Read:            false,
		Deleted:         false,
		NewsTitle:       feedItem.Title,
		NewsDescription: feedItem.Description,
		NewsContent:     feedItem.Content,
		NewsImageURL:    imageURL,
		NewsLink:        feedItem.Link,
		NewsLinkLabel:   newsLinkLabel,
	}

	err = m.addActivityCenterNotification(response, notification, nil)
	if err != nil {
		m.logger.Error("HandleFeedItem: failed to save notification", zap.Error(err))
		return nil, err
	}

	return response, nil
}

func (m *Messenger) HandleFeedItemAndSend(feedItem *gofeed.Item) error {
	response, err := m.HandleFeedItem(feedItem)
	if err != nil {
		m.logger.Error("HandleFeedItemAndSend: failed to handle feed item", zap.Error(err))
	}
	signal.SendNewMessages(response)
	return nil
}

func (m *Messenger) FetchNewsMessages() (*MessengerResponse, error) {
	items, err := m.newsFeedManager.FetchRSS()
	if err != nil {
		m.logger.Error("FetchNewsMessages: error fetching RSS feed", zap.Error(err))
		return nil, err
	}

	response := &MessengerResponse{}

	for _, item := range items {
		resp, err := m.HandleFeedItem(item)
		if err != nil {
			m.logger.Error("FetchNewsMessages: error handling feed item", zap.Error(err))
			return nil, err
		}
		err = response.Merge(resp)
		if err != nil {
			m.logger.Error("FetchNewsMessages: error merging responses", zap.Error(err))
			return nil, err
		}
	}
	return response, nil
}
