package images

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
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

func Test(t *testing.T) {
	// Test AAC file compression with gzip
	f, err := ioutil.ReadFile(path + "sample3.aac")
	require.NoError(t, err)
	spew.Dump(len(f)) // (int) 1758426

	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(f)
	w.Close()
	spew.Dump(b.Len()) // (int) 1733347
}

func TestWavToAAC(t *testing.T) {
	file, err := os.Open(path + "BabyElephantWalk60.wav")
	if err != nil {
		panic(err)
	}

	wreader := wav.NewReader(file)
	f, err := wreader.Format()
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer(make([]byte, 0))

	opts := &aac.Options{}
	opts.SampleRate = int(f.SampleRate)
	opts.NumChannels = int(f.NumChannels)

	enc, err := aac.NewEncoder(buf, opts)
	if err != nil {
		panic(err)
	}

	err = enc.Encode(wreader)
	if err != nil {
		panic(err)
	}

	err = enc.Close()
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("test.aac", buf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}
