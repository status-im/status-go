package server

type MediaServerInterface interface {
	MakeCommunityDescriptionTokenImageURL(communityID, symbol string) string
	MakeCommunityImageURL(communityID, name string) string
}
