package newsfeed

import (
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/mmcdole/gofeed"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/services/newsfeed/migrations"
	mock_newsfeed "github.com/status-im/status-go/services/newsfeed/mock"
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
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     migrations.AssetNames(),
			AssetFunc: migrations.Asset,
		},
	}))
	s.Require().NoError(err)
	s.storage = NewSQLitePersistence(db)

	s.service = NewService(testutils.MustCreateTestLogger(), s.storage, s.ac)
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
	lastFetched, err := s.service.storage.GetLastFetchedTimestamp()
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(time.Now().UTC().Second(), lastFetched.UTC().Second())
}

func (s *MessengerNewsFeedSuite) TestHandleNewsFeedItemWithNamespacedCustomFields() {
	const feedXML = `<?xml version="1.0" encoding="UTF-8"?><rss xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:atom="http://www.w3.org/2005/Atom" version="2.0" xmlns:status="https://status.app/ns/rss/1.0"><channel><title>Desktop news - Our Status</title><description>Research &amp; Development at Status</description><link>https://our.status.im/</link><atom:link href="http://127.0.0.1:8099/tag/desktop-news/rss/" rel="self" type="application/rss+xml"></atom:link><item><title>Status v2.38.2: fixes &amp; tweaks &lt;beta&gt;</title><description>Fixes &amp;amp; improvements:&lt;br /&gt;shipped today.&lt;ul&gt;&lt;li&gt;High CPU usage&lt;/li&gt;&lt;li&gt;Endless loading, see the &lt;a href="https://wikipedia.org/"&gt;test link&lt;/a&gt; for details&lt;ol&gt;&lt;li&gt;in communities&lt;/li&gt;&lt;li&gt;on iOS&lt;/li&gt;&lt;/ol&gt;&lt;/li&gt;&lt;li&gt;&lt;a href="https://wikipedia.org/wiki/RSS"&gt;RSS on Wikipedia&lt;/a&gt;&lt;/li&gt;&lt;/ul&gt;Thanks for testing.</description><link>https://our.status.im/status-v2-38-2-is-here/</link><guid isPermaLink="false">6a5f5210834292000196a904</guid><category>Desktop news</category><dc:creator>Bobby Zalke</dc:creator><pubDate>Tue, 21 Jul 2026 11:19:19 GMT</pubDate><content:encoded>Fixes &amp;amp; improvements:&lt;br /&gt;shipped today.&lt;ul&gt;&lt;li&gt;High CPU usage&lt;/li&gt;&lt;li&gt;Endless loading, see the &lt;a href="https://wikipedia.org/"&gt;test link&lt;/a&gt; for details&lt;ol&gt;&lt;li&gt;in communities&lt;/li&gt;&lt;li&gt;on iOS&lt;/li&gt;&lt;/ol&gt;&lt;/li&gt;&lt;li&gt;&lt;a href="https://wikipedia.org/wiki/RSS"&gt;RSS on Wikipedia&lt;/a&gt;&lt;/li&gt;&lt;/ul&gt;Thanks for testing.</content:encoded><status:newsLink>https://status.app/</status:newsLink><status:newsLinkLabel>Update your Status</status:newsLinkLabel></item></channel></rss>`

	feed, err := gofeed.NewParser().Parse(strings.NewReader(feedXML))
	s.Require().NoError(err)
	s.Require().Len(feed.Items, 1)

	item := feed.Items[0]
	s.Require().Contains(item.Extensions, "status")
	s.Require().Contains(item.Extensions["status"], "newsLink")
	s.Require().Contains(item.Extensions["status"], "newsLinkLabel")
	s.Require().Equal("https://status.app/", item.Extensions["status"]["newsLink"][0].Value)
	s.Require().Equal("Update your Status", item.Extensions["status"]["newsLinkLabel"][0].Value)

	s.ac.EXPECT().AddNotification(gomock.Any(), gomock.Cond(func(a any) bool {
		notification, ok := a.(*protocol.ActivityCenterNotification)
		s.Require().True(ok)

		return notification.Type == protocol.ActivityCenterNotificationTypeNews &&
			notification.NewsTitle == item.Title &&
			notification.NewsDescription == item.Description &&
			notification.NewsContent == item.Content &&
			notification.NewsLink == "https://status.app/" &&
			notification.NewsLinkLabel == "Update your Status"
	}))

	err = s.service.HandleFeedItemAndSend(item)
	s.Require().NoError(err)
	lastFetched, err := s.service.storage.GetLastFetchedTimestamp()
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(time.Now().UTC().Second(), lastFetched.UTC().Second())
}

func (s *MessengerNewsFeedSuite) TestGetCustomFieldFromCustomMap() {
	item := &gofeed.Item{
		Custom: map[string]string{
			"newsLink":            "https://status.app/direct",
			"status:newsLinkLabel": "Update from namespaced custom",
		},
	}

	s.Require().Equal("https://status.app/direct", getCustomField(item, "newsLink"))
	s.Require().Equal("Update from namespaced custom", getCustomField(item, "newsLinkLabel"))
}

func (s *MessengerNewsFeedSuite) TestToggleSettings() {
	err := s.service.Start()
	s.Require().NoError(err)

	err = s.service.ToggleEnabled(true)
	s.Require().NoError(err)
	s.Require().True(s.service.newsFeedManager.IsPolling())

	oldFetchFrom := s.service.newsFeedManager.GetFetchFrom()

	// Polling is off as soon as one setting is off
	err = s.service.ToggleEnabled(false)
	s.Require().NoError(err)
	s.Require().False(s.service.newsFeedManager.IsPolling())

	err = s.service.ToggleRSSEnabled(false)
	s.Require().NoError(err)
	s.Require().False(s.service.newsFeedManager.IsPolling())

	//Poolling is still off
	err = s.service.ToggleRSSEnabled(true)
	s.Require().NoError(err)
	s.Require().False(s.service.newsFeedManager.IsPolling())

	// Polling restart if both settings are on
	err = s.service.ToggleEnabled(true)
	s.Require().NoError(err)
	s.Require().True(s.service.newsFeedManager.IsPolling())

	// Check that the fetchFrom timestamp is updated
	newFetchFrom := s.service.newsFeedManager.GetFetchFrom()
	s.Require().Greater(newFetchFrom, oldFetchFrom)

	s.service.newsFeedManager.StopPolling()
	s.Require().False(s.service.newsFeedManager.IsPolling())
}

func (s *MessengerNewsFeedSuite) TestPauseResumeBackground() {
	err := s.service.Start()
	s.Require().NoError(err)
	s.Require().True(s.service.started)

	err = s.service.Pause()
	s.Require().NoError(err)
	s.Require().False(s.service.started)

	err = s.service.Resume()
	s.Require().NoError(err)
	s.Require().True(s.service.started)
}
