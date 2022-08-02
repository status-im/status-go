package identity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/protobuf"
)

func TestEquals(t *testing.T) {
	socialLinks := SocialLinks{
		{
			Text: "A",
			URL:  "B",
		},
		{
			Text: "X",
			URL:  "Y",
		},
	}

	other := []*protobuf.SocialLink{}
	require.False(t, socialLinks.EqualsProtobuf(other))

	other = append(other, &protobuf.SocialLink{Text: "A", Url: "B"})
	other = append(other, &protobuf.SocialLink{Text: "X", Url: "Y"})
	require.True(t, socialLinks.EqualsProtobuf(other))

	// order does not matter
	other = []*protobuf.SocialLink{}
	other = append(other, &protobuf.SocialLink{Text: "X", Url: "Y"})
	other = append(other, &protobuf.SocialLink{Text: "A", Url: "B"})
	require.True(t, socialLinks.EqualsProtobuf(other))
}
