package reliability

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestBuildChannelID(t *testing.T) {
	communityID := "community-id"
	contentTopic := "community-chat"

	channelID := BuildChannelID(communityID, contentTopic)

	require.Equal(t, communityID+":"+contentTopic, channelID)
}

func TestBuildChannelIDWithoutContentTopic(t *testing.T) {
	communityID := "community-id"

	require.Equal(t, communityID, BuildChannelID(communityID, ""))
}

func TestExtractSDSContent(t *testing.T) {
	content := []byte("application payload")
	wrapped := protowire.AppendTag(nil, 1, protowire.BytesType)
	wrapped = protowire.AppendString(wrapped, "message-id")
	wrapped = protowire.AppendTag(wrapped, 2, protowire.VarintType)
	wrapped = protowire.AppendVarint(wrapped, 1)
	wrapped = protowire.AppendTag(wrapped, 4, protowire.BytesType)
	wrapped = protowire.AppendString(wrapped, "channel-id")
	wrapped = protowire.AppendTag(wrapped, 5, protowire.BytesType)
	wrapped = protowire.AppendBytes(wrapped, content)

	extracted, channelID, ok := ExtractSDSMessage(wrapped)
	require.True(t, ok)
	require.Equal(t, content, extracted)
	require.Equal(t, "channel-id", channelID)
	extracted, ok = ExtractSDSContent(wrapped)
	require.True(t, ok)
	require.Equal(t, content, extracted)

	mismatchedKnownField := append([]byte(nil), wrapped...)
	mismatchedKnownField = protowire.AppendTag(mismatchedKnownField, 5, protowire.VarintType)
	mismatchedKnownField = protowire.AppendVarint(mismatchedKnownField, 1)
	extracted, channelID, ok = ExtractSDSMessage(mismatchedKnownField)
	require.True(t, ok, "known fields with an unexpected wire type are skipped like unknown fields")
	require.Equal(t, content, extracted)
	require.Equal(t, "channel-id", channelID)

	wrapped = protowire.AppendTag(nil, 1, protowire.BytesType)
	wrapped = protowire.AppendString(wrapped, "message-id")
	wrapped = protowire.AppendTag(wrapped, 4, protowire.BytesType)
	wrapped = protowire.AppendString(wrapped, "channel-id")
	extracted, channelID, ok = ExtractSDSMessage(wrapped)
	require.True(t, ok, "SDS headers without application content must still be recognized")
	require.Nil(t, extracted)
	require.Equal(t, "channel-id", channelID)

	wrapped = protowire.AppendTag(nil, 5, protowire.BytesType)
	wrapped = protowire.AppendBytes(wrapped, content)
	extracted, channelID, ok = ExtractSDSMessage(wrapped)
	require.False(t, ok, "content without an SDS message ID and channel must not be identified as SDS")
	require.Nil(t, extracted)
	require.Empty(t, channelID)

	extracted, channelID, ok = ExtractSDSMessage([]byte{0x2a, 0x05, 0x01})
	require.False(t, ok, "malformed protobuf must not be identified as SDS")
	require.Nil(t, extracted)
	require.Empty(t, channelID)
}
