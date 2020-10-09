package images

import (
	"bytes"
	"image"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	path = "../_assets/tests/"
)

func TestGet(t *testing.T) {

	cs := []struct {
		Filepath string
		Error    bool
		Nil      bool
		Bounds   image.Rectangle
	}{
		{
			"elephant.jpg",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 80, Y: 80},
			},
		},
		{
			"status.png",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 256, Y: 256},
			},
		},
		{
			"spin.gif",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 256, Y: 256},
			},
		},
		{
			"rose.webp",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 400, Y: 301},
			},
		},
		{
			"test.aac",
			true,
			true,
			image.Rectangle{},
		},
	}

	for _, c := range cs {
		img, err := Get(path + c.Filepath)

		if c.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		if c.Nil {
			require.Nil(t, img)
			continue
		} else {
			require.NotNil(t, img)
		}

		require.Exactly(t, c.Bounds, img.Bounds())
	}
}

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
