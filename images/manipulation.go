package images

import (
	"fmt"
	"image"

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

// CropImage takes an image, usually downloaded from a URL
// If the image is square, the full image is returned
// It the image is rectangular, the largest central square is returned
// calculations here
func CropCenter(img image.Image) (image.Image, error) {
	maxBounds := img.Bounds().Max

	if maxBounds.X == maxBounds.Y {
		return img, nil
	} else {
		if maxBounds.X > maxBounds.Y {
			// the final output should be YxY
			cropRect := image.Rectangle{
				Min: image.Point{X: maxBounds.X/2 - maxBounds.Y/2, Y: 0},
				Max: image.Point{X: maxBounds.X/2 + maxBounds.Y/2, Y: maxBounds.Y},
			}
			return Crop(img, cropRect)
		} else {
			// the final output should be XxX
			cropRect := image.Rectangle{
				Min: image.Point{X: 0, Y: maxBounds.Y/2 - maxBounds.X/2},
				Max: image.Point{X: maxBounds.X, Y: maxBounds.Y/2 + maxBounds.X/2},
			}
			return Crop(img, cropRect)
		}
	}
}
