package settings

import (
	"database/sql"
	"encoding/json"

	"github.com/status-im/status-go/common/stickers"
)

func AppendStickerPacks(sf SettingField, value interface{}, db *Database) error {
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

	return db.SaveSettingField(sf, sc)
}
