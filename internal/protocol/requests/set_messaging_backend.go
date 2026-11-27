package requests

// SetMessagingBackend sets the messaging backend the next login creates:
// "go-waku", "logos-delivery", or empty for the default of the build.
type SetMessagingBackend struct {
	Backend string `json:"backend"`
}
