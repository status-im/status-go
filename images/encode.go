package images

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io"
	"regexp"
)

type EncodeConfig struct {
	Quality int
}

func Encode(w io.Writer, img image.Image, config EncodeConfig) error {
	// Currently a wrapper for renderJpeg, but this function is useful if multiple render formats are needed
	return renderJpeg(w, img, config)
}

func renderJpeg(w io.Writer, m image.Image, config EncodeConfig) error {
	o := new(jpeg.Options)
	o.Quality = config.Quality

	return jpeg.Encode(w, m, o)
}

func EncodeToLimits(bb *bytes.Buffer, img image.Image, bounds FileSizeLimits) error {
	q := MaxJpegQuality
	for q > MinJpegQuality-1 {

		err := Encode(bb, img, EncodeConfig{Quality: q})
		if err != nil {
			return err
		}

		if bounds.Ideal > bb.Len() {
			return nil
		}

		if q == MinJpegQuality {
			if bounds.Max > bb.Len() {
				return nil
			}
			return fmt.Errorf(
				"image size after processing exceeds max, expect < '%d', received < '%d'",
				bounds.Max,
				bb.Len(),
			)
		}

		bb.Reset()
		q -= 2
	}

	return nil
}

// CompressToFileLimits takes an image.Image and analyses the pixel dimensions, if the longest side is greater
// than the `longSideMax` image.Image will be resized, before compression begins.
// Next the image.Image is repeatedly encoded and resized until the data fits within
// the given FileSizeLimits. There is no limit on the number of times the cycle is performed, the image.Image
// is reduced to 95% of its size at the end of every round the file size exceeds the given limits.
func CompressToFileLimits(bb *bytes.Buffer, img image.Image, bounds FileSizeLimits) error {
	longSideMax := 2000

	// Do we need to do a pre-compression resize?
	if img.Bounds().Max.X > img.Bounds().Max.Y {
		// X is longer
		if img.Bounds().Max.X > longSideMax {
			img = resize.Resize(uint(longSideMax), 0, img, resize.Bilinear)
		}
	} else {
		// Y is longer or equal
		if img.Bounds().Max.Y > longSideMax {
			img = resize.Resize(0, uint(longSideMax), img, resize.Bilinear)
		}
	}

	for {
		err := EncodeToLimits(bb, img, bounds)
		if err == nil {
			return nil
		}
		if err.Error()[:50] != "image size after processing exceeds max, expect < " {
			return err
		}

		img = ResizeTo(95, img)
	}
}

func EncodeToBestSize(bb *bytes.Buffer, img image.Image, size ResizeDimension) error {
	return EncodeToLimits(bb, img, DimensionSizeLimit[size])
}

func GetPayloadDataURI(payload []byte) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}

	mt, err := GetMimeType(payload)
	if err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(payload)

	return "data:image/" + mt + ";base64," + b64, nil
}

func GetPayloadFromURI(uri string) ([]byte, error) {
	re := regexp.MustCompile("^data:image/(.*?);base64,(.*?)$")
	res := re.FindStringSubmatch(uri)
	if len(res) != 3 {
		return nil, errors.New("wrong uri format")
	}
	return base64.StdEncoding.DecodeString(res[2])
}
