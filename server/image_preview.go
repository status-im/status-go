package server

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"

	"go.uber.org/zap"

	"github.com/nfnt/resize"
	"github.com/status-im/status-go/images"
)

type ImagePreviewParams struct {
	imageUrl string
	size     int
}

var RequiredParams = []string{"image_url", "size"}

func parseImagePreviewParams(params url.Values) (*ImagePreviewParams, error) {
	urlParam := params.Get(RequiredParams[0])
	sizeParam := params.Get(RequiredParams[1])

	if len(urlParam) == 0 || len(sizeParam) == 0 {
		return nil, fmt.Errorf("wrong query params; required: %s", RequiredParams)
	}

	size, err := strconv.Atoi(sizeParam)

	if err != nil {
		return nil, fmt.Errorf("size param should be a valid number, got %s", sizeParam)
	}

	return &ImagePreviewParams{imageUrl: urlParam, size: size}, nil
}

func generateImagePreviewFromURL(params *ImagePreviewParams, logger *zap.Logger) (*bytes.Buffer, error) {
	imageUrl := params.imageUrl
	size := params.size

	image, err := images.DecodeFromURL(imageUrl)
	if err != nil {
		message := "couldn't decode image: " + err.Error()
		logger.Error(message, zap.Error(err))
		return nil, fmt.Errorf(message)
	}

	// resize the image based on the width only if it's bigger than the chosen size
	if size < image.Bounds().Dx() {
		image = resize.Resize(uint(size), 0, image, resize.Bilinear)
	}

	bb := bytes.NewBuffer([]byte{})
	err = images.EncodeToBestSize(bb, image, images.LargeDim)

	if err != nil {
		message := "couldn't compress image: " + err.Error()
		logger.Error(message, zap.Error(err))
		return nil, fmt.Errorf(message)
	}

	return bb, nil
}
