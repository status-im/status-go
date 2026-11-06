package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/signal"
)

func sectionCountsTestDB(t testing.TB) *sqlitePersistence {
	t.Helper()
	db, err := openTestDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	p := newSQLitePersistence(db)
	p.threadClock = func() uint64 { return uint64(30 * 24 * time.Hour / time.Millisecond) }
	_, err = db.Exec(`DELETE FROM communities_communities`)
	require.NoError(t, err)
	return p
}

func insertSectionCountChat(t testing.TB, p *sqlitePersistence, id, communityID string, chatType ChatType, unread, mentions int) {
	t.Helper()
	_, err := p.db.Exec(`INSERT INTO chats
		(id, name, type, timestamp, community_id, unviewed_message_count, unviewed_mentions_count)
		VALUES (?, ?, ?, 0, ?, ?, ?)`, id, id, chatType, communityID, unread, mentions)
	require.NoError(t, err)
}

func insertSectionCountThread(t testing.TB, p *sqlitePersistence, chatID, threadID string, unread, mentions int) {
	t.Helper()
	_, err := p.db.Exec(`INSERT INTO threads
		(thread_id, chat_id, parent_message_id, name, followed, last_activity_at)
		VALUES (?, ?, ?, '', 1, ?)`, threadID, chatID, threadID, p.threadNow())
	require.NoError(t, err)
	for i := 0; i < unread; i++ {
		_, err = p.db.Exec(`INSERT INTO user_messages
			(id, local_chat_id, chat_id, thread_id, whisper_timestamp, source, text,
			 content_type, timestamp, clock_value, mentioned, outgoing_status)
			VALUES (?, ?, ?, ?, 0, '', '', 1, 0, ?, ?, '')`,
			fmt.Sprintf("%s-%d", threadID, i), chatID, chatID, threadID, i+1, i < mentions)
		require.NoError(t, err)
	}
}

func TestSectionUnreadCountsSections(t *testing.T) {
	p := sectionCountsTestDB(t)
	_, err := p.db.Exec(`INSERT INTO communities_communities(id, description, joined, spectated)
		VALUES (X'01', X'', 1, 0), (X'02', X'', 0, 1), (X'03', X'', 0, 0), (X'04', X'', 1, 1)`)
	require.NoError(t, err)
	insertSectionCountChat(t, p, "dm", "", ChatTypeOneToOne, 5, 2)
	insertSectionCountChat(t, p, "group", "", ChatTypePrivateGroupChat, 3, 0)
	insertSectionCountChat(t, p, "public", "", ChatTypePublic, 2, 0)
	insertSectionCountChat(t, p, "community", "0x01", ChatTypeCommunityChat, 4, 1)
	insertSectionCountChat(t, p, "spectated", "0x02", ChatTypeCommunityChat, 6, 0)
	insertSectionCountChat(t, p, "left-community", "0x03", ChatTypeCommunityChat, 100, 100)
	insertSectionCountChat(t, p, "inactive", "", ChatTypeOneToOne, 100, 100)
	insertSectionCountChat(t, p, "unknown", "", 0, 100, 100)
	_, err = p.db.Exec(`UPDATE chats SET active = 0 WHERE id = 'inactive'`)
	require.NoError(t, err)
	insertSectionCountThread(t, p, "dm", "dm-thread", 2, 0)
	insertSectionCountThread(t, p, "community", "community-thread", 3, 1)
	insertSectionCountThread(t, p, "community", "second-thread", 2, 0)
	insertSectionCountThread(t, p, "spectated", "spectated-thread", 1, 1)
	insertSectionCountThread(t, p, "inactive", "inactive-thread", 10, 10)
	insertSectionCountThread(t, p, "missing", "orphan-thread", 10, 10)

	for _, includeThreads := range []bool{false, true} {
		t.Run(fmt.Sprint(includeThreads), func(t *testing.T) {
			personalUnread, personalMentions := uint64(10), uint64(2)
			communityUnread, communityMentions := uint64(4), uint64(1)
			spectatedUnread, spectatedMentions := uint64(6), uint64(0)
			if includeThreads {
				personalUnread, personalMentions = 12, 4
				communityUnread, communityMentions = 9, 2
				spectatedUnread, spectatedMentions = 7, 1
			}
			counts, err := p.SectionUnreadCounts(context.Background(), includeThreads)
			require.NoError(t, err)
			require.Equal(t, []SectionUnreadCount{
				{PersonalSectionID, personalUnread, personalMentions},
				{"0x01", communityUnread, communityMentions},
				{"0x02", spectatedUnread, spectatedMentions},
				{"0x04", 0, 0},
			}, counts)
		})
	}
}

