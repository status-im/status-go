package adapters

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol"
	"github.com/status-im/status-go/internal/protocol/contacts"
)

func TestThreadPreviewImage(t *testing.T) {
	photo := func(size int) images.IdentityImage {
		var buffer bytes.Buffer
		require.NoError(t, png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, size, size))))
		return images.IdentityImage{Payload: buffer.Bytes()}
	}
	large, small := photo(4), photo(2)
	largeURI, err := large.GetDataURI()
	require.NoError(t, err)
	smallURI, err := small.GetDataURI()
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		chat     protocol.Chat
		contact  *contacts.Contact
		expected string
	}{
		{name: "group photo", chat: protocol.Chat{ChatType: protocol.ChatTypePrivateGroupChat, Base64Image: largeURI}, expected: largeURI},
		{name: "group without photo", chat: protocol.Chat{ChatType: protocol.ChatTypePrivateGroupChat, Identicon: largeURI}},
		{name: "contact large photo", chat: protocol.Chat{ChatType: protocol.ChatTypeOneToOne}, contact: &contacts.Contact{Images: map[string]images.IdentityImage{images.LargeDimName: large, images.SmallDimName: small}}, expected: largeURI},
		{name: "contact small photo", chat: protocol.Chat{ChatType: protocol.ChatTypeOneToOne}, contact: &contacts.Contact{Images: map[string]images.IdentityImage{images.SmallDimName: small}}, expected: smallURI},
		{name: "empty large photo", chat: protocol.Chat{ChatType: protocol.ChatTypeOneToOne}, contact: &contacts.Contact{Images: map[string]images.IdentityImage{images.LargeDimName: {}, images.SmallDimName: small}}, expected: smallURI},
		{name: "invalid large photo", chat: protocol.Chat{ChatType: protocol.ChatTypeOneToOne}, contact: &contacts.Contact{Images: map[string]images.IdentityImage{images.LargeDimName: {Payload: []byte("invalid")}, images.SmallDimName: small}}, expected: smallURI},
		{name: "contact without photo", chat: protocol.Chat{ChatType: protocol.ChatTypeOneToOne}, contact: &contacts.Contact{Identicon: largeURI}},
		{name: "unknown contact", chat: protocol.Chat{ChatType: protocol.ChatTypeOneToOne}},
		{name: "community uses banner separately", chat: protocol.Chat{ChatType: protocol.ChatTypeCommunityChat, Base64Image: largeURI}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, threadPreviewImage(&tc.chat, tc.contact))
		})
	}
}
