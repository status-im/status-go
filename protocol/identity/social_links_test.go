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

	protobufLinks := []*protobuf.SocialLink{}
	transformedLinks := NewSocialLinks(protobufLinks)
	require.False(t, socialLinks.Equals(*transformedLinks))

	protobufLinks = append(protobufLinks, &protobuf.SocialLink{Text: "A", Url: "B"})
	protobufLinks = append(protobufLinks, &protobuf.SocialLink{Text: "X", Url: "Y"})
	transformedLinks = NewSocialLinks(protobufLinks)
	require.True(t, socialLinks.Equals(*transformedLinks))
}
