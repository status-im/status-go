package types

import (
	"encoding/hex"
	"strings"
)

const (
	communityChatTopicSuffix = "-memberUpdate"
	scopedSDSTopicSuffix     = "-sds-v2"
)

// CommunityChatTopic is the existing shared Waku content topic for community chats.
func CommunityChatTopic(communityID string) string {
	return communityID + communityChatTopicSuffix
}

// ScopedSDSContentTopic isolates topic-scoped copies from older subscribers.
func ScopedSDSContentTopic(legacyTopic string) string {
	return legacyTopic + scopedSDSTopicSuffix
}

// IsCommunityChatTopic recognizes both community chat topics during the overlap.
func IsCommunityChatTopic(topic string) bool {
	topic = strings.TrimSuffix(topic, scopedSDSTopicSuffix)
	id, ok := strings.CutSuffix(topic, communityChatTopicSuffix)
	if !ok || len(id) != 68 || !strings.HasPrefix(id, "0x") {
		return false
	}
	var key [33]byte
	_, err := hex.Decode(key[:], []byte(id[2:]))
	return err == nil && (key[0] == 2 || key[0] == 3)
}
