package images

import (
	"bytes"
	"image"
)

func GenerateIdentityImages(filepath string, aX, aY, bX, bY int) ([]*IdentityImage, error) {
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

	iis := make([]*IdentityImage, len(ResizeDimensions))
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
