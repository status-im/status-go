package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func TestShouldCommunityMessageBeEncryptedForThreadDeletion(t *testing.T) {
	require.True(t, shouldCommunityMessageBeEncrypted(protobuf.ApplicationMetadataMessage_DELETE_THREAD))
}
