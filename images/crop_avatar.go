package images

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/fogleman/gg"
)

func CropAvatar(inputImage []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(inputImage))
	if err != nil {
		return nil, err
	}

	width := img.Bounds().Max.X
	radius := width / 2

	dc := gg.NewContext(width, width)
	dc.DrawImage(img, 0, 0)

	dc.SetColor(color.Transparent)
	for i := 0; i <= width; i++ {
		for j := 0; j <= width; j++ {
			if (math.Pow(float64(i-radius), 2) + math.Pow(float64(j-radius), 2)) > math.Pow(float64(radius), 2) {
				dc.SetPixel(int(i), int(j))
			}
		}
	}

	var outputImage bytes.Buffer
	err = png.Encode(&outputImage, dc.Image())
	if err != nil {
		return nil, err
	}
	return outputImage.Bytes(), nil
}
