package sharedurls

import (
	messagingtypes "github.com/status-im/status-go/messaging/types"
)

type CommunityURLData struct {
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description"`
	MembersCount uint32   `json:"membersCount"`
	Color        string   `json:"color"`
	TagIndices   []uint32 `json:"tagIndices"`
	CommunityID  string   `json:"communityId"`
}

type CommunityChannelURLData struct {
	Emoji       string `json:"emoji"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Color       string `json:"color"`
	ChannelUUID string `json:"channelUuid"`
}

type ContactURLData struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	PublicKey   string `json:"publicKey"`
}

type URLDataResponse struct {
	Community *CommunityURLData        `json:"community"`
	Channel   *CommunityChannelURLData `json:"channel"`
	Contact   *ContactURLData          `json:"contact"`
	Shard     *messagingtypes.Shard    `json:"shard,omitempty"`
}
