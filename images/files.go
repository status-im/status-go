package images

import (
	"image"
	"image/jpeg"
	"io"
	"os"
)

func Get(fileName string) (image.Image, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	switch getFileType(file) {
	case JPEG:
		err = decodeJpeg(file)
		break
	case PNG:
		err = decodePng(file)
		break
	case WEBP:
		err = decodeWebp(file)
		break
	}

	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func Render(img image.Image, imgDetail *Details) error {
	out, err := os.Create(imgDetail.FileName)
	if err != nil {
		return err
	}
	defer out.Close()

	err = renderJpeg(out, img, imgDetail)
	if err != nil {
		return err
	}

	fi, _ := out.Stat()
	imgDetail.SizeFile = fi.Size()

	return nil
}

func getFileType(file *os.File) FileType {
	// TODO
	return JPEG
}

func decodeJpeg(file *os.File) error {
	// TODO
	return nil
}

func decodePng(file *os.File) error {
	// TODO
	return nil
}

func decodeWebp(file *os.File) error {
	// TODO
	return nil
}

func renderJpeg(w io.Writer, m image.Image, imgDetail *Details) error {
	o := new(jpeg.Options)
	o.Quality = imgDetail.Quality

	return jpeg.Encode(w, m, o)
}
