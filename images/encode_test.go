package images

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
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
		img, err := Get(path + c.FileName)
		require.NoError(t, err)

		bb := bytes.NewBuffer([]byte{})
		err = Render(bb, img, &options)
		require.NoError(t, err)

		require.Exactly(t, c.RenderSize, bb.Len())
	}
}

func TestMakeAndRenderFile(t *testing.T) {
	cs := []struct {
		FileName   string
		OutName    string
		OutputSize int64
	}{
		{
			"elephant.jpg",
			"_elephant.jpg",
			1447,
		},
		{
			"rose.webp",
			"_rose.jpg",
			11119,
		},
		{
			"spin.gif",
			"_spin.jpg",
			2263,
		},
		{
			"status.png",
			"_status.jpg",
			5834,
		},
	}

	for _, c := range cs {
		img, err := Get(path + c.FileName)
		require.NoError(t, err)

		options := &Details{
			FileName: path + c.OutName,
			Quality:  70,
		}

		err = RenderAndMakeFile(img, options)
		require.NoError(t, err)
		require.Exactly(t, c.OutputSize, options.SizeFile)

		// tidy up
		err = os.Remove(options.FileName)
		require.NoError(t, err)
	}
}
