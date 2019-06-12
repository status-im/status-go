package adapters

// MailServerPassword is a password that is required
// to request messages from a Status mail server.
const MailServerPassword = "status-offline-inbox"

// Whisper message properties.
const (
	WhisperTTL     = 15
	WhisperPoW     = 0.002
	WhisperPoWTime = 5
)

// Whisper known topics.
const (
	TopicDiscovery = "contact-discovery"
)
