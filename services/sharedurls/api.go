package sharedurls

import (
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/protocol/requests"
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

func (api *PublicAPI) ShareCommunityChannelURLWithChatKey(request *requests.CommunityChannelShareURL) (string, error) {
	return api.service.ShareCommunityChannelURLWithChatKey(request)
}

func (api *PublicAPI) ShareCommunityChannelURLWithData(request *requests.CommunityChannelShareURL) (string, error) {
	return api.service.ShareCommunityChannelURLWithData(request)
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
