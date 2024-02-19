package images

import (
	"errors"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	path = "../_assets/tests/"
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

func TestDecodeFromURL(t *testing.T) {
	s := httptest.NewServer(http.FileServer(http.Dir(path)))
	defer s.Close()

	cs := []struct {
		Filepath string
		Nil      bool
		Bounds   image.Rectangle
	}{
		{
			s.URL + "/2x1.png",
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 2, Y: 1},
			},
		},
		{
			s.URL + "/1.jpg",
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 1, Y: 1},
			},
		},
		{
			s.URL + "/1.gif",
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 1, Y: 1},
			},
		},
		{
			s.URL + "/1.webp", // nolint: goconst
			false,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 1, Y: 1},
			},
		},
		{
			s.URL + "/1.webp", // nolint: goconst
			true,
			image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: 10, Y: 10},
			},
		},
	}

	for _, c := range cs {
		img, err := DecodeFromURL(c.Filepath)

		if c.Nil {
			require.Nil(t, err)
		} else {
			require.NoError(t, err)
			require.Exactly(t, c.Bounds, img.Bounds())
		}

	}
}

func TestDecodeFromURL_WithErrors(t *testing.T) {
	s := httptest.NewServer(http.FileServer(http.Dir(path)))
	defer s.Close()

	_, err := DecodeFromURL("https://doesnt-exist.com")
	require.Error(t, err)
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
