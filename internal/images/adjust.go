package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	maxChatMessageImageSize = 400000
	resizeTargetImageSize   = 350000
	idealTargetImageSize    = 50000
)

var DefaultBounds = FileSizeLimits{Ideal: idealTargetImageSize, Max: resizeTargetImageSize}

func FetchAndStoreRemoteImage(url string) (string, error) {
	resp, err := http.Get(url) //nolint
	if err != nil {
		return "", fmt.Errorf("error fetching image from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %s", resp.Status)
	}

	tempFile, err := ioutil.TempFile("", "image-*")
	if err != nil {
		return "", fmt.Errorf("error creating a temporary file: %w", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name()) // Ensure temporary file is deleted on error
		return "", fmt.Errorf("error writing image to temp file: %w", err)
	}

	return tempFile.Name(), nil
}

func OpenAndDecodeImage(imagePath string) (image.Image, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("error opening image file: %w", err)
	}
	defer file.Close()

	img, err := Decode(imagePath)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %w", err)
	}

	return img, nil
}

func AdjustImage(img image.Image, crop bool, inputImage CroppedImage) ([]byte, error) {
	if crop {
		cropRect := image.Rectangle{
			Min: image.Point{X: inputImage.X, Y: inputImage.Y},
			Max: image.Point{X: inputImage.X + inputImage.Width, Y: inputImage.Y + inputImage.Height},
		}
		var err error
		img, err = Crop(img, cropRect)
		if err != nil {
			return nil, fmt.Errorf("error cropping image: %w", err)
		}
	}

	bb := bytes.NewBuffer([]byte{})
	err := CompressToFileLimits(bb, img, DefaultBounds)
	if err != nil {
		return nil, fmt.Errorf("error compressing image: %w", err)
	}

	payload := bb.Bytes()
	if len(payload) > maxChatMessageImageSize {
		return nil, errors.New("image too large")
	}

	return payload, nil
}
func OpenAndAdjustImage(inputImage CroppedImage, crop bool) ([]byte, error) {
	var imgPath string = inputImage.ImagePath
	var err error

	// Check if the image is from a remote source
	if strings.HasPrefix(inputImage.ImagePath, "http://") || strings.HasPrefix(inputImage.ImagePath, "https://") {
		imgPath, err = FetchAndStoreRemoteImage(inputImage.ImagePath)
		if err != nil {
			return nil, err
		}
		defer os.Remove(imgPath) // Clean up the temporary file
	}

	// Decode the image
	img, err := OpenAndDecodeImage(imgPath)
	if err != nil {
		return nil, err
	}

	// Adjust (crop and compress) the image
	return AdjustImage(img, crop, inputImage)
}
