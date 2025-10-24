package sharedurls

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/andybalholm/brotli"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

func parseUserURLWithData(data string, chatKey string) (*URLDataResponse, error) {
	if data == "" {
		return &URLDataResponse{
			Contact: &ContactURLData{
				DisplayName: "",
				Description: "",
				PublicKey:   chatKey,
			},
		}, nil
	}
	urlData, err := decodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var urlDataProto protobuf.URLData
	err = proto.Unmarshal(urlData, &urlDataProto)
	if err != nil {
		return nil, err
	}

	var userProto protobuf.User
	err = proto.Unmarshal(urlDataProto.Content, &userProto)
	if err != nil {
		return nil, err
	}

	return &URLDataResponse{
		Contact: &ContactURLData{
			DisplayName: userProto.DisplayName,
			Description: userProto.Description,
			PublicKey:   chatKey,
		},
	}, nil
}

func IsStatusSharedURL(url string) bool {
	return strings.HasPrefix(url, sharedURLUserPrefix) ||
		strings.HasPrefix(url, sharedURLUserPrefixWithData) ||
		strings.HasPrefix(url, sharedURLCommunityPrefix) ||
		strings.HasPrefix(url, sharedURLCommunityPrefixWithData) ||
		strings.HasPrefix(url, sharedURLChannelPrefixWithData)
}

func splitSharedURLData(data string) (string, string, error) {
	const count = 2
	contents := strings.SplitN(data, "#", count)
	if len(contents) != count {
		return "", "", fmt.Errorf("url should contain at least one `#` separator")
	}
	return contents[0], contents[1], nil
}

func ParseSharedURL(url string) (*URLDataResponse, error) {

	if strings.HasPrefix(url, sharedURLUserPrefix) {
		chatKey := strings.TrimPrefix(url, sharedURLUserPrefix)
		if strings.HasPrefix(chatKey, "zQ3sh") {
			return parseUserURLWithChatKey(chatKey)
		}
		return parseUserURLWithENS(chatKey)
	}

	if strings.HasPrefix(url, sharedURLUserPrefixWithData) {
		trimmedURL := strings.TrimPrefix(url, sharedURLUserPrefixWithData)
		encodedData, chatKey, err := splitSharedURLData(trimmedURL)
		if err != nil {
			return nil, err
		}
		return parseUserURLWithData(encodedData, chatKey)
	}

	if strings.HasPrefix(url, sharedURLCommunityPrefix) {
		chatKey := strings.TrimPrefix(url, sharedURLCommunityPrefix)
		return parseCommunityURLWithChatKey(chatKey)
	}

	if strings.HasPrefix(url, sharedURLCommunityPrefixWithData) {
		trimmedURL := strings.TrimPrefix(url, sharedURLCommunityPrefixWithData)
		encodedData, chatKey, err := splitSharedURLData(trimmedURL)
		if err != nil {
			return nil, err
		}
		return parseCommunityURLWithData(encodedData, chatKey)
	}

	if strings.HasPrefix(url, sharedURLChannelPrefixWithData) {
		trimmedURL := strings.TrimPrefix(url, sharedURLChannelPrefixWithData)
		encodedData, chatKey, err := splitSharedURLData(trimmedURL)
		if err != nil {
			return nil, err
		}

		if channelRegExp.MatchString(encodedData) {
			return parseCommunityChannelURLWithChatKey(encodedData, chatKey)
		}
		return parseCommunityChannelURLWithData(encodedData, chatKey)
	}

	return nil, fmt.Errorf("not a status shared url")
}

func encodeDataURL(data []byte) (string, error) {
	bb := bytes.NewBuffer([]byte{})
	writer := brotli.NewWriter(bb)
	_, err := writer.Write(data)
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bb.Bytes()), nil
}

func decodeDataURL(data string) ([]byte, error) {
	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	output := make([]byte, 4096)
	bb := bytes.NewBuffer(decoded)
	reader := brotli.NewReader(bb)
	n, err := reader.Read(output)
	if err != nil {
		return nil, err
	}

	return output[:n], nil
}
