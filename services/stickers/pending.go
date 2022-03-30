package stickers

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/zenthangplus/goccm"
)

func (api *API) AddPending(chainID uint64, packID *bigint.BigInt) error {
	pendingPacks, err := api.pendingStickerPacks()
	if err != nil {
		return err
	}

	if _, exists := pendingPacks[uint(packID.Uint64())]; exists {
		return errors.New("sticker pack is already pending")
	}

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return err
	}

	stickerPack, err := api.fetchPackData(stickerType, packID.Int, false)
	if err != nil {
		return err
	}

	pendingPacks[uint(packID.Uint64())] = *stickerPack

	return api.accountsDB.SaveSettingField(settings.StickersPacksPending, pendingPacks)
}

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

func (api *API) ProcessPending(chainID uint64) (pendingChanged StickerPackCollection, err error) {
	pendingStickerPacks, err := api.pendingStickerPacks()
	if err != nil {
		return nil, err
	}

	stickerType, err := api.contractMaker.NewStickerType(chainID)
	if err != nil {
		return nil, err
	}

	stickerChan := make(chan StickerPack, 10)
	go func() {
		c := goccm.New(maxConcurrentRequests)
		for _, pendingStickerPack := range pendingStickerPacks {
			c.Wait()
			go func(pendingStickerPack StickerPack) {
				defer c.Done()
				stickerPack, err := api.fetchPackData(stickerType, pendingStickerPack.ID.Int, true)
				if err != nil {
					log.Warn("Could not retrieve stickerpack data", "packID", pendingStickerPack.ID.Int, "error", err)
					return
				}

				if stickerPack.Status == statusPurchased {
					stickerChan <- *stickerPack
				}
			}(pendingStickerPack)
		}

		c.WaitAllDone()
		close(stickerChan)
	}()

	result := make(StickerPackCollection)
	for stickerPack := range stickerChan {
		packID := uint(stickerPack.ID.Uint64())
		if _, exists := pendingStickerPacks[packID]; !exists {
			continue
		}
		delete(pendingStickerPacks, packID)
		result[packID] = stickerPack
	}

	err = api.accountsDB.SaveSettingField(settings.StickersPacksPending, pendingStickerPacks)
	return result, err
}

func (api *API) RemovePending(packID *bigint.BigInt) error {
	pendingPacks, err := api.pendingStickerPacks()
	if err != nil {
		return err
	}

	if _, exists := pendingPacks[uint(packID.Uint64())]; !exists {
		return nil
	}

	delete(pendingPacks, uint(packID.Uint64()))

	return api.accountsDB.SaveSettingField(settings.StickersPacksPending, pendingPacks)
}
