package protocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/browsers"

	"github.com/stretchr/testify/suite"
)

func TestMessengerSyncBookmarkSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncBookmarkSuite))
}

type MessengerSyncBookmarkSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerSyncBookmarkSuite) TestSyncBookmark() {
	//add bookmark
	bookmark := browsers.Bookmark{
		Name:    "status official site",
		URL:     "https://status.im",
		Removed: false,
	}
	_, err := s.m.browserDatabase.StoreBookmark(bookmark)
	s.Require().NoError(err)

	// pair
	theirMessenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)

	err = theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := theirMessenger.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	// sync
	err = s.m.SyncBookmark(context.Background(), &bookmark, s.m.dispatchMessage)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}
		if response.Bookmarks != nil {
			return nil
		}
		return errors.New("Not received all bookmarks")
	})

	s.Require().NoError(err)

	bookmarks, err := theirMessenger.browserDatabase.GetBookmarks()
	s.Require().NoError(err)
	s.Require().Equal(1, len(bookmarks))
	s.Require().False(bookmarks[0].Removed)

	// sync removed state
	bookmark.Removed = true
	err = s.m.SyncBookmark(context.Background(), &bookmark, s.m.dispatchMessage)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}
		if response.Bookmarks != nil {
			return nil
		}
		return errors.New("Not received all bookmarks")
	})
	bookmarks, err = theirMessenger.browserDatabase.GetBookmarks()
	s.Require().NoError(err)
	s.Require().Equal(1, len(bookmarks))
	s.Require().True(bookmarks[0].Removed)

	s.Require().NoError(theirMessenger.Shutdown())

}

func (s *MessengerSyncBookmarkSuite) TestGarbageCollectRemovedBookmarks() {

	now := time.Now()

	// Create bookmarks that are flagged as deleted for more than 30 days
	bookmark1 := browsers.Bookmark{
		Name:      "status official site",
		URL:       "https://status.im",
		Removed:   true,
		DeletedAt: uint64(now.AddDate(0, 0, -31).Unix()),
	}

	bookmark2 := browsers.Bookmark{
		Name:      "Uniswap",
		URL:       "https://uniswap.org",
		Removed:   true,
		DeletedAt: uint64(now.AddDate(0, 0, -31).Unix()),
	}

	// This one is flagged for deletion less than 30 days
	bookmark3 := browsers.Bookmark{
		Name:      "Maker DAO",
		URL:       "https://makerdao.com",
		Removed:   true,
		DeletedAt: uint64(now.AddDate(0, 0, -29).Unix()),
	}

	// Store bookmarks
	_, err := s.m.browserDatabase.StoreBookmark(bookmark1)
	s.Require().NoError(err)

	_, err = s.m.browserDatabase.StoreBookmark(bookmark2)
	s.Require().NoError(err)

	_, err = s.m.browserDatabase.StoreBookmark(bookmark3)
	s.Require().NoError(err)

	bookmarks, err := s.m.browserDatabase.GetBookmarks()
	s.Require().NoError(err)
	s.Require().Len(bookmarks, 3)

	// err = s.m.GarbageCollectRemovedBookmarks(&now)
	err = s.m.GarbageCollectRemovedBookmarks()
	s.Require().NoError(err)

	bookmarks, err = s.m.browserDatabase.GetBookmarks()
	s.Require().NoError(err)
	s.Require().Len(bookmarks, 1)
}
