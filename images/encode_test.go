package images

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gen2brain/aac-go"
	"github.com/stretchr/testify/require"
	"github.com/youpy/go-wav"
	"io/ioutil"
	"os"
	"runtime/debug"
	"testing"
	"time"
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

type SamplingRate uint

const (
	Sampling8000Hz  SamplingRate = 8000
	Sampling11025Hz SamplingRate = 11025
	Sampling16000Hz SamplingRate = 16000
	Sampling22050Hz SamplingRate = 22050
	Sampling24000Hz SamplingRate = 24000
	Sampling32000Hz SamplingRate = 32000
	Sampling44100Hz SamplingRate = 44100
	Sampling48000Hz SamplingRate = 48000
	Sampling64000Hz SamplingRate = 64000
	Sampling96000Hz SamplingRate = 96000
)

var (
	SamplingRates = []SamplingRate{
		Sampling96000Hz,
		Sampling64000Hz,
		Sampling48000Hz,
		Sampling44100Hz,
		Sampling32000Hz,
		Sampling24000Hz,
		Sampling22050Hz,
		Sampling16000Hz,
		Sampling11025Hz,
		Sampling8000Hz,
	}
)

type BitRate uint

const (
	BitRate8000s BitRate = 8000 + iota*4000
	BitRate12000s
	BitRate16000s
	BitRate20000s
	BitRate24000s
	BitRate28000s
	BitRate32000s
	BitRate36000s
	BitRate40000s
	BitRate44000s
)

var (
	BitRates = []BitRate{
		BitRate44000s,
		BitRate40000s,
		BitRate36000s,
		BitRate32000s,
		BitRate28000s,
		BitRate24000s,
		BitRate20000s,
		BitRate16000s,
		BitRate12000s,
		BitRate8000s,
	}
)

func encodeToAAC(fn string, enc *aac.Encoder, wreader *wav.Reader) error {
	ec := make(chan error, 1)
	go func() {
		debug.SetPanicOnFault(true)
		defer func() {
			if r := recover(); r != nil {
				spew.Dump("Recovered panic:", r)
			}
		}()
		ec <- enc.Encode(wreader)
	}()

	select {
	case err := <-ec:
		return err
	case <-time.After(2 * time.Second):
		close(ec)
		spew.Dump("timeout encoding " + fn)
		return nil
	}
}

func TestWavToAAC(t *testing.T) {
	file, err := os.Open(path + "file_example_WAV_10MG.wav")
	if err != nil {
		require.NoError(t, err)
	}
	defer file.Close()

	wreader := wav.NewReader(file)
	f, err := wreader.Format()
	if err != nil {
		require.NoError(t, err)
	}

	spew.Dump(f)

	for _, sr := range SamplingRates {
		// TODO resolve the issue of encoding above original rates
		if f.SampleRate < uint32(sr) {
			//continue
		}
		for _, br := range BitRates {
			spew.Dump("bitrate", br)
			if f.ByteRate < uint32(br) {
				//continue
			}

			if sr == Sampling96000Hz && br == BitRate16000s {
				continue
			}

			opts := &aac.Options{
				SampleRate:  int(sr),
				BitRate:     int(br), // Yes BitRate == ByteRate, don't know why
				NumChannels: int(f.NumChannels),
			}

			spew.Dump(opts)

			buf := bytes.NewBuffer(make([]byte, 0))
			enc, err := aac.NewEncoder(buf, opts)
			if err != nil {
				require.NoError(t, err)
			}

			spew.Dump("Making new wav.Reader")
			wreader := wav.NewReader(file)
			spew.Dump("Encoding wav.Reader into aac")
			// TODO fix hanging by timing out processes that take too long

			fn := fmt.Sprintf(path+"test_test_%d_%d.aac", opts.SampleRate, opts.BitRate)

			err = encodeToAAC(fn, enc, wreader)
			if err != nil {
				require.NoError(t, err)
			}

			err = enc.Close()
			if err != nil {
				require.NoError(t, err)
			}

			spew.Dump("Making new audio file " + fn)
			err = ioutil.WriteFile(fn, buf.Bytes(), 0644)
			if err != nil {
				require.NoError(t, err)
			}
		}
	}
}