func TestSectionUnreadCountsMembershipAndChatLifecycle(t *testing.T) {
	p := sectionCountsTestDB(t)
	_, err := p.db.Exec(`INSERT INTO communities_communities(id, description, joined) VALUES (X'01', X'', 1)`)
	require.NoError(t, err)
	insertSectionCountChat(t, p, "channel", "0x01", ChatTypeCommunityChat, 3, 1)
	insertSectionCountThread(t, p, "channel", "thread", 2, 1)
	assertCounts := func(expected []SectionUnreadCount) {
		counts, err := p.SectionUnreadCounts(context.Background(), true)
		require.NoError(t, err)
		require.Equal(t, expected, counts)
	}
	joined := []SectionUnreadCount{{PersonalSectionID, 0, 0}, {"0x01", 5, 2}}
	assertCounts(joined)
	_, err = p.db.Exec(`UPDATE communities_communities SET joined = 0, spectated = 1`)
	require.NoError(t, err)
	assertCounts(joined)
	_, err = p.db.Exec(`UPDATE communities_communities SET spectated = 0`)
	require.NoError(t, err)
	assertCounts([]SectionUnreadCount{{PersonalSectionID, 0, 0}})
	_, err = p.db.Exec(`UPDATE communities_communities SET joined = 1; UPDATE chats SET active = 0`)
	require.NoError(t, err)
	assertCounts([]SectionUnreadCount{{PersonalSectionID, 0, 0}, {"0x01", 0, 0}})
}

func TestSectionUnreadCountsMuteRules(t *testing.T) {
	for _, mute := range []string{"chat", "community"} {
		t.Run(mute, func(t *testing.T) {
			p := sectionCountsTestDB(t)
			_, err := p.db.Exec(`INSERT INTO communities_communities(id, description, joined) VALUES (X'01', X'', 1)`)
			require.NoError(t, err)
			insertSectionCountChat(t, p, "plain", "0x01", ChatTypeCommunityChat, 7, 0)
			insertSectionCountChat(t, p, "mentioned", "0x01", ChatTypeCommunityChat, 5, 1)
			insertSectionCountThread(t, p, "plain", "plain-thread", 8, 0)
			insertSectionCountThread(t, p, "plain", "mentioned-thread", 4, 1)
			// A parent's mention must not expose unrelated muted thread unread.
			insertSectionCountThread(t, p, "mentioned", "another-plain-thread", 9, 0)
			if mute == "chat" {
				_, err = p.db.Exec(`UPDATE chats SET muted = 1`)
			} else {
				_, err = p.db.Exec(`UPDATE communities_communities SET muted = 1`)
			}
			require.NoError(t, err)
			counts, err := p.SectionUnreadCounts(context.Background(), true)
			require.NoError(t, err)
			require.Equal(t, SectionUnreadCount{"0x01", 9, 2}, counts[1])
		})
	}
}

func TestSectionUnreadCountsPersonalMuteRules(t *testing.T) {
	p := sectionCountsTestDB(t)
	insertSectionCountChat(t, p, "dm", "", ChatTypeOneToOne, 3, 3)
	insertSectionCountChat(t, p, "group", "", ChatTypePrivateGroupChat, 7, 0)
	insertSectionCountThread(t, p, "dm", "dm-thread", 2, 0)
	insertSectionCountThread(t, p, "group", "plain-thread", 4, 0)
	insertSectionCountThread(t, p, "group", "mentioned-thread", 5, 1)
	_, err := p.db.Exec(`UPDATE chats SET muted = 1`)
	require.NoError(t, err)
	counts, err := p.SectionUnreadCounts(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, []SectionUnreadCount{{PersonalSectionID, 10, 6}}, counts)
}

