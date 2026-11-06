package protocol

import (
	"context"
	"fmt"
)

const PersonalSectionID = "personal"

// SectionUnreadCount contains badge-ready counts, including the existing mute
// exception for a chat or thread with mentions.
type SectionUnreadCount struct {
	SectionID             string `json:"sectionId"`
	UnviewedMessagesCount uint64 `json:"unviewedMessagesCount"`
	UnviewedMentionsCount uint64 `json:"unviewedMentionsCount"`
}

type SectionUnreadCountsResponse struct {
	Sections []SectionUnreadCount `json:"sections"`
}

func (m *Messenger) SectionUnreadCounts(ctx context.Context, includeThreads bool) (*SectionUnreadCountsResponse, error) {
	sections, err := m.persistence.SectionUnreadCounts(ctx, includeThreads && m.featureFlags.Threads)
	if err != nil {
		return nil, err
	}
	return &SectionUnreadCountsResponse{Sections: sections}, nil
}

func (db sqlitePersistence) SectionUnreadCounts(ctx context.Context, includeThreads bool) ([]SectionUnreadCount, error) {
	query, args := db.sectionUnreadCountsQuery(includeThreads)
	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query section unread counts: %w", err)
	}
	defer rows.Close()

	counts := make([]SectionUnreadCount, 0)
	for rows.Next() {
		var count SectionUnreadCount
		if err := rows.Scan(&count.SectionID, &count.UnviewedMessagesCount, &count.UnviewedMentionsCount); err != nil {
			return nil, fmt.Errorf("scan section unread counts: %w", err)
		}
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read section unread counts: %w", err)
	}
	return counts, nil
}

func (db sqlitePersistence) sectionUnreadCountsQuery(includeThreads bool) (string, []interface{}) {
	// Aggregate each source before combining it, so joining multiple threads or
	// messages cannot multiply a parent chat's persisted counters.
	query := `
		WITH sections AS (
			SELECT 'personal' AS section_id, 0 AS muted
			UNION ALL
			SELECT '0x' || lower(hex(id)), muted
			FROM communities_communities
			WHERE joined OR spectated
		), section_chats AS (
			SELECT c.id, c.type, c.muted OR s.muted AS muted, s.section_id,
				MAX(0, c.unviewed_message_count) AS unread,
				MAX(0, c.unviewed_mentions_count) AS mentions
			FROM chats c
			JOIN sections s ON s.section_id = CASE
				WHEN COALESCE(c.community_id, '') = '' THEN 'personal'
				ELSE c.community_id END
			WHERE c.active AND c.type != 0
		), contributions AS (
			SELECT section_id, CASE WHEN muted AND mentions = 0 THEN 0 ELSE unread END AS unread, mentions
			FROM section_chats`
	var args []interface{}
	if includeThreads {
		query += `
			UNION ALL
			SELECT section_id,
				CASE WHEN muted AND mentions = 0 THEN 0 ELSE unread END, mentions
			FROM (
				SELECT c.section_id, c.muted, t.thread_id,
					COUNT(1) AS unread,
					SUM(CASE WHEN um.mentioned OR um.replied OR c.type = ? THEN 1 ELSE 0 END) AS mentions
				FROM threads t
				JOIN section_chats c ON c.id = t.chat_id
				JOIN user_messages um ON um.local_chat_id = t.chat_id AND um.thread_id = t.thread_id
				WHERE t.followed AND NOT t.deleted AND t.last_activity_at >= ?
					AND c.type IN (?, ?, ?)
					AND NOT um.seen AND NOT um.hide AND NOT um.deleted AND NOT um.deleted_for_me
					AND um.thread_id != ''
				GROUP BY c.id, t.thread_id
			)`
		args = append(args, ChatTypeOneToOne, threadArchiveCutoff(db.threadNow()),
			ChatTypeOneToOne, ChatTypePrivateGroupChat, ChatTypeCommunityChat)
	}
	query += `
		)
		SELECT s.section_id, COALESCE(SUM(c.unread), 0), COALESCE(SUM(c.mentions), 0)
		FROM sections s
		LEFT JOIN contributions c ON c.section_id = s.section_id
		GROUP BY s.section_id
		ORDER BY s.section_id != 'personal', s.section_id`

	return query, args
}
