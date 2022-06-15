package stickers

import (
	"encoding/json"

	"github.com/status-im/status-go/common/stickers"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/services/wallet/bigint"
)

const maxNumberRecentStickers = 24

func (api *API) recentStickers() ([]stickers.Sticker, error) {
	recentStickersList := make([]stickers.Sticker, 0)

	recentStickersJSON, err := api.accountsDB.GetRecentStickers()
	if err != nil {
		return recentStickersList, err
	}

	if recentStickersJSON == nil {
		return recentStickersList, nil
	}

	err = json.Unmarshal(*recentStickersJSON, &recentStickersList)
	if err != nil {
		return recentStickersList, err
	}

	return recentStickersList, nil
}

func (api *API) ClearRecent() error {
	var recentStickersList []stickers.Sticker
	return api.accountsDB.SaveSettingField(settings.StickersRecentStickers, recentStickersList)
}

func (api *API) Recent() ([]stickers.Sticker, error) {
	recentStickersList, err := api.recentStickers()
	if err != nil {
		return nil, err
	}

	for i, sticker := range recentStickersList {
		sticker.URL = api.hashToURL(sticker.Hash)
		recentStickersList[i] = sticker
	}

	return recentStickersList, nil
}

func (api *API) AddRecent(packID *bigint.BigInt, hash string) error {
	sticker := stickers.Sticker{
		PackID: packID,
		Hash:   hash,
	}

	recentStickersList, err := api.recentStickers()
	if err != nil {
		return err
	}

	// Remove duplicated
	idx := -1
	for i, currSticker := range recentStickersList {
		if currSticker.PackID.Cmp(sticker.PackID.Int) == 0 && currSticker.Hash == sticker.Hash {
			idx = i
		}
	}
	if idx > -1 {
		recentStickersList = append(recentStickersList[:idx], recentStickersList[idx+1:]...)
	}

	sticker.URL = ""

	if len(recentStickersList) >= maxNumberRecentStickers {
		recentStickersList = append([]stickers.Sticker{sticker}, recentStickersList[:maxNumberRecentStickers-1]...)
	} else {
		recentStickersList = append([]stickers.Sticker{sticker}, recentStickersList...)
	}

	return api.accountsDB.SaveSettingField(settings.StickersRecentStickers, recentStickersList)
}
