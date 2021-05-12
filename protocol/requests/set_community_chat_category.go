package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrSetCommunityChatCategoryInvalidCommunityID = errors.New("set-community-chat-category: invalid community id")
var ErrSetCommunityChatCategoryInvalidChatID = errors.New("set-community-chat-category: invalid category id")

type SetCommunityChatCategory struct {
	CommunityID types.HexBytes `json:"communityId"`
	ChatID      string         `json:"chatId"`
	CategoryID  string         `json:"categoryId"`
	Position    uint           `json:"position"`
}

func (j *SetCommunityChatCategory) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrSetCommunityChatCategoryInvalidCommunityID
	}

	if len(j.ChatID) == 0 {
		return ErrSetCommunityChatCategoryInvalidChatID

	}

	return nil
}
