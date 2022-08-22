package images

import (
	"encoding/json"
	"errors"

	"github.com/status-im/status-go/eth-node/crypto"
)

type IdentityImage struct {
	KeyUID       string `json:"key_uid"`
	Name         string `json:"name"`
	Payload      []byte `json:"payload"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size"`
	ResizeTarget int    `json:"resize_target"`
	Clock        uint64 `json:"clock"`
}

func (i IdentityImage) GetType() (ImageType, error) {
	it := GetType(i.Payload)
	if it == UNKNOWN {
		return it, errors.New("unsupported file type")
	}

	return it, nil
}

func (i IdentityImage) Hash() []byte {
	return crypto.Keccak256(i.Payload)
}

func (i IdentityImage) GetDataURI() (string, error) {
	return GetPayloadDataURI(i.Payload)
}

func (i IdentityImage) MarshalJSON() ([]byte, error) {
	uri, err := i.GetDataURI()
	if err != nil {
		return nil, err
	}

	temp := struct {
		KeyUID       string `json:"keyUid"`
		Name         string `json:"type"`
		URI          string `json:"uri"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		FileSize     int    `json:"fileSize"`
		ResizeTarget int    `json:"resizeTarget"`
		Clock        uint64 `json:"clock"`
	}{
		KeyUID:       i.KeyUID,
		Name:         i.Name,
		URI:          uri,
		Width:        i.Width,
		Height:       i.Height,
		FileSize:     i.FileSize,
		ResizeTarget: i.ResizeTarget,
		Clock:        i.Clock,
	}

	return json.Marshal(temp)
}
