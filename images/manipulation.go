package images

import (
	"image"

	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"
)

func ResizeSquare(size uint, img image.Image) image.Image {
	return resize.Resize(size, 0, img, resize.Bilinear)
}

func CropSquare(img image.Image, size int, anchor image.Point) (image.Image, error) {
	return cutter.Crop(img, cutter.Config{
		Width:  size,
		Height: size,
		Anchor: anchor,
	})
}
