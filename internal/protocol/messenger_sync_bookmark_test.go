package protocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/connection"
	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
	"github.com/status-im/status-go/pkg/services/browsers"
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
	theirMessenger := s.anotherMessenger()

	err = theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &messagingtypes.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := theirMessenger.SendPairInstallation(context.Background(), "", nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations()) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations()[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	_, err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	// sync
	err = s.m.SyncBookmark(context.Background(), &bookmark, s.m.dispatchMessage)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	err = testutils.RetryWithBackOff(func() error {
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
	err = testutils.RetryWithBackOff(func() error {
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
}

func (s *MessengerSyncBookmarkSuite) TestPairedInstallationsRetryIndependently() {
	s.messagingEnv.SetProcessMailserverBatchHook(func(context.Context, types.MailserverBatch, uint64, func(int) (bool, uint64), bool) error {
		return nil
	})
	origin := s.m
	online := s.anotherMessenger()
	offline := s.anotherMessenger()
	ctx := context.Background()
	for _, receiver := range []*Messenger{online, offline} {
		s.Require().NoError(receiver.SetInstallationMetadata(receiver.installationID, &messagingtypes.InstallationMetadata{
			Name: "paired", DeviceType: "desktop",
		}))
		_, err := receiver.SendPairInstallation(ctx, "", nil)
		s.Require().NoError(err)
		_, err = WaitOnMessengerResponse(origin, func(r *MessengerResponse) bool {
			_, ok := origin.allInstallations.Load(receiver.installationID)
			return ok
		}, "installation not received")
		s.Require().NoError(err)
		_, err = origin.EnableInstallation(receiver.installationID)
		s.Require().NoError(err)
	}

	bookmark := browsers.Bookmark{Name: "paired retry", URL: "https://example.org"}
	s.Require().NoError(origin.SyncBookmark(ctx, &bookmark, origin.dispatchMessage))
	_, err := WaitOnMessengerResponse(online, func(r *MessengerResponse) bool {
		return len(r.Bookmarks) == 1
	}, "online installation did not receive bookmark")
	s.Require().NoError(err)

	onlinePeer := reliability.InstallationPeerID(&origin.identity.PublicKey, online.installationID)
	offlinePeer := reliability.InstallationPeerID(&origin.identity.PublicKey, offline.installationID)
	s.Require().Eventually(func() bool {
		_, _ = origin.RetrieveAll()
		var onlinePending, offlineSent int
		err := origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", onlinePeer[:]).Scan(&onlinePending)
		if err != nil {
			return false
		}
		err = origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ? AND send_count > 0", offlinePeer[:]).Scan(&offlineSent)
		return err == nil && onlinePending == 0 && offlineSent == 1
	}, 10*time.Second, 100*time.Millisecond)

	origin.messaging.ConnectionChanged(connection.State{Offline: true})
	// Discard C's live envelopes to simulate a missed transmission, then
	// make its pending retry eligible without waiting for the real backoff.
	_, err = offline.messaging.RetrieveRawAll()
	s.Require().NoError(err)
	_, err = origin.database.Exec("UPDATE mvds_states SET send_epoch = 0 WHERE peer_id = ?", offlinePeer[:])
	s.Require().NoError(err)
	origin.messaging.ConnectionChanged(connection.State{})
	_, err = WaitOnMessengerResponse(offline, func(r *MessengerResponse) bool {
		return len(r.Bookmarks) == 1
	}, "offline installation did not receive bookmark on retry")
	s.Require().NoError(err)
	bookmarks, err := offline.browserDatabase.GetBookmarks()
	s.Require().NoError(err)
	s.Require().Len(bookmarks, 1)
	s.Require().Equal(bookmark.URL, bookmarks[0].URL)

	s.Require().Eventually(func() bool {
		_, _ = origin.RetrieveAll()
		var pending int
		err := origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", offlinePeer[:]).Scan(&pending)
		return err == nil && pending == 0
	}, 10*time.Second, 100*time.Millisecond)

	// Self-chat uses the same delivery route and identifies a visible message
	// for delivery confirmation.
	_, err = origin.CreateOneToOneChat(&requests.CreateOneToOneChat{ID: cryptotypes.FromHex(origin.myHexIdentity())})
	s.Require().NoError(err)
	message := buildTestMessage(*origin.Chat(origin.myHexIdentity()))
	message.Text = "self-chat"
	response, err := origin.SendChatMessage(ctx, message)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	id := response.Messages()[0].ID
	_, err = WaitOnMessengerResponse(online, func(r *MessengerResponse) bool {
		for _, item := range r.Messages() {
			if item.ID == id {
				return item.Seen && item.LocalChatID == origin.myHexIdentity()
			}
		}
		return false
	}, "online installation did not receive self-message")
	s.Require().NoError(err)
	received, err := online.persistence.MessageByID(id)
	s.Require().NoError(err)
	s.Require().Equal(common.OutgoingStatusDelivered, received.OutgoingStatus)
	s.Require().Eventually(func() bool {
		_, _ = origin.RetrieveAll()
		outgoing, err := origin.persistence.MessageByID(id)
		return err == nil && outgoing.OutgoingStatus == common.OutgoingStatusDelivered
	}, 10*time.Second, 100*time.Millisecond)

	var pending int
	err = origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", offlinePeer[:]).Scan(&pending)
	s.Require().NoError(err)
	s.Require().Equal(1, pending)
	s.Require().NoError(origin.DisableInstallation(offline.installationID))
	err = origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", offlinePeer[:]).Scan(&pending)
	s.Require().NoError(err)
	s.Require().Zero(pending)

	_, err = origin.EditMessage(ctx, &requests.EditMessage{ID: cryptotypes.FromHex(id), Text: "edited self-chat"})
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(online, func(response *MessengerResponse) bool {
		for _, item := range response.Messages() {
			if item.ID == id {
				return item.Text == "edited self-chat"
			}
		}
		return false
	}, "online installation did not receive self-chat edit")
	s.Require().NoError(err)

	_, err = origin.DeleteMessageAndSend(ctx, id)
	s.Require().NoError(err)
	s.Require().Eventually(func() bool {
		_, _ = online.RetrieveAll()
		item, err := online.persistence.MessageByID(id)
		return err == nil && item.Deleted
	}, 10*time.Second, 100*time.Millisecond)

	// Emulate an older peer's advertised version. Both peers use the same
	// profile key, but only the upgraded one should get a scoped entry.
	_, err = origin.EnableInstallation(offline.installationID)
	s.Require().NoError(err)
	_, err = origin.database.Exec("UPDATE installations SET version = 1 WHERE installation_id = ?", offline.installationID)
	s.Require().NoError(err)
	bookmark = browsers.Bookmark{Name: "mixed versions", URL: "https://example.net"}
	origin.messaging.ConnectionChanged(connection.State{Offline: true})
	s.Require().NoError(origin.SyncBookmark(ctx, &bookmark, origin.dispatchMessage))
	var legacyPeer [65]byte
	copy(legacyPeer[:], crypto.FromECDSAPub(&origin.identity.PublicKey))
	err = origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", legacyPeer[:]).Scan(&pending)
	s.Require().NoError(err)
	s.Require().Equal(1, pending)
	err = origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", offlinePeer[:]).Scan(&pending)
	s.Require().NoError(err)
	s.Require().Zero(pending)
	origin.messaging.ConnectionChanged(connection.State{})
	_, err = WaitOnMessengerResponse(offline, func(r *MessengerResponse) bool {
		for _, received := range r.Bookmarks {
			if received.URL == bookmark.URL {
				return true
			}
		}
		return false
	}, "legacy installation did not receive bookmark")
	s.Require().NoError(err)
	s.Require().Eventually(func() bool {
		_, _ = origin.RetrieveAll()
		var legacyPending, upgradedPending int
		if err := origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", legacyPeer[:]).Scan(&legacyPending); err != nil {
			return false
		}
		err := origin.database.QueryRow("SELECT count(*) FROM mvds_states WHERE peer_id = ?", onlinePeer[:]).Scan(&upgradedPending)
		return err == nil && legacyPending == 0 && upgradedPending > 0
	}, 10*time.Second, 100*time.Millisecond)

	// Exercise a profile-wide send in the reverse direction, too.
	_, err = offline.EnableInstallation(origin.installationID)
	s.Require().NoError(err)
	_, err = offline.database.Exec("UPDATE installations SET version = 1 WHERE installation_id = ?", origin.installationID)
	s.Require().NoError(err)
	reply := browsers.Bookmark{Name: "legacy sender", URL: "https://example.edu"}
	s.Require().NoError(offline.SyncBookmark(ctx, &reply, offline.dispatchMessage))
	_, err = WaitOnMessengerResponse(origin, func(r *MessengerResponse) bool {
		for _, received := range r.Bookmarks {
			if received.URL == reply.URL {
				return true
			}
		}
		return false
	}, "upgraded installation did not receive legacy bookmark")
	s.Require().NoError(err)
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
