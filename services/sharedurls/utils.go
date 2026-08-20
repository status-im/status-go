package sharedurls

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/andybalholm/brotli"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/protocol/protobuf"
)

const statusInternalBaseShareURL = "status-app://"
const messagePath = "m/"
const sharedURLMessagePrefix = baseShareURL + "/" + messagePath
const sharedURLMessagePrefixHTTP = "http://status.app/" + messagePath
const sharedURLMessagePrefixInternal = statusInternalBaseShareURL + messagePath
const messageIDQueryParam = "message-id"

func ShareMessageURL(chatID string, messageID string) (string, error) {
	if strings.TrimSpace(chatID) == "" {
		return "", fmt.Errorf("chatID is required")
	}

	if strings.TrimSpace(messageID) == "" {
		return "", fmt.Errorf("messageID is required")
	}

	return fmt.Sprintf("%s%s?%s=%s",
		sharedURLMessagePrefix,
		url.PathEscape(chatID),
		messageIDQueryParam,
		url.QueryEscape(messageID),
	), nil
}

func decodeMessageChatID(chatIDPathPart string) (string, error) {
	chatID, unescapeErr := url.PathUnescape(chatIDPathPart)
	if unescapeErr != nil {
		return "", unescapeErr
	}

	if chatID == "" {
		return "", fmt.Errorf("chatID is required")
	}

	return chatID, nil
}

func extractDeepLinkValueUntilSeparator(value string) string {
	if value == "" {
		return ""
	}

	endIdx := len(value)
	for _, separator := range []string{"?", "#", "&"} {
		idx := strings.Index(value, separator)
		if idx != -1 && idx < endIdx {
			endIdx = idx
		}
	}

	return value[:endIdx]
}

func extractQueryParamFromLink(link string, paramName string) (string, error) {
	parsedLink, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	queryValues, err := url.ParseQuery(parsedLink.RawQuery)
	if err != nil {
		return "", err
	}

	return queryValues.Get(paramName), nil
}

func deriveCommunityIDFromChatID(chatID string) string {
	if len(chatID) <= 36 {
		return ""
	}

	channelID := chatID[len(chatID)-36:]
	if !channelRegExp.MatchString(channelID) {
		return ""
	}

	return chatID[:len(chatID)-36]
}

func ParseMessageURL(rawURL string) (*MessageURLData, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return nil, fmt.Errorf("not a status message url")
	}

	pathRemainder := ""
	switch {
	case strings.HasPrefix(trimmedURL, sharedURLMessagePrefix):
		pathRemainder = strings.TrimPrefix(trimmedURL, sharedURLMessagePrefix)
	case strings.HasPrefix(trimmedURL, sharedURLMessagePrefixHTTP):
		pathRemainder = strings.TrimPrefix(trimmedURL, sharedURLMessagePrefixHTTP)
	case strings.HasPrefix(trimmedURL, sharedURLMessagePrefixInternal):
		pathRemainder = strings.TrimPrefix(trimmedURL, sharedURLMessagePrefixInternal)
	default:
		return nil, fmt.Errorf("not a status message url")
	}

	chatIDPathPart := extractDeepLinkValueUntilSeparator(pathRemainder)
	if chatIDPathPart == "" {
		return nil, fmt.Errorf("chatID is required")
	}

	chatID, err := decodeMessageChatID(chatIDPathPart)
	if err != nil {
		return nil, err
	}

	messageID, err := extractQueryParamFromLink(trimmedURL, messageIDQueryParam)
	if err != nil {
		return nil, err
	}

	if messageID == "" {
		return nil, fmt.Errorf("messageID is required")
	}

	return &MessageURLData{
		ChatID:      chatID,
		MessageID:   messageID,
		CommunityID: deriveCommunityIDFromChatID(chatID),
	}, nil
}

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
		strings.HasPrefix(url, sharedURLMessagePrefix) ||
		strings.HasPrefix(url, sharedURLChannelPrefixWithData) ||
		strings.HasPrefix(url, sharedURLMessagePrefixHTTP) ||
		strings.HasPrefix(url, sharedURLMessagePrefixInternal)
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
