package stickers

import (
	"encoding/json"
)

const maxNumberRecentStickers = 24

func (api *API) recentStickers() ([]Sticker, error) {
	recentStickersList := make([]Sticker, 0)

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
	recentStickersList := []Sticker{}
	return api.accountsDB.SaveSetting("stickers/recent-stickers", recentStickersList)
}

func (api *API) Recent() ([]Sticker, error) {
	recentStickersList, err := api.recentStickers()
	if err != nil {
		return nil, err
	}

	for i, sticker := range recentStickersList {
		sticker.URL, err = decodeStringHash(sticker.Hash)
		if err != nil {
			return nil, err
		}
		recentStickersList[i] = sticker
	}

	return recentStickersList, nil
}

func (api *API) AddRecent(sticker Sticker) error {
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
		recentStickersList = append([]Sticker{sticker}, recentStickersList[:maxNumberRecentStickers-1]...)
	} else {
		recentStickersList = append([]Sticker{sticker}, recentStickersList...)
	}

	return api.accountsDB.SaveSetting("stickers/recent-stickers", recentStickersList)
}
