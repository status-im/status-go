package images

import (
	"bytes"
	"image"
	"strings"
)


func GetDecodedImage(filepath string) (image.Image, error) {
	var err error
	var img image.Image

	if strings.HasPrefix(filepath, "http") {
		img, err = DecodeFromURL(filepath)
	} else {
		img, err = Decode(filepath)
	}

	if err != nil {
		return nil, err
	}

	return img, nil
}

func GenerateIdentityImages(filepath string, aX, aY, bX, bY int) ([]*IdentityImage, error) {
	img, err := GetDecodedImage(filepath)
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

	var iis []*IdentityImage
	for _, s := range ResizeDimensions {
		rImg := Resize(s, cImg)

		bb := bytes.NewBuffer([]byte{})
		err = EncodeToBestSize(bb, rImg, s)
		if err != nil {
			return nil, err
		}

		ii := &IdentityImage{
			Name:         ResizeDimensionToName[s],
			Payload:      bb.Bytes(),
			Width:        rImg.Bounds().Dx(),
			Height:       rImg.Bounds().Dy(),
			FileSize:     bb.Len(),
			ResizeTarget: int(s),
		}

		iis = append(iis, ii)
	}

	return iis, nil
}
