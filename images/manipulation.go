package images

import (
	"fmt"
	"image"

	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"
)

func Resize(width, height uint, img image.Image) image.Image {
	return resize.Resize(width, height, img, resize.Bilinear)
}

func Crop(img image.Image, rect image.Rectangle) (image.Image, error) {

	if img.Bounds().Max.X < rect.Max.X || img.Bounds().Max.Y < rect.Max.Y{
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
