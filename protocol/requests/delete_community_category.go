package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrDeleteCommunityCategoryInvalidCommunityID = errors.New("set-community-chat-category: invalid community id")
var ErrDeleteCommunityCategoryInvalidCategoryID = errors.New("set-community-chat-category: invalid category id")

type DeleteCommunityCategory struct {
	CommunityID types.HexBytes `json:"communityId"`
	CategoryID  string         `json:"categoryId"`
}

func (j *DeleteCommunityCategory) Validate() error {
	if len(j.CommunityID) == 0 {
		return ErrDeleteCommunityCategoryInvalidCommunityID
	}

	if len(j.CategoryID) == 0 {
		return ErrDeleteCommunityCategoryInvalidCategoryID

	}

	return nil
}
