package identity

import "reflect"

type VisibleProfileShowcaseEntry struct {
	EntryID string `json:"entryId"`
	Order   int    `json:"order"`
}

type ProfileShowcase struct {
	Communities  []*VisibleProfileShowcaseEntry `json:"communities"`
	Accounts     []*VisibleProfileShowcaseEntry `json:"accounts"`
	Collectibles []*VisibleProfileShowcaseEntry `json:"collectibles"`
	Assets       []*VisibleProfileShowcaseEntry `json:"assets"`
}

func (p1 ProfileShowcase) Equal(p2 ProfileShowcase) bool {
	return reflect.DeepEqual(p1, p2)
}
