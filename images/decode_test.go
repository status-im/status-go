package images

import (
	"image"
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
