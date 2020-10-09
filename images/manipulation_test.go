package images

import (
	"bytes"
	"errors"
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResize(t *testing.T) {
	sizes := []uint{80, 240, 1000}

	cs := []struct{
		Filename string
		Bounds map[uint]image.Rectangle
	}{
		{
			"elephant.jpg",
			map[uint]image.Rectangle{
				80:	{
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 80, Y: 80},
				},
				240: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 240, Y: 240},
				},
				1000: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 1000, Y: 1000},
				},
			},
		},
		{
			"rose.webp",
			map[uint]image.Rectangle{
				80:	{
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 107, Y: 80},
				},
				240: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 319, Y: 240},
				},
				1000: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 1329, Y: 1000},
				},
			},
		},
		{
			"spin.gif",
			map[uint]image.Rectangle{
				80:	{
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 80, Y: 80},
				},
				240: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 240, Y: 240},
				},
				1000: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 1000, Y: 1000},
				},
			},
		},
		{
			"status.png",
			map[uint]image.Rectangle{
				80:	{
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 80, Y: 80},
				},
				240: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 240, Y: 240},
				},
				1000: {
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: 1000, Y: 1000},
				},
			},
		},
	}

	for _, c := range cs {
		img, err := Decode(path + c.Filename)
		require.NoError(t, err)

		for _, s := range sizes {
			rsImg := Resize(s, img)
			require.Exactly(t, c.Bounds[s], rsImg.Bounds())
		}
	}
}

func TestCrop(t *testing.T) {
	type params struct{
		Rectangle image.Rectangle
		OutputBound image.Rectangle
		OutputSize int
		CropError error
	}

	topLeftSquare := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: 80, Y: 80},
	}
	offsetSquare := image.Rectangle{
		Min: image.Point{X: 80,	Y: 80},
		Max: image.Point{X: 160, Y: 160},
	}
	outOfBoundsSquare := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: 1000000, Y: 1000000},
	}
	rect := image.Rectangle{}
	options := &Details{
		Quality: 70,
	}

	cs := []struct{
		Filename string
		Params   []params
	}{
		{
			"elephant.jpg",
			[]params{
				{topLeftSquare, topLeftSquare, 1447, nil},
				{offsetSquare, rect, 0, errors.New("crop dimensions out of bounds of image, image width '80px' & height '80px'; crop bottom right coordinate at X '160px' Y '160px'")},
				{outOfBoundsSquare, rect, 0, errors.New("crop dimensions out of bounds of image, image width '80px' & height '80px'; crop bottom right coordinate at X '1000000px' Y '1000000px'")},
			},
		},
		{
			"rose.webp",
			[]params{
				{topLeftSquare, topLeftSquare, 1183, nil},
				{offsetSquare, offsetSquare, 1251, nil},
				{outOfBoundsSquare, rect, 0, errors.New("crop dimensions out of bounds of image, image width '400px' & height '301px'; crop bottom right coordinate at X '1000000px' Y '1000000px'")},
			},
		},
		{
			"spin.gif",
			[]params{
				{topLeftSquare, topLeftSquare, 693, nil},
				{offsetSquare, offsetSquare, 1339, nil},
				{outOfBoundsSquare, rect, 0, errors.New("crop dimensions out of bounds of image, image width '256px' & height '256px'; crop bottom right coordinate at X '1000000px' Y '1000000px'")},
			},
		},
		{
			"status.png",
			[]params{
				{topLeftSquare, topLeftSquare, 1027, nil},
				{offsetSquare, offsetSquare, 1157, nil},
				{outOfBoundsSquare, rect, 0, errors.New("crop dimensions out of bounds of image, image width '256px' & height '256px'; crop bottom right coordinate at X '1000000px' Y '1000000px'")},
			},
		},
	}

	for _, c := range cs {
		img, err := Decode(path + c.Filename)
		require.NoError(t, err)

		for _, p := range c.Params {
			cImg, err := Crop(img, p.Rectangle)
			if p.CropError != nil {
				require.EqualError(t, err, p.CropError.Error())
				continue
			} else {
				require.NoError(t, err)
			}
			require.Exactly(t, p.OutputBound.Dx(), cImg.Bounds().Dx(), c.Filename)
			require.Exactly(t, p.OutputBound.Dy(), cImg.Bounds().Dy(), c.Filename)

			bb := bytes.NewBuffer([]byte{})
			err = Encode(bb, cImg, options)
			require.NoError(t, err)
			require.Exactly(t, p.OutputSize, bb.Len())
		}
	}
}
