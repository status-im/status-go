package images

import (
	"errors"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/protobuf"
)

func ImageType(buf []byte) protobuf.ImageType {
	switch images.GetType(buf) {
	case images.JPEG:
		return protobuf.ImageType_JPEG
	case images.PNG:
		return protobuf.ImageType_PNG
	case images.GIF:
		return protobuf.ImageType_GIF
	case images.WEBP:
		return protobuf.ImageType_WEBP
	default:
		return protobuf.ImageType_UNKNOWN_IMAGE_TYPE
	}
}

func ImageMime(buf []byte) (string, error) {
	switch images.GetType(buf) {
	case images.JPEG:
		return "image/jpeg", nil
	case images.PNG:
		return "image/png", nil
	case images.GIF:
		return "image/gif", nil
	case images.WEBP:
		return "image/webp", nil
	default:
		return "", errors.New("mime type not found")
	}
}