func TestSectionUnreadCountsThreadEligibilityAndVisibility(t *testing.T) {
	p := sectionCountsTestDB(t)
	insertSectionCountChat(t, p, "group", "", ChatTypePrivateGroupChat, 0, 0)
	for _, id := range []string{"eligible", "unfollowed", "archived", "deleted", "boundary", "filtered"} {
		insertSectionCountThread(t, p, "group", id, 3, 1)
	}
	_, err := p.db.Exec(`UPDATE threads SET followed = 0 WHERE thread_id = 'unfollowed';
		UPDATE threads SET deleted = 1 WHERE thread_id = 'deleted';
		UPDATE threads SET last_activity_at = ? WHERE thread_id = 'archived';
		UPDATE threads SET last_activity_at = ? WHERE thread_id = 'boundary';
		UPDATE user_messages SET seen = 1 WHERE id = 'filtered-0';
		UPDATE user_messages SET hide = 1 WHERE id = 'filtered-1';
		UPDATE user_messages SET deleted_for_me = 1 WHERE id = 'filtered-2';
		UPDATE user_messages SET deleted = 1 WHERE id = 'eligible-2';
		UPDATE user_messages SET replied = 1 WHERE id = 'eligible-1'`,
		threadArchiveCutoff(p.threadNow())-1, threadArchiveCutoff(p.threadNow()))
	require.NoError(t, err)
	counts, err := p.SectionUnreadCounts(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, []SectionUnreadCount{{PersonalSectionID, 5, 3}}, counts)

	// Archival is derived from time, not persisted as a separate flag.
	p.threadClock = func() uint64 { return uint64(40 * 24 * time.Hour / time.Millisecond) }
	counts, err = p.SectionUnreadCounts(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, []SectionUnreadCount{{PersonalSectionID, 0, 0}}, counts)
}

