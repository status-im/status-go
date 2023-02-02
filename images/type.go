package images

import (
	"errors"

	"github.com/status-im/status-go/protocol/protobuf"
)

func GetProtobufImageType(buf []byte) protobuf.ImageType {
	switch GetType(buf) {
	case JPEG:
		return protobuf.ImageType_JPEG
	case PNG:
		return protobuf.ImageType_PNG
	case GIF:
		return protobuf.ImageType_GIF
	case WEBP:
		return protobuf.ImageType_WEBP
	default:
		return protobuf.ImageType_UNKNOWN_IMAGE_TYPE
	}
}

func GetProtobufImageMime(buf []byte) (string, error) {
	switch GetType(buf) {
	case JPEG:
		return "image/jpeg", nil
	case PNG:
		return "image/png", nil
	case GIF:
		return "image/gif", nil
	case WEBP:
		return "image/webp", nil
	default:
		return "", errors.New("mime type not found")
	}
}
