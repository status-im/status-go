package protocol

import (
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/signal"
)

func (m *Messenger) HandleFeedItem(feedItem *gofeed.Item) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	imageURL := ""
	if feedItem.Image != nil {
		imageURL = feedItem.Image.URL
	}
	newsLink := ""
	newsLinkLabel := ""
	if feedItem.Custom != nil {
		newsLink = feedItem.Custom["newsLink"]
		newsLinkLabel = feedItem.Custom["newsLinkLabel"]
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
		NewsLink:        newsLink,
		NewsLinkLabel:   newsLinkLabel,
	}

	err = m.addActivityCenterNotification(response, notification, nil)
	if err != nil {
		m.logger.Error("HandleFeedItem: failed to save notification", zap.Error(err))
		return nil, err
	}

	// Update the lastFetch time to the current time
	err = m.settings.SaveSetting(settings.NewsFeedLastFetchedTimestamp.GetReactName(), time.Now())
	if err != nil {
		m.logger.Error("HandleFeedItem: failed to save last fetch time", zap.Error(err))
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

// The News Feed is enabled if both:
// 1. The News Feed is enabled in the settings.
// 2. The RSS feed is enabled in the settings.
func (m *Messenger) IsNewsFeedEnabled() (bool, error) {
	newsFeedEnabled, err := m.settings.NewsFeedEnabled()
	if err != nil {
		return false, err
	}
	if !newsFeedEnabled {
		return false, nil
	}
	newsRSSEnabled, err := m.settings.NewsRSSEnabled()
	if err != nil {
		return false, err
	}
	return newsRSSEnabled, nil
}

func (m *Messenger) changeNewsFeedManagerAfterUpdate() error {
	if m.newsFeedManager == nil {
		return nil
	}
	isNewsFeedEnabled, err := m.IsNewsFeedEnabled()
	if err != nil {
		return err
	}

	if isNewsFeedEnabled {
		m.newsFeedManager.StartPolling(m.ctx)
	} else {
		m.newsFeedManager.StopPolling()
	}
	return nil
}

func (m *Messenger) ToggleNewsFeedEnabled(value bool) error {
	err := m.settings.SaveSetting(settings.NewsFeedEnabled.GetReactName(), value)
	if err != nil {
		return err
	}
	return m.changeNewsFeedManagerAfterUpdate()
}

func (m *Messenger) ToggleNewsRSSEnabled(value bool) error {
	err := m.settings.SaveSetting(settings.NewsRSSEnabled.GetReactName(), value)
	if err != nil {
		return err
	}
	return m.changeNewsFeedManagerAfterUpdate()
}