func TestSectionUnreadCountsMessengerFeatureGateAndErrors(t *testing.T) {
	p := sectionCountsTestDB(t)
	insertSectionCountChat(t, p, "group", "", ChatTypePrivateGroupChat, 2, 1)
	insertSectionCountThread(t, p, "group", "thread", 3, 1)
	m := &Messenger{persistence: p}
	for _, enabled := range []bool{false, true} {
		m.featureFlags.Threads = enabled
		for _, includeThreads := range []bool{false, true} {
			result, err := m.SectionUnreadCounts(context.Background(), includeThreads)
			require.NoError(t, err)
			expected := SectionUnreadCount{PersonalSectionID, 2, 1}
			if enabled && includeThreads {
				expected = SectionUnreadCount{PersonalSectionID, 5, 2}
			}
			require.Equal(t, []SectionUnreadCount{expected}, result.Sections)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := m.SectionUnreadCounts(ctx, true)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
}

func TestSectionUnreadCountsQueryPlan(t *testing.T) {
	p := sectionCountsTestDB(t)
	for _, includeThreads := range []bool{false, true} {
		query, args := p.sectionUnreadCountsQuery(includeThreads)
		rows, err := p.db.Query("EXPLAIN QUERY PLAN "+query, args...)
		require.NoError(t, err)
		var details []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
			details = append(details, detail)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		plan := strings.Join(details, "\n")
		t.Logf("includeThreads=%t\n%s", includeThreads, plan)
		if includeThreads {
			require.True(t, strings.Contains(plan, "um USING INDEX user_messages_unread_threads") ||
				strings.Contains(plan, "um USING INDEX idx_user_messages_unseen"), plan)
		} else {
			require.NotContains(t, plan, "user_messages")
			require.NotContains(t, plan, "threads")
		}
	}
}

func TestSectionUnreadCountsLiveReadAndFollowUpdates(t *testing.T) {
	p := sectionCountsTestDB(t)
	insertSectionCountChat(t, p, "group", "", ChatTypePrivateGroupChat, 0, 0)
	insertSectionCountThread(t, p, "group", "thread", 3, 1)
	m := &Messenger{persistence: p}
	m.featureFlags.Threads = true
	var invalidations int
	signal.SetHandler(func(data []byte) {
		var event signal.Envelope
		require.NoError(t, json.Unmarshal(data, &event))
		if event.Type == signal.EventSectionUnreadCountsChanged {
			invalidations++
		}
	})
	defer signal.ResetHandler()

	initial, err := m.SectionUnreadCounts(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, []SectionUnreadCount{{PersonalSectionID, 3, 1}}, initial.Sections)
	require.NoError(t, m.markThreadRead("group", "thread", 3, false))
	require.Equal(t, 1, invalidations)
	read, err := m.SectionUnreadCounts(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, []SectionUnreadCount{{PersonalSectionID, 0, 0}}, read.Sections)

	_, err = p.db.Exec(`UPDATE user_messages SET seen = 0; UPDATE threads SET followed = 0`)
	require.NoError(t, err)
	unfollowed, err := m.SectionUnreadCounts(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, read, unfollowed)
	_, err = p.db.Exec(`UPDATE threads SET followed = 1`)
	require.NoError(t, err)
	followed, err := m.SectionUnreadCounts(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, initial, followed)

	_, err = p.db.Exec(`DROP TABLE threads`)
	require.NoError(t, err)
	require.Error(t, m.markThreadRead("group", "thread", 3, false))
	require.Equal(t, 1, invalidations, "rolled-back writes must not invalidate")
	result, err := m.SectionUnreadCounts(context.Background(), true)
	require.Error(t, err)
	require.Nil(t, result)
}

func (s *MessengerThreadsSuite) TestSectionUnreadCountsLifecycle() {
	const chatID = "section-counts-group"
	p := s.m.persistence
	insertSectionCountChat(s.T(), p, chatID, "", ChatTypePrivateGroupChat, 0, 0)
	s.m.allChats.Store(chatID, &Chat{ID: chatID, ChatType: ChatTypePrivateGroupChat, Active: true})
	insertSectionCountThread(s.T(), p, chatID, "plain", 3, 0)
	insertSectionCountThread(s.T(), p, chatID, "mentioned", 2, 1)

	assertCounts := func(unread, mentions uint64) {
		result, err := s.m.SectionUnreadCounts(context.Background(), true)
		s.Require().NoError(err)
		s.Require().Equal(SectionUnreadCount{PersonalSectionID, unread, mentions}, result.Sections[0])
	}
	assertCounts(5, 1)
	response, err := s.m.UnfollowThread(context.Background(), &requests.UnfollowThread{ChatID: chatID, ThreadID: "mentioned"})
	s.Require().NoError(err)
	s.Require().True(response.HasSectionUnreadCountsChanges())
	assertCounts(3, 0)
	response, err = s.m.FollowThread(context.Background(), &requests.FollowThread{ChatID: chatID, ThreadID: "mentioned"})
	s.Require().NoError(err)
	s.Require().True(response.HasSectionUnreadCountsChanges())
	assertCounts(5, 1)

	_, err = s.m.MuteChat(&requests.MuteChat{ChatID: chatID, MutedType: MuteTillUnmuted})
	s.Require().NoError(err)
	assertCounts(2, 1)
	s.Require().NoError(s.m.UnmuteChat(chatID))
	assertCounts(5, 1)
	_, err = s.m.MarkAllRead(context.Background(), chatID)
	s.Require().NoError(err)
	assertCounts(5, 1)
	response, err = s.m.MarkThreadRead(context.Background(), chatID, "mentioned")
	s.Require().NoError(err)
	s.Require().True(response.HasSectionUnreadCountsChanges())
	assertCounts(3, 0)
}

func TestSectionUnreadCountsResponseInvalidation(t *testing.T) {
	emptyJSON, err := (&MessengerResponse{}).MarshalJSON()
	require.NoError(t, err)
	require.NotContains(t, string(emptyJSON), "sectionUnreadCountsChanged")
	tests := []struct {
		name string
		add  func(*MessengerResponse)
	}{
		{"chat", func(r *MessengerResponse) { r.AddChat(&Chat{ID: "chat"}) }},
		{"thread", func(r *MessengerResponse) { r.AddThread(&Thread{ThreadID: "thread"}) }},
		{"message", func(r *MessengerResponse) {
			r.AddMessage(&common.Message{ID: "message", ChatMessage: &protobuf.ChatMessage{}})
		}},
		{"removed-chat", func(r *MessengerResponse) { r.AddRemovedChat("chat") }},
		{"removed-message", func(r *MessengerResponse) {
			r.AddRemovedMessage(&RemovedMessage{ChatID: "chat", MessageID: "message"})
		}},
		{"read", func(r *MessengerResponse) { r.AddSeenAndUnseenMessages(&SeenUnseenMessages{ChatID: "chat"}) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := &MessengerResponse{}
			require.False(t, response.HasSectionUnreadCountsChanges())
			test.add(response)
			merged := &MessengerResponse{}
			require.NoError(t, merged.Merge(response))
			require.True(t, merged.HasSectionUnreadCountsChanges())
			data, err := merged.MarshalJSON()
			require.NoError(t, err)
			var decoded map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(data, &decoded))
			require.JSONEq(t, `true`, string(decoded["sectionUnreadCountsChanged"]))
		})
	}
}

func BenchmarkSectionUnreadCountsLargeAccount(b *testing.B) {
	p := sectionCountsTestDB(b)
	// 100 communities, 1,000 chats, 20,000 threads, 200,000 messages.
	_, err := p.db.Exec(`
		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i < 100)
		INSERT INTO communities_communities(id, description, joined)
		SELECT CAST(printf('%04d', i) AS BLOB), X'', 1 FROM n;

		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i < 10)
		INSERT INTO chats(id, name, type, timestamp, community_id, unviewed_message_count, unviewed_mentions_count)
		SELECT hex(c.id) || '-' || n.i, '', 6, 0, '0x' || lower(hex(c.id)), 3, 1
		FROM communities_communities c CROSS JOIN n;

		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i < 20)
		INSERT INTO threads(thread_id, chat_id, parent_message_id, name, followed, last_activity_at)
		SELECT c.id || '-' || n.i, c.id, '', '', 1, ? FROM chats c CROSS JOIN n;

		WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i < 10)
		INSERT INTO user_messages(id, local_chat_id, chat_id, thread_id, whisper_timestamp,
			source, text, content_type, timestamp, clock_value, seen, mentioned)
		SELECT t.thread_id || '-' || n.i, t.chat_id, t.chat_id, t.thread_id,
			0, '', '', 1, 0, n.i, n.i > 2, n.i = 1 FROM threads t CROSS JOIN n;
	`, p.threadNow())
	require.NoError(b, err)

	for _, includeThreads := range []bool{false, true} {
		b.Run(fmt.Sprintf("includeThreads=%t", includeThreads), func(b *testing.B) {
			counts, err := p.SectionUnreadCounts(context.Background(), includeThreads)
			require.NoError(b, err)
			require.Len(b, counts, 101)
			unread, mentions := uint64(30), uint64(10)
			if includeThreads {
				unread, mentions = 430, 210
			}
			require.Equal(b, unread, counts[1].UnviewedMessagesCount)
			require.Equal(b, mentions, counts[1].UnviewedMentionsCount)
			encoded, err := json.Marshal(SectionUnreadCountsResponse{Sections: counts})
			require.NoError(b, err)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := p.SectionUnreadCounts(context.Background(), includeThreads)
				if err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(len(encoded)), "payload-bytes")
		})
	}
}
