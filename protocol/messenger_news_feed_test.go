package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/mmcdole/gofeed"
)

type MessengerNewsFeedSuite struct {
	MessengerBaseTestSuite
	m *Messenger // main instance of Messenger
}

func (s *MessengerNewsFeedSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()

	s.m = s.newMessenger()
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
