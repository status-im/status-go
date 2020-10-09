package images

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	cs := []struct {
		FileName   string
		RenderSize int
	}{
		{
			"elephant.jpg",
			1447,
		},
		{
			"rose.webp",
			11119,
		},
		{
			"spin.gif",
			2263,
		},
		{
			"status.png",
			5834,
		},
	}
	options := Details{
		Quality: 70,
	}

	for _, c := range cs {
		img, err := Decode(path + c.FileName)
		require.NoError(t, err)

		bb := bytes.NewBuffer([]byte{})
		err = Encode(bb, img, &options)
		require.NoError(t, err)

		require.Exactly(t, c.RenderSize, bb.Len())
	}
}
