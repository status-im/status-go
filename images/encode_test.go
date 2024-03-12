package images

import (
	"bytes"
	"errors"
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
	options := EncodeConfig{
		Quality: 70,
	}

	for _, c := range cs {
		img, err := Decode(path + c.FileName)
		require.NoError(t, err)

		bb := bytes.NewBuffer([]byte{})
		err = Encode(bb, img, options)
		require.NoError(t, err)

		require.Exactly(t, c.RenderSize, bb.Len())
	}
}

func TestEncodeToBestSize(t *testing.T) {
	cs := []struct {
		FileName   string
		RenderSize int
		Error      error
	}{
		{
			"elephant.jpg",
			1467,
			nil,
		},
		{
			"rose.webp",
			8513,
			errors.New("image size after processing exceeds max, expected < '5632', received < '8513'"),
		},
		{
			"spin.gif",
			2407,
			nil,
		},
		{
			"status.png",
			4725,
			nil,
		},
	}

	for _, c := range cs {
		img, err := Decode(path + c.FileName)
		require.NoError(t, err)

		bb := bytes.NewBuffer([]byte{})
		err = EncodeToBestSize(bb, img, ResizeDimensions[0])

		require.Exactly(t, c.RenderSize, bb.Len())

		if c.Error != nil {
			require.EqualError(t, err, c.Error.Error())
		} else {
			require.NoError(t, err)
		}
	}
}

func TestCompressToFileLimits(t *testing.T) {
	img, err := Decode(path + "IMG_1205.HEIC.jpg")
	require.NoError(t, err)

	bb := bytes.NewBuffer([]byte{})
	err = CompressToFileLimits(bb, img, FileSizeLimits{50000, 350000})
	require.NoError(t, err)
	require.Equal(t, 291645, bb.Len())
}

func TestGetPayloadFromURI(t *testing.T) {
	payload, err := GetPayloadFromURI("data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=")
	require.NoError(t, err)
	require.Equal(
		t,
		[]byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50},
		payload,
	)
}

func TestIsSvg(t *testing.T) {
	GoodSVG := []byte(`<svg width="300" height="130" xmlns="http://www.w3.org/2000/svg">
	  <rect width="200" height="100" x="10" y="10" rx="20" ry="20" fill="blue" />
	  Sorry, your browser does not support inline SVG.  
	</svg>`)

	BadSVG := []byte(`<head>
	<link rel="stylesheet" href="styles.css">
  </head>`)

	require.Equal(t, IsSVG(BadSVG), false)
	require.Equal(t, IsSVG(GoodSVG), true)
}

func TestIsIco(t *testing.T) {
	GoodICO, err := _assetsTestsWikipediaIcoBytes()
	require.NoError(t, err)

	GoodPNG, err := _assetsTestsQrDefaultqrPngBytes()
	require.NoError(t, err)

	require.Equal(t, IsIco(GoodICO), true)
	require.Equal(t, IsIco(GoodPNG), false)
	require.Equal(t, IsPng(GoodPNG), true)
	require.Equal(t, IsPng(GoodICO), false)
}
