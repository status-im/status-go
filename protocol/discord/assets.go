package discord

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"strings"

	"github.com/ethereum/go-ethereum/log"
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

func DownloadAndEncodeImage(url string) ([]byte, string, string, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Get(url)
	if err != nil {
		return nil, "", "", err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Error("failed to close profile pic http request body", "err", err)
		}
	}()

	contentType := res.Header.Get("Content-Type")
  fmt.Println("CONTENT TYPE: ", contentType)

  // Could simply switch/case over `contentType` here but sometimes it
  // includes additional data, so check for prefix instead
  if !strings.HasPrefix(contentType, "image/png") &&
    !strings.HasPrefix(contentType, "image/jpeg") &&
    !strings.HasPrefix(contentType, "image/jpg") &&
    !strings.HasPrefix(contentType, "image/webp") {
    fmt.Println("ERROR: ", strings.HasPrefix(contentType, "image/png"))
	  return nil, "", "", errors.New("Unsupported asset type")
  }

	bodyBytes := make([]byte, 0)
  bodyBytes, err = ioutil.ReadAll(res.Body)
  // img, err := images.DecodeFromBytes(bodyBytes)
  // if err != nil {
  //   return nil, err
  // }

	// imageType := protocolImages.ImageType(bodyBytes)
	e64 := base64.StdEncoding

	maxEncLen := e64.EncodedLen(len(bodyBytes))
	encBuf := make([]byte, maxEncLen)
	e64.Encode(encBuf, bodyBytes)
	mime, err := images.GetMimeType(bodyBytes)
	if err != nil {
    return nil, "", "", err
	}
	imageBase64 := fmt.Sprintf("data:image/%s;base64,%s", mime, encBuf)
  return bodyBytes, contentType, imageBase64, nil
}
