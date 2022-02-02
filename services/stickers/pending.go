package stickers

import (
	"encoding/json"
	"errors"
	"math/big"
)

func (api *API) AddPending(chainID uint64, packID uint64) error {
	pendingPacks, err := api.pendingStickerPacks()
	if err != nil {
		return err
	}

	if _, exists := pendingPacks[uint(packID)]; exists {
		return errors.New("sticker pack is already pending")
	}

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return err
	}

	stickerPack, err := api.fetchPackData(stickerType, new(big.Int).SetUint64(packID), false)
	if err != nil {
		return err
	}

	pendingPacks[uint(packID)] = *stickerPack

	return api.accountsDB.SaveSetting("stickers/packs-pending", pendingPacks)
}

func (api *API) pendingStickerPacks() (map[uint]StickerPack, error) {
	stickerPacks := make(map[uint]StickerPack)

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

func (api *API) Pending() (map[uint]StickerPack, error) {
	stickerPacks, err := api.pendingStickerPacks()
	if err != nil {
		return nil, err
	}

	for packID, stickerPack := range stickerPacks {
		stickerPack.Status = statusPending
		stickerPacks[packID] = stickerPack
	}

	return stickerPacks, nil
}

func (api *API) RemovePending(packID uint64) error {
	pendingPacks, err := api.pendingStickerPacks()
	if err != nil {
		return err
	}

	if _, exists := pendingPacks[uint(packID)]; !exists {
		return errors.New("sticker pack is not pending")
	}

	delete(pendingPacks, uint(packID))

	return api.accountsDB.SaveSetting("stickers/packs-pending", pendingPacks)
}
