package reliability

// BuildChannelID scopes community SDS history to a content topic. Messages
// without a content topic retain the community-level channel.
func BuildChannelID(communityID, contentTopic string) string {
	if contentTopic == "" {
		return communityID
	}

	return communityID + ":" + contentTopic
}
