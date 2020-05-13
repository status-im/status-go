package images

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func jpeg(buf []byte) bool {
	return len(buf) > 2 &&
		buf[0] == 0xFF &&
		buf[1] == 0xD8 &&
		buf[2] == 0xFF
}

func png(buf []byte) bool {
	return len(buf) > 3 &&
		buf[0] == 0x89 && buf[1] == 0x50 &&
		buf[2] == 0x4E && buf[3] == 0x47
}

func gif(buf []byte) bool {
	return len(buf) > 2 &&
		buf[0] == 0x47 && buf[1] == 0x49 && buf[2] == 0x46
}

func webp(buf []byte) bool {
	return len(buf) > 11 &&
		buf[8] == 0x57 && buf[9] == 0x45 &&
		buf[10] == 0x42 && buf[11] == 0x50
}

func ImageType(buf []byte) protobuf.ImageMessage_ImageType {
	if jpeg(buf) {
		return protobuf.ImageMessage_JPEG
	} else if png(buf) {
		return protobuf.ImageMessage_PNG
	} else if gif(buf) {
		return protobuf.ImageMessage_GIF
	} else if webp(buf) {
		return protobuf.ImageMessage_WEBP
	}

	return protobuf.ImageMessage_UNKNOWN_IMAGE_TYPE
}
