package server

import "github.com/status-im/status-go/protocol/protobuf"

type MediaServerInterface interface {
	MakeCommunityDescriptionTokenImageURL(communityID, symbol string) string
	MakeCommunityImageURL(communityID, name string) string
	SetCommunityImageVersionReader(func(communityID string) uint32)
	SetCommunityImageReader(func(communityID string) (map[string]*protobuf.IdentityImage, error))
	SetCommunityTokensReader(func(communityID string) ([]*protobuf.CommunityTokenMetadata, error))
}
