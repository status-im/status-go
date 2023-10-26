package community

import (
	"database/sql"
	"fmt"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type DataDB struct {
	db *sql.DB
}

func NewDataDB(sqlDb *sql.DB) *DataDB {
	return &DataDB{
		db: sqlDb,
	}
}

const communityDataColumns = "id, name, color, image"
const selectCommunityDataColumns = "name, color, image"

func (o *DataDB) SetCommunityInfo(id string, c thirdparty.CommunityInfo) (err error) {
	tx, err := o.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	update, err := tx.Prepare(fmt.Sprintf(`INSERT OR REPLACE INTO community_data_cache (%s) 
		VALUES (?, ?, ?, ?)`, communityDataColumns))
	if err != nil {
		return err
	}

	_, err = update.Exec(
		id,
		c.CommunityName,
		c.CommunityColor,
		c.CommunityImage,
	)

	return err
}

func (o *DataDB) GetCommunityInfo(id string) (*thirdparty.CommunityInfo, error) {
	var ret thirdparty.CommunityInfo

	getData, err := o.db.Prepare(fmt.Sprintf(`SELECT %s
		FROM community_data_cache
		WHERE id=?`, selectCommunityDataColumns))
	if err != nil {
		return nil, err
	}

	row := getData.QueryRow(id)

	err = row.Scan(
		&ret.CommunityName,
		&ret.CommunityColor,
		&ret.CommunityImage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &ret, nil
}
