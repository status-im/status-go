package images

import (
	"github.com/stretchr/testify/require"
	"image"
	"testing"
)

func TestGet(t *testing.T) {

	cs := []struct{
		Filepath string
		Error bool
		Nil bool
		Bounds image.Rectangle
	}{
		{
			"../_assets/tests/elephant.jpg",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 80, Y: 80},
			},
		},
		{
			"../_assets/tests/status.png",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 256, Y: 256},
			},
		},
		{
			"../_assets/tests/spin.gif",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 256, Y: 256},
			},
		},
		{
			"../_assets/tests/rose.webp",
			false,
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 400, Y: 301},
			},
		},
		{
			"../_assets/tests/test.aac",
			true,
			true,
			image.Rectangle{},
		},
	}

	for _, test := range cs {
		img, err := Get(test.Filepath)

		if test.Error {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		if test.Nil {
			require.Nil(t, img)
			continue
		} else {
			require.NotNil(t, img)
		}

		require.Exactly(t, test.Bounds, img.Bounds())
	}
}
