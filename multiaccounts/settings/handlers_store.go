package settings

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common/stickers"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/errors"
)

func OverwriteStoreHandler(db *Database, sf SettingField, value interface{}, clock uint64) error {
	err := db.saveSyncSetting(sf, value, clock)
	if err == errors.ErrNewClockOlderThanCurrent {
		logger := logutils.Logger()
		logger.Info("OverwriteStoreHandler - saveSyncSetting :", zap.Error(err))
		return nil
	}
	return err
}

func StickerPacksStoreHandler(db *Database, sf SettingField, value interface{}, clock uint64) error {
	v, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("value must be of type []byte")
	}

	// Get current sticker packs from db
	jrm := new(json.RawMessage)
	err := db.makeSelectRow(sf).Scan(&jrm)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Unmarshal the db sticker pack data into a sticker.StickerPackCollection
	sc := make(stickers.StickerPackCollection)
	if jrm != nil {
		mj, err := jrm.MarshalJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(mj, &sc)
		if err != nil {
			return err
		}
	}

	// Unmarshal the sync sticker pack data into a sticker.StickerPackCollection
	sp := make(stickers.StickerPackCollection)
	if v != nil && len(v) > 0 {
		err = json.Unmarshal(v, &sp)
		if err != nil {
			return err
		}
	}

	sc.Merge(sp)

	scv, err := sf.ValueHandler()(sc)
	if err != nil {
		return err
	}

	ls, err := db.GetSettingLastSynced(sf)
	if err != nil {
		return err
	}
	if clock <= ls {
		// If clock is less than or equal to last synced time, set clock to ls plus 1
		// Because we are merging and not overwriting
		clock = ls + 1
	}

	err = db.SetSettingLastSynced(sf, clock)
	if err != nil {
		return err
	}

	return db.saveSetting(sf, scv)
}
