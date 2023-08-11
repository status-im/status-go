package images

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/fogleman/gg"
)

func AddStatusIndicatorToImage(inputImage []byte, innerColor color.Color, indicatorSize, indicatorBorder float64) ([]byte, error) {
	// decode the input image
	img, _, err := image.Decode(bytes.NewReader(inputImage))
	if err != nil {
		return nil, err
	}

	// get the dimensions of the image
	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y

	indicatorRadius := indicatorSize / 2

	// calculate the center point
	x := float64(width) - indicatorRadius
	y := float64(height) - indicatorRadius

	// create a new gg.Context instance
	dc := gg.NewContext(width, height)
	dc.DrawImage(img, 0, 0)

	// Loop through each pixel in the hole and set it to transparent
	dc.SetColor(color.Transparent)
	for i := x - indicatorRadius; i <= x+indicatorRadius; i++ {
		for j := y - indicatorRadius; j <= y+indicatorRadius; j++ {
			if math.Pow(i-x, 2)+math.Pow(j-y, 2) <= math.Pow(indicatorRadius, 2) {
				dc.SetPixel(int(i), int(j))
			}
		}
	}

	// draw inner circle
	dc.DrawCircle(x, y, indicatorRadius-indicatorBorder)
	dc.SetColor(innerColor)
	dc.Fill()

	// encode the modified image as PNG and return as []byte
	var outputImage bytes.Buffer
	err = png.Encode(&outputImage, dc.Image())
	if err != nil {
		return nil, err
	}
	return outputImage.Bytes(), nil
}
