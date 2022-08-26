package discord

import (
	"encoding/base64"
	"fmt"

	"github.com/status-im/status-go/images"
	protocolImages "github.com/status-im/status-go/protocol/images"
	"github.com/status-im/status-go/protocol/protobuf"
)

func DownloadAndEncodeAvatarAsset(url string) ([]byte, *protobuf.ImageType, string, error) {
	imgs, err := images.GenerateIdentityImagesFromURL(url)
	if err != nil {
		return nil, nil, "", err
	}

	payload := imgs[0].Payload
	imageType := protocolImages.ImageType(payload)
	e64 := base64.StdEncoding

	maxEncLen := e64.EncodedLen(len(payload))
	encBuf := make([]byte, maxEncLen)
	e64.Encode(encBuf, payload)

	mime, err := images.GetMimeType(payload)
	if err != nil {
		return nil, nil, "", err
	}
	imageBase64 := fmt.Sprintf("data:image/%s;base64,%s", mime, encBuf)
	return payload, &imageType, imageBase64, nil
}
