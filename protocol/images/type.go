package images

import (
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/protobuf"
)

func ImageType(buf []byte) protobuf.ImageMessage_ImageType {
	switch images.GetFileType(buf){
	case images.JPEG:
		return protobuf.ImageMessage_JPEG
	case images.PNG:
		return protobuf.ImageMessage_PNG
	case images.GIF:
		return protobuf.ImageMessage_GIF
	case images.WEBP:
		return protobuf.ImageMessage_WEBP
	default:
		return protobuf.ImageMessage_UNKNOWN_IMAGE_TYPE
	}
}
