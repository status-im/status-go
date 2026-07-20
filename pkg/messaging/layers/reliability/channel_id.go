package reliability

// BuildChannelID returns the SDS channel ID format for community public
// messages. Keep the format centralized so sender/receiver code changes in one
// place.
func BuildChannelID(chatID string) string {
	return chatID
}
