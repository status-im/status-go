package images

import (
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/protobuf"
)

func ImageType(buf []byte) protobuf.ImageType {
	switch images.GetFileType(buf){
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
