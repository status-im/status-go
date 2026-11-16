package images

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"regexp"
	"strings"
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

type FileSizeError struct {
	expected int
	received int
}

func (e *FileSizeError) Error() string {
	return fmt.Sprintf("image size after processing exceeds max, expected < '%d', received < '%d'", e.expected, e.received)
}

// EncodeToLimits encodes img as JPEG at the highest quality on the ladder MaxJpegQuality,
// MaxJpegQuality-2, ..., MinJpegQuality whose output is smaller than bounds.Ideal. When none is,
// it uses MinJpegQuality and accepts the result if it is smaller than bounds.Max, otherwise it
// returns a *FileSizeError (bb then holds the MinJpegQuality output).
//
// The output size grows with the quality, so instead of walking the ladder (up to 16 encodes)
// it tries MaxJpegQuality and then binary-searches the rest: at most 1 + 4 encodes, same result.
func EncodeToLimits(bb *bytes.Buffer, img image.Image, bounds FileSizeLimits) error {
	var ladder []int
	for q := MaxJpegQuality; q >= MinJpegQuality; q -= 2 {
		ladder = append(ladder, q)
	}

	encoded := make(map[int][]byte, 6)
	encodeAt := func(i int) ([]byte, error) {
		if data, ok := encoded[i]; ok {
			return data, nil
		}
		var b bytes.Buffer
		if err := Encode(&b, img, EncodeConfig{Quality: ladder[i]}); err != nil {
			return nil, err
		}
		encoded[i] = b.Bytes()
		return encoded[i], nil
	}

	// Most small images fit at the highest quality: settle them with a single encode.
	data, err := encodeAt(0)
	if err != nil {
		return err
	}
	if bounds.Ideal > len(data) {
		bb.Write(data)
		return nil
	}

	// Find the highest remaining quality (lowest index) whose output is under Ideal.
	lo, hi, best := 1, len(ladder)-1, -1
	for lo <= hi {
		mid := (lo + hi) / 2
		data, err = encodeAt(mid)
		if err != nil {
			return err
		}
		if bounds.Ideal > len(data) {
			best, hi = mid, mid-1
		} else {
			lo = mid + 1
		}
	}
	if best >= 0 {
		bb.Write(encoded[best])
		return nil
	}

	// Nothing under Ideal: fall back to the lowest quality and check it against Max.
	data, err = encodeAt(len(ladder) - 1)
	if err != nil {
		return err
	}
	bb.Write(data)
	if bounds.Max > len(data) {
		return nil
	}
	return &FileSizeError{expected: bounds.Max, received: len(data)}
}

// longSideMax is the longest side, in pixels, of an image produced by CompressToFileLimits.
const longSideMax = 2000

// CompressToFileLimits takes an image.Image and analyses the pixel dimensions, if the longest side is greater
// than the `longSideMax` image.Image will be resized, before compression begins.
//
// An image with transparent pixels is encoded as PNG to keep its transparency, as long as that fits
// within bounds.Max; otherwise it is flattened onto white and compressed as JPEG like any other image.
//
// Next the image.Image is repeatedly encoded and resized until the data fits within
// the given FileSizeLimits. There is no limit on the number of times the cycle is performed, the image.Image
// is reduced to 95% of its size at the end of every round the file size exceeds the given limits.
func CompressToFileLimits(bb *bytes.Buffer, img image.Image, bounds FileSizeLimits) error {
	// Do we need to do a pre-compression resize?
	if img.Bounds().Max.X > img.Bounds().Max.Y {
		// X is longer
		if img.Bounds().Max.X > longSideMax {
			img = resizeImage(img, longSideMax, 0)
		}
	} else {
		// Y is longer or equal
		if img.Bounds().Max.Y > longSideMax {
			img = resizeImage(img, 0, longSideMax)
		}
	}

	if hasTransparency(img) {
		var pb bytes.Buffer
		if err := png.Encode(&pb, img); err != nil {
			return err
		}
		if bounds.Max >= pb.Len() {
			bb.Write(pb.Bytes())
			return nil
		}
		// JPEG has no alpha: without this, transparent areas would turn black.
		img = flattenOnWhite(img)
	}

	for {
		bb.Reset()
		err := EncodeToLimits(bb, img, bounds)
		if err == nil {
			return nil
		}
		// If error is not a FileSizeError then we need to return it up
		if fse := (*FileSizeError)(nil); !errors.As(err, &fse) {
			return err
		}

		img = Scale(95, img)
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

func IsPayloadDataURI(uri string) bool {
	return strings.HasPrefix(uri, "data:image")
}
