package utils

import (
	"github.com/status-im/status-go/internal/crypto/types"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TopicsAsByteArrays(topics []messagingtypes.ContentTopic) [][]byte {
	var topicsAsByteArrays [][]byte
	for _, t := range topics {
		topicsAsByteArrays = append(topicsAsByteArrays, t.Bytes())
	}
	return topicsAsByteArrays
}

// UniversalChatIDFromCommunityID constructs the universal chat ID from a community ID.
// This is used as a content-topic for all chats in the community.
// It corresponds to Community.UniversalChatID() which returns Community.MemberUpdateChannelID()
// which is IDString() + "-memberUpdate".
func UniversalChatIDFromCommunityID(communityID types.HexBytes) string {
	return types.EncodeHex(communityID) + "-memberUpdate"
}
