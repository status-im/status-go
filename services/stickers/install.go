package stickers

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/status-im/status-go/services/wallet/bigint"
)

func (api *API) Install(chainID uint64, packID uint64) error {
	installedPacks, err := api.installedStickerPacks()
	if err != nil {
		return err
	}

	if _, exists := installedPacks[uint(packID)]; exists {
		return errors.New("sticker pack is already installed")
	}

	// TODO: this does not validate if the pack is purchased. Should it?

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return err
	}

	stickerPack, err := api.fetchPackData(stickerType, new(big.Int).SetUint64(packID), false)
	if err != nil {
		return err
	}

	installedPacks[uint(packID)] = *stickerPack

	err = api.accountsDB.SaveSetting("stickers/packs-installed", installedPacks)
	if err != nil {
		return err
	}

	return nil
}

func (api *API) installedStickerPacks() (map[uint]StickerPack, error) {
	stickerPacks := make(map[uint]StickerPack)

	installedStickersJSON, err := api.accountsDB.GetInstalledStickerPacks()
	if err != nil {
		return nil, err
	}

	if installedStickersJSON == nil {
		return stickerPacks, nil
	}

	err = json.Unmarshal(*installedStickersJSON, &stickerPacks)
	if err != nil {
		return nil, err
	}

	return stickerPacks, nil
}

func (api *API) Installed() (map[uint]StickerPack, error) {
	stickerPacks, err := api.installedStickerPacks()
	if err != nil {
		return nil, err
	}

	for packID, stickerPack := range stickerPacks {
		stickerPack.Status = statusInstalled

		stickerPack.Preview, err = decodeStringHash(stickerPack.Preview)
		if err != nil {
			return nil, err
		}

		stickerPack.Thumbnail, err = decodeStringHash(stickerPack.Thumbnail)
		if err != nil {
			return nil, err
		}

		for i, sticker := range stickerPack.Stickers {
			sticker.URL, err = decodeStringHash(sticker.Hash)
			if err != nil {
				return nil, err
			}
			stickerPack.Stickers[i] = sticker
		}

		stickerPacks[packID] = stickerPack
	}

	return stickerPacks, nil
}

func (api *API) Uninstall(packID uint64) error {
	installedPacks, err := api.installedStickerPacks()
	if err != nil {
		return err
	}

	if _, exists := installedPacks[uint(packID)]; !exists {
		return errors.New("sticker pack is not installed")
	}

	delete(installedPacks, uint(packID))

	err = api.accountsDB.SaveSetting("stickers/packs-installed", installedPacks)
	if err != nil {
		return err
	}

	// Removing uninstalled pack from recent stickers

	recentStickers, err := api.recentStickers()
	if err != nil {
		return err
	}

	pID := &bigint.BigInt{Int: new(big.Int).SetUint64(packID)}
	idx := -1
	for i, r := range recentStickers {
		if r.PackID.Cmp(pID.Int) == 0 {
			idx = i
			break
		}
	}

	if idx > -1 {
		var newRecentStickers []Sticker
		newRecentStickers = append(newRecentStickers, recentStickers[:idx]...)
		if idx != len(recentStickers)-1 {
			newRecentStickers = append(newRecentStickers, recentStickers[idx+1:]...)
		}
		return api.accountsDB.SaveSetting("stickers/recent-stickers", newRecentStickers)
	}

	return nil
}
