package images

import (
	"fmt"
	"image"
	"math"

	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"

	"github.com/ethereum/go-ethereum/log"
)

func Resize(size ResizeDimension, img image.Image) image.Image {
	var width, height uint

	switch {
	case img.Bounds().Max.X == img.Bounds().Max.Y:
		width, height = uint(size), uint(size)
	case img.Bounds().Max.X > img.Bounds().Max.Y:
		width, height = 0, uint(size)
	default:
		width, height = uint(size), 0
	}

	log.Info("resizing", "size", size, "width", width, "height", height)

	return resize.Resize(width, height, img, resize.Bilinear)
}

func ResizeTo(percent int, img image.Image) image.Image {
	width := uint(img.Bounds().Max.X * percent / 100)
	height := uint(img.Bounds().Max.Y * percent / 100)

	return resize.Resize(width, height, img, resize.Bilinear)
}

func ShrinkOnly(size ResizeDimension, img image.Image) image.Image {
	finalSize := int(math.Min(float64(size), math.Min(float64(img.Bounds().Dx()), float64(img.Bounds().Dy()))))
	return Resize(ResizeDimension(finalSize), img)
}

func Crop(img image.Image, rect image.Rectangle) (image.Image, error) {

	if img.Bounds().Max.X < rect.Max.X || img.Bounds().Max.Y < rect.Max.Y {
		return nil, fmt.Errorf(
			"crop dimensions out of bounds of image, image width '%dpx' & height '%dpx'; crop bottom right coordinate at X '%dpx' Y '%dpx'",
			img.Bounds().Max.X, img.Bounds().Max.Y,
			rect.Max.X, rect.Max.Y,
		)
	}

	return cutter.Crop(img, cutter.Config{
		Width:  rect.Dx(),
		Height: rect.Dy(),
		Anchor: rect.Min,
	})
}

// CropCenter takes an image, usually downloaded from a URL
// If the image is square, the full image is returned
// If the image is rectangular, the largest central square is returned
// calculations at _docs/image-center-crop-calculations.png
func CropCenter(img image.Image) (image.Image, error) {
	var cropRect image.Rectangle
	maxBounds := img.Bounds().Max

	if maxBounds.X == maxBounds.Y {
		return img, nil
	}

	if maxBounds.X > maxBounds.Y {
		// the final output should be YxY
		cropRect = image.Rectangle{
			Min: image.Point{X: maxBounds.X/2 - maxBounds.Y/2, Y: 0},
			Max: image.Point{X: maxBounds.X/2 + maxBounds.Y/2, Y: maxBounds.Y},
		}
	} else {
		// the final output should be XxX
		cropRect = image.Rectangle{
			Min: image.Point{X: 0, Y: maxBounds.Y/2 - maxBounds.X/2},
			Max: image.Point{X: maxBounds.X, Y: maxBounds.Y/2 + maxBounds.X/2},
		}
	}
	return Crop(img, cropRect)
}
