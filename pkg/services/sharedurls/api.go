package sharedurls

import (
	"github.com/status-im/status-go/internal/crypto/types"
)

type PublicAPI struct {
	service *Service
}

func (api *PublicAPI) ShareCommunityURLWithChatKey(communityID types.HexBytes) (string, error) {
	return api.service.ShareCommunityURLWithChatKey(communityID)
}

func (api *PublicAPI) ShareCommunityURLWithData(communityID types.HexBytes) (string, error) {
	return api.service.ShareCommunityURLWithData(communityID)
}

func (api *PublicAPI) ShareCommunityChannelURLWithChatKey(communityID types.HexBytes, channelID string) (string, error) {
	return api.service.ShareCommunityChannelURLWithChatKey(communityID, channelID)
}

func (api *PublicAPI) ShareCommunityChannelURLWithData(communityID types.HexBytes, channelID string) (string, error) {
	return api.service.ShareCommunityChannelURLWithData(communityID, channelID)
}

func (api *PublicAPI) ShareUserURLWithENS(pubKey string) (string, error) {
	return api.service.ShareUserURLWithENS(pubKey)
}

func (api *PublicAPI) ShareUserURLWithChatKey(pubKey string) (string, error) {
	return api.service.ShareUserURLWithChatKey(pubKey)
}

func (api *PublicAPI) ShareUserURLWithData(pubKey string) (string, error) {
	return api.service.ShareUserURLWithData(pubKey)
}

func (api *PublicAPI) ParseSharedURL(url string) (*URLDataResponse, error) {
	return ParseSharedURL(url)
}

func (api *PublicAPI) ShareMessageURL(chatID string, messageID string) (string, error) {
	return ShareMessageURL(chatID, messageID)
}

func (api *PublicAPI) ParseMessageURL(url string) (*MessageURLData, error) {
	return ParseMessageURL(url)
}
