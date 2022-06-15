package stickers

import (
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type Sticker struct {
	PackID *bigint.BigInt `json:"packID,omitempty"`
	URL    string         `json:"url,omitempty"`
	Hash   string         `json:"hash,omitempty"`
}

type StickerPack struct {
	ID        *bigint.BigInt    `json:"id"`
	Name      string            `json:"name"`
	Author    string            `json:"author"`
	Owner     ethCommon.Address `json:"owner,omitempty"`
	Price     *bigint.BigInt    `json:"price"`
	Preview   string            `json:"preview"`
	Thumbnail string            `json:"thumbnail"`
	Stickers  []Sticker         `json:"stickers"`

	Status StickerStatus `json:"status"`
}

type StickerPackCollection map[uint]StickerPack

func (spc StickerPackCollection) Merge(sp StickerPackCollection) {
	for _, s := range sp {
		if _, exists := spc[uint(s.ID.Uint64())]; !exists {
			spc[uint(s.ID.Uint64())] = s
		}
	}
}
