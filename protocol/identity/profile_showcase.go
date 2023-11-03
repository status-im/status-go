package identity

import "reflect"

type ProfileShowcaseCommunity struct {
	CommunityID string `json:"communityId"`
	Order       int    `json:"order"`
}

type ProfileShowcaseAccount struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	ColorID string `json:"colorId"`
	Emoji   string `json:"emoji"`
	Order   int    `json:"order"`
}

type ProfileShowcaseCollectible struct {
	UID   string `json:"uid"`
	Order int    `json:"order"`
}

type ProfileShowcaseAsset struct {
	Symbol string `json:"symbol"`
	Order  int    `json:"order"`
}

type ProfileShowcase struct {
	Communities  []*ProfileShowcaseCommunity   `json:"communities"`
	Accounts     []*ProfileShowcaseAccount     `json:"accounts"`
	Collectibles []*ProfileShowcaseCollectible `json:"collectibles"`
	Assets       []*ProfileShowcaseAsset       `json:"assets"`
}

func (p1 ProfileShowcase) Equal(p2 ProfileShowcase) bool {
	return reflect.DeepEqual(p1, p2)
}
