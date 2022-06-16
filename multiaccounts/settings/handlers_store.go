package settings

import (
	"database/sql"
	"encoding/json"
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
	jrm := new(json.RawMessage)
	err := db.makeSelectRow(sf).Scan(&jrm)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var sc stickers.StickerPackCollection
	if jrm == nil {
		sc = make(stickers.StickerPackCollection)
	} else {
		mj, err := jrm.MarshalJSON()
		if err != nil {
			return err
		}

		sc = make(stickers.StickerPackCollection)
		err = json.Unmarshal(mj, &sc)
		if err != nil {
			return err
		}
	}

	sp := make(stickers.StickerPackCollection)
	err = json.Unmarshal(value.([]byte), &sp)
	if err != nil {
		return err
	}

	sc.Merge(sp)

	v, err := sf.ValueHandler()(sc)
	if err != nil {
		return err
	}

	err = db.saveSyncSetting(sf, v, clock)
	if err == errors.ErrNewClockOlderThanCurrent {
		logger := logutils.Logger()
		logger.Info("StickerPacksStoreHandler - saveSyncSetting :", zap.Error(err))
		return nil
	}
	return err
}
