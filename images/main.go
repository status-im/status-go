package images

import (
	"bytes"
	"image"
)

func GenerateProfileImages(filepath string, aX, aY, bX, bY int) ([][]byte, error) {
	img, err := Decode(filepath)
	if err != nil {
		return nil, err
	}

	cropRect := image.Rectangle{
		Min: image.Point{X: aX, Y: aY},
		Max: image.Point{X: bX, Y: bY},
	}
	cImg, err := Crop(img, cropRect)
	if err != nil {
		return nil, err
	}

	imgs := make([][]byte, len(ResizeDimensions))
	for _, s := range ResizeDimensions {
		rImg := Resize(s, cImg)

		bb := bytes.NewBuffer([]byte{})
		err = EncodeToBestSize(bb, rImg, s)
		if err != nil {
			return nil, err
		}
		imgs = append(imgs, bb.Bytes())
	}

	return imgs, nil
}
