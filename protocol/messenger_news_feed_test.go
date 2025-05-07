package protocol

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
)

type MessengerNewsFeedSuite struct {
	MessengerBaseTestSuite
	m *Messenger // main instance of Messenger
}

func (s *MessengerNewsFeedSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, []Option{WithNewsFeed()})
	s.Require().NoError(err)

	s.m = messenger
}

func TestMessengerNewsFeedSuite(t *testing.T) {
	suite.Run(t, new(MessengerNewsFeedSuite))
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func (s *MessengerNewsFeedSuite) TestHandleNewsFeedItem() {
	item := gofeed.Item{
		GUID:            gofakeit.UUID(),
		Title:           gofakeit.LetterN(5),
		PublishedParsed: ptrTime(time.Now().Add(-1 * time.Hour)),
		Description:     gofakeit.LetterN(5),
		Link:            gofakeit.URL(),
		Content:         gofakeit.LetterN(5),
		Image:           &gofeed.Image{URL: gofakeit.URL()},
	}

	err := s.m.HandleFeedItemAndSend(&item)
	s.Require().NoError(err)

	// Check that the lastFetched timestamp is updated
	lastFetched, err := s.m.settings.NewsFeedLastFetchedTimestamp()
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(time.Now().UTC().Second(), lastFetched.UTC().Second())

	// Check that the notification is saved in the database
	_, notifications, err := s.m.persistence.ActivityCenterNotifications("", 10, []ActivityCenterType{ActivityCenterNotificationTypeNews}, ActivityCenterQueryParamsReadAll, true)
	s.Require().NoError(err)
	s.Require().NotNil(notifications)
	s.Require().Len(notifications, 1)
	s.Require().Equal(item.Title, notifications[0].NewsTitle)
}

func (s *MessengerNewsFeedSuite) TestToggleSettings() {
	err := s.m.ToggleNewsFeedEnabled(true)
	s.Require().NoError(err)
	s.Require().True(s.m.newsFeedManager.IsPolling())

	// Polling is off as soon as one setting is off
	err = s.m.ToggleNewsFeedEnabled(false)
	s.Require().NoError(err)
	s.Require().False(s.m.newsFeedManager.IsPolling())

	err = s.m.ToggleNewsRSSEnabled(false)
	s.Require().NoError(err)
	s.Require().False(s.m.newsFeedManager.IsPolling())

	//Poolling is still off
	err = s.m.ToggleNewsRSSEnabled(true)
	s.Require().NoError(err)
	s.Require().False(s.m.newsFeedManager.IsPolling())

	// Polling restart if both settings are on
	err = s.m.ToggleNewsFeedEnabled(true)
	s.Require().NoError(err)
	s.Require().True(s.m.newsFeedManager.IsPolling())

	s.m.newsFeedManager.StopPolling()
	s.Require().False(s.m.newsFeedManager.IsPolling())
}
