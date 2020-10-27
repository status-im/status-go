package images

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
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

func EncodeToBestSize(bb *bytes.Buffer, img image.Image, size ResizeDimension) error {
	q := MaxJpegQuality
	for q > MinJpegQuality-1 {

		err := Encode(bb, img, EncodeConfig{Quality: q})
		if err != nil {
			return err
		}

		if DimensionSizeLimit[size].Ideal > bb.Len() {
			return nil
		}

		if q == MinJpegQuality {
			if DimensionSizeLimit[size].Max > bb.Len() {
				return nil
			} else {
				return fmt.Errorf(
					"image size after processing exceeds max, expect < '%d', received < '%d'",
					DimensionSizeLimit[size].Max,
					bb.Len(),
				)
			}
		}

		bb.Reset()
		q -= 2
	}

	return nil
}
