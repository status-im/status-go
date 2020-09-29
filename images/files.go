package images

import (
	"image"
	"image/jpeg"
	"os"
)

func Get(fileName string) (image.Image, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, err
	}
	file.Close()

	return img, nil
}

func Render(img image.Image, imgDetail *Details) error {
	out, err := os.Create(imgDetail.FileName)
	if err != nil {
		return err
	}
	defer out.Close()

	o := new(jpeg.Options)
	o.Quality = imgDetail.Quality

	jpeg.Encode(out, img, o)

	fi, _ := out.Stat()
	imgDetail.SizeFile = fi.Size()

	return nil
}
