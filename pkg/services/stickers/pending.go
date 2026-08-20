package stickers

import (
	"encoding/json"
)

func (api *API) pendingStickerPacks() (StickerPackCollection, error) {
	stickerPacks := make(StickerPackCollection)

	pendingStickersJSON, err := api.accountsDB.GetPendingStickerPacks()
	if err != nil {
		return nil, err
	}

	if pendingStickersJSON == nil {
		return stickerPacks, nil
	}

	err = json.Unmarshal(*pendingStickersJSON, &stickerPacks)
	if err != nil {
		return nil, err
	}

	return stickerPacks, nil
}

func (api *API) Pending() (StickerPackCollection, error) {
	stickerPacks, err := api.pendingStickerPacks()
	if err != nil {
		return nil, err
	}

	for packID, stickerPack := range stickerPacks {
		stickerPack.Status = statusPending
		stickerPack.Preview = api.hashToURL(stickerPack.Preview)
		stickerPack.Thumbnail = api.hashToURL(stickerPack.Thumbnail)
		for i, sticker := range stickerPack.Stickers {
			sticker.URL = api.hashToURL(sticker.Hash)
			stickerPack.Stickers[i] = sticker
		}
		stickerPacks[packID] = stickerPack
	}

	return stickerPacks, nil
}
