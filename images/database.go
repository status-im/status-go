package images

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
)

type Database struct {
	db *sql.DB
}

type IdentityImage struct {
	Type         string `json:"type"`
	Payload      []byte
	Width        int `json:"width"`
	Height       int `json:"height"`
	FileSize     int `json:"file_size"`
	ResizeTarget int `json:"resize_target"`
}

func NewDatabase(db *sql.DB) Database {
	return Database{db: db}
}

func (d *Database) GetIdentityImages() ([]*IdentityImage, error) {
	rows, err := d.db.Query(`SELECT type, image_payload, width, height, file_size, resize_target FROM identity_images`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var iis []*IdentityImage
	for rows.Next() {
		ii := &IdentityImage{}
		err = rows.Scan(&ii.Type, &ii.Payload, &ii.Width, &ii.Height, &ii.FileSize, &ii.ResizeTarget)
		if err != nil {
			return nil, err
		}

		iis = append(iis, ii)
	}

	return iis, nil
}

func (i IdentityImage) GetDataURI() (string, error) {
	mt, err := GetMimeType(i.Payload)
	if err != nil {
		return "", err
	}

	b64 := base64.StdEncoding.EncodeToString(i.Payload)

	return "data:image/" + mt + ";base64," + b64, nil
}

func (d *Database) StoreIdentityImages() {
	// TODO
}

func (i IdentityImage) MarshalJSON() ([]byte, error) {
	uri, err := i.GetDataURI()
	if err != nil {
		return nil, err
	}

	temp := struct {
		Type         string `json:"type"`
		URI          string `json:"uri"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		FileSize     int    `json:"file_size"`
		ResizeTarget int    `json:"resize_target"`
	}{
		Type:         i.Type,
		URI:          uri,
		Width:        i.Width,
		Height:       i.Height,
		FileSize:     i.FileSize,
		ResizeTarget: i.ResizeTarget,
	}

	return json.Marshal(temp)
}
