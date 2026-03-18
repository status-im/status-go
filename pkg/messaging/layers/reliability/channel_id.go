package reliability

import cryptotypes "github.com/status-im/status-go/internal/crypto/types"

const sdsChannelIDSeparator = "|"

// BuildChannelID returns the SDS channel ID format for community public
// messages. Keep the format centralized so sender/receiver code changes in one
// place.
func BuildChannelID(communityID []byte, contentTopic string) string {
	return cryptotypes.EncodeHex(communityID) + sdsChannelIDSeparator + contentTopic
}
