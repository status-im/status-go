package newsfeed

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/mmcdole/gofeed"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/newsfeed/migrations"
	mock_newsfeed "github.com/status-im/status-go/services/newsfeed/mock"
	"github.com/status-im/status-go/t/helpers"
)

type MessengerNewsFeedSuite struct {
	suite.Suite
	service *Service
	storage Persistence
	ac      *mock_newsfeed.MockActivityCenter
}

func (s *MessengerNewsFeedSuite) SetupTest() {
	// Mock activity center
	ctrl := gomock.NewController(s.T())
	s.ac = mock_newsfeed.NewMockActivityCenter(ctrl)

	// Setup storage
	db, err := helpers.SetupTestMemorySQLDB(helpers.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     migrations.AssetNames(),
			AssetFunc: migrations.Asset,
		},
	}))
	s.Require().NoError(err)
	s.storage = NewSQLitePersistence(db)

	s.service = NewService(tt.MustCreateTestLogger(), s.storage, s.ac)
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

	s.ac.EXPECT().AddNotification(gomock.Any(), gomock.Cond(func(a any) bool {
		notification, ok := a.(*protocol.ActivityCenterNotification)
		s.Require().True(ok)

		return notification.Type == protocol.ActivityCenterNotificationTypeNews &&
			notification.Read == false &&
			notification.Deleted == false &&
			notification.NewsTitle == item.Title &&
			notification.NewsDescription == item.Description &&
			notification.NewsContent == item.Content &&
			notification.NewsImageURL == item.Image.URL &&
			notification.NewsLink == "" &&
			notification.NewsLinkLabel == ""
	}))

	err := s.service.HandleFeedItemAndSend(&item)
	s.Require().NoError(err)

	// Check that the lastFetched timestamp is updated
	lastFetched, err := s.service.storage.NewsFeedLastFetchedTimestamp()
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(time.Now().UTC().Second(), lastFetched.UTC().Second())
}

func (s *MessengerNewsFeedSuite) TestToggleSettings() {
	err := s.service.Start()
	s.Require().NoError(err)

	err = s.service.ToggleNewsFeedEnabled(true)
	s.Require().NoError(err)
	s.Require().True(s.service.newsFeedManager.IsPolling())

	oldFetchFrom := s.service.newsFeedManager.GetFetchFrom()

	// Polling is off as soon as one setting is off
	err = s.service.ToggleNewsFeedEnabled(false)
	s.Require().NoError(err)
	s.Require().False(s.service.newsFeedManager.IsPolling())

	err = s.service.ToggleNewsRSSEnabled(false)
	s.Require().NoError(err)
	s.Require().False(s.service.newsFeedManager.IsPolling())

	//Poolling is still off
	err = s.service.ToggleNewsRSSEnabled(true)
	s.Require().NoError(err)
	s.Require().False(s.service.newsFeedManager.IsPolling())

	// Polling restart if both settings are on
	err = s.service.ToggleNewsFeedEnabled(true)
	s.Require().NoError(err)
	s.Require().True(s.service.newsFeedManager.IsPolling())

	// Check that the fetchFrom timestamp is updated
	newFetchFrom := s.service.newsFeedManager.GetFetchFrom()
	s.Require().Greater(newFetchFrom, oldFetchFrom)

	s.service.newsFeedManager.StopPolling()
	s.Require().False(s.service.newsFeedManager.IsPolling())
}
