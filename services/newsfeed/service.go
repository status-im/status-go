package newsfeed

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/newsfeed"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/signal"
)

var (
	errServiceNotRunning = errors.New("service is not running")
	errNoActivityCenter  = errors.New("no activity center set up")
)

func isNil(data interface{}) bool {
	v := reflect.ValueOf(data).Kind()
	return data == nil || v == reflect.Ptr && reflect.ValueOf(data).IsNil()
}

type Service struct {
	logger          *zap.Logger
	storage         Persistence
	ac              ActivityCenter
	newsFeedManager *newsfeed.NewsFeedManager
}

func NewService(logger *zap.Logger, storage Persistence, ac ActivityCenter) *Service {
	service := &Service{
		logger:  logger,
		storage: storage,
	}
	service.SetActivityCenter(ac)
	return service
}

func (s *Service) Start() error {
	lastFetched, err := s.storage.GetLastFetchedTimestamp()
	if err != nil {
		return err
	}
	var feedUrl string
	if gocommon.IsMobilePlatform() {
		feedUrl = newsfeed.STATUS_MOBILE_FEED_URL
	} else {
		feedUrl = newsfeed.STATUS_DESKTOP_FEED_URL
	}
	s.newsFeedManager = newsfeed.NewNewsFeedManager(
		newsfeed.WithURL(feedUrl),
		newsfeed.WithParser(gofeed.NewParser()),
		newsfeed.WithHandler(s),
		newsfeed.WithLogger(s.logger),
		newsfeed.WithPollingInterval(30*time.Minute),
		newsfeed.WithFetchFrom(lastFetched),
	)

	newsFeedEnabled, err := s.IsNewsFeedEnabled()
	if err != nil {
		return err
	}
	if newsFeedEnabled {
		s.newsFeedManager.StartPolling(context.Background())
	}

	return nil
}

func (s *Service) Stop() error {
	if s.newsFeedManager != nil {
		s.newsFeedManager.StopPolling()
	}
	return nil
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "newsfeed",
			Version:   "0.1.0",
			Service: &API{
				service: s,
			},
		},
	}
}

func (s *Service) SetActivityCenter(ac ActivityCenter) {
	if isNil(ac) {
		s.ac = nil
	} else {
		s.ac = ac
	}
}

func (s *Service) HandleFeedItem(feedItem *gofeed.Item) (*protocol.MessengerResponse, error) {
	if s.ac == nil {
		return nil, errNoActivityCenter
	}

	response := &protocol.MessengerResponse{}

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
		s.logger.Error("HandleFeedItem: failed to generate a UUID", zap.Error(err))
		return nil, err
	}
	notification := &protocol.ActivityCenterNotification{
		ID:              types.FromHex(id.String()),
		Type:            protocol.ActivityCenterNotificationTypeNews,
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

	err = s.ac.AddNotification(response, notification)
	if err != nil {
		s.logger.Error("HandleFeedItem: failed to save notification", zap.Error(err))
		return nil, err
	}

	// Update the lastFetch time to the current time
	err = s.storage.SaveLastFetchedTimestamp(time.Now().UTC())
	if err != nil {
		s.logger.Error("HandleFeedItem: failed to save last fetch time", zap.Error(err))
		return nil, err
	}

	return response, nil
}

func (s *Service) HandleFeedItemAndSend(feedItem *gofeed.Item) error {
	response, err := s.HandleFeedItem(feedItem)
	if err != nil {
		s.logger.Error("HandleFeedItemAndSend: failed to handle feed item", zap.Error(err))
	}
	signal.SendNewMessages(response)
	return nil
}

func (s *Service) FetchNewsMessages() (*protocol.MessengerResponse, error) {
	if s.newsFeedManager == nil {
		return nil, errServiceNotRunning
	}

	items, err := s.newsFeedManager.FetchRSS()
	if err != nil {
		s.logger.Error("FetchNewsMessages: error fetching RSS feed", zap.Error(err))
		return nil, err
	}

	response := &protocol.MessengerResponse{}

	for _, item := range items {
		resp, err := s.HandleFeedItem(item)
		if err != nil {
			s.logger.Error("FetchNewsMessages: error handling feed item", zap.Error(err))
			return nil, err
		}
		err = response.Merge(resp)
		if err != nil {
			s.logger.Error("FetchNewsMessages: error merging responses", zap.Error(err))
			return nil, err
		}
	}
	return response, nil
}

// The News Feed is enabled if both:
// 1. The News Feed is enabled in the settings.
// 2. The RSS feed is enabled in the settings.
func (s *Service) IsNewsFeedEnabled() (bool, error) {
	newsFeedEnabled, err := s.storage.GetEnabled()
	if err != nil {
		return false, err
	}
	if !newsFeedEnabled {
		return false, nil
	}
	newsRSSEnabled, err := s.storage.GetRSSEnabled()
	if err != nil {
		return false, err
	}
	return newsRSSEnabled, nil
}

func (s *Service) changeNewsFeedManagerAfterUpdate() error {
	if s.newsFeedManager == nil {
		return nil
	}
	isNewsFeedEnabled, err := s.IsNewsFeedEnabled()
	if err != nil {
		return err
	}

	if isNewsFeedEnabled {
		// Reset the fetchFrom time to now since we don't want to fetch items that were posted while the feed was disabled
		now := time.Now().UTC()
		s.newsFeedManager.SetFetchFrom(now)
		err = s.storage.SaveLastFetchedTimestamp(now)
		if err != nil {
			s.logger.Error("changeNewsFeedManagerAfterUpdate: failed to save last fetch time", zap.Error(err))
			return err
		}
		s.newsFeedManager.StartPolling(context.Background())
	} else {
		s.newsFeedManager.StopPolling()
	}
	return nil
}

func (s *Service) ToggleEnabled(value bool) error {
	err := s.storage.SaveEnabled(value)
	if err != nil {
		return err
	}
	return s.changeNewsFeedManagerAfterUpdate()
}

func (s *Service) ToggleRSSEnabled(value bool) error {
	err := s.storage.SaveRSSEnabled(value)
	if err != nil {
		return err
	}
	return s.changeNewsFeedManagerAfterUpdate()
}
