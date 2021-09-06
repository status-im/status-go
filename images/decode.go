package images

import (
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"ioutil"
	"net/http"
	"os"

	"golang.org/x/image/webp"
)

func Decode(fileName string) (image.Image, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fb, err := prepareFileForDecode(file)
	if err != nil {
		return nil, err
	}

	return decodeImageData(fb, file)
}

func DecodeFromURL(path string) (image.Image, error) {
	res, err := http.Get(path)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return decodeImageData(bodyBytes, res.Body)
}

func prepareFileForDecode(file *os.File) ([]byte, error) {
	// Read the first 14 bytes, used for performing image type checks before parsing the image data
	fb := make([]byte, 14)
	_, err := file.Read(fb)
	if err != nil {
		return nil, err
	}

	// Reset the read cursor
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	return fb, nil
}

func decodeImageData(buf []byte, r io.Reader) (img image.Image, err error) {
	switch GetType(buf) {
	case JPEG:
		img, err = jpeg.Decode(r)
	case PNG:
		img, err = png.Decode(r)
	case GIF:
		img, err = gif.Decode(r)
	case WEBP:
		img, err = webp.Decode(r)
	case UNKNOWN:
		fallthrough
	default:
		return nil, errors.New("unsupported file type")
	}
	if err != nil {
		return nil, err
	}

	return img, nil
}

func GetType(buf []byte) ImageType {
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

func GetMimeType(buf []byte) (string, error) {
	switch {
	case isJpeg(buf):
		return "jpeg", nil
	case isPng(buf):
		return "png", nil
	case isGif(buf):
		return "gif", nil
	case isWebp(buf):
		return "webp", nil
	default:
		return "", errors.New("image format not supported")
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
