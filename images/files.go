package images

import (
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"

	"golang.org/x/image/webp"
)

func Get(fileName string) (image.Image, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fb := make([]byte, 12)
	file.Read(fb)
	ft := GetFileType(fb)
	if ft == UNKNOWN {
		return nil, errors.New("unsupported file type")
	}

	var img image.Image
	switch ft {
	case JPEG:
		img, err = jpeg.Decode(file)
		break
	case PNG:
		img, err = png.Decode(file)
		break
	case GIF:
		img, err = gif.Decode(file)
		break
	case WEBP:
		img, err = webp.Decode(file)
		break
	}
	if err != nil {
		return nil, err
	}

	return img, nil
}

func GetFileType(buf []byte) FileType {
	switch {
	case isJpeg(buf):
		return JPEG
	case isPng(buf):
		return PNG
	case isGif(buf):
		return GIF
	case isWebp(buf):
		return WEBP
	default:
		return UNKNOWN
	}
}

func isJpeg(buf []byte) bool {
	return len(buf) > 2 &&
		buf[0] == 0xFF &&
		buf[1] == 0xD8 &&
		buf[2] == 0xFF
}

func isPng(buf []byte) bool {
	return len(buf) > 3 &&
		buf[0] == 0x89 && buf[1] == 0x50 &&
		buf[2] == 0x4E && buf[3] == 0x47
}

func isGif(buf []byte) bool {
	return len(buf) > 2 &&
		buf[0] == 0x47 && buf[1] == 0x49 && buf[2] == 0x46
}

func isWebp(buf []byte) bool {
	return len(buf) > 11 &&
		buf[8] == 0x57 && buf[9] == 0x45 &&
		buf[10] == 0x42 && buf[11] == 0x50
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

func renderJpeg(w io.Writer, m image.Image, imgDetail *Details) error {
	o := new(jpeg.Options)
	o.Quality = imgDetail.Quality

	return jpeg.Encode(w, m, o)
}
