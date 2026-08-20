package protocol

import "testing"

func TestRequestMissingCommunityEncryptionKeysSkipsWhenPaused(t *testing.T) {
	messenger := &Messenger{}
	messenger.paused.Store(true)

	messenger.requestMissingCommunityEncryptionKeys()
}
