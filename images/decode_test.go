package images

import (
	"errors"
	"image"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	path = "../_assets/tests/"
)

var (
	testJpegBytes = []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x84, 0x00, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50}
	testPngBytes  = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48}
	testGifBytes  = []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x00, 0x01, 0x00, 0x01, 0x84, 0x1f, 0x00, 0xff}
	testWebpBytes = []byte{0x52, 0x49, 0x46, 0x46, 0x90, 0x49, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50, 0x56, 0x50}
	testAacBytes  = []byte{0xff, 0xf1, 0x50, 0x80, 0x1c, 0x3f, 0xfc, 0xda, 0x00, 0x4c, 0x61, 0x76, 0x63, 0x35}
)

func TestDecode(t *testing.T) {

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
		img, err := Decode(path + c.Filepath)

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

func TestGetType(t *testing.T) {
	cs := []struct {
		Buf   []byte
		Value ImageType
	}{
		{testJpegBytes, JPEG},
		{testPngBytes, PNG},
		{testGifBytes, GIF},
		{testWebpBytes, WEBP},
		{testAacBytes, UNKNOWN},
	}

	for _, c := range cs {
		require.Exactly(t, c.Value, GetType(c.Buf))
	}
}

func TestGetMimeType(t *testing.T) {
	cs := []struct {
		Buf   []byte
		Value string
		Error error
	}{
		{testJpegBytes, "jpeg", nil},
		{testPngBytes, "png", nil},
		{testGifBytes, "gif", nil},
		{testWebpBytes, "webp", nil},
		{testAacBytes, "", errors.New("image format not supported")},
	}

	for _, c := range cs {
		mt, err := GetMimeType(c.Buf)
		if c.Error == nil {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, c.Error.Error())
		}

		require.Exactly(t, c.Value, mt)
	}
}
