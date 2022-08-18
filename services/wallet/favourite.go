package wallet

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
)

type Favourite struct {
	Address common.Address `json:"address"`
	Name    string         `json:"name"`
}

type FavouriteManager struct {
	db *sql.DB
}

func (fm *FavouriteManager) GetFavourites() ([]Favourite, error) {
	rows, err := fm.db.Query(`SELECT address, name FROM favourites`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rst []Favourite
	for rows.Next() {
		favourite := Favourite{}
		err := rows.Scan(&favourite.Address, &favourite.Name)
		if err != nil {
			return nil, err
		}

		rst = append(rst, favourite)
	}

	return rst, nil
}

func (fm *FavouriteManager) AddFavourite(favourite Favourite) error {
	insert, err := fm.db.Prepare("INSERT OR REPLACE INTO favourites (address, name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	_, err = insert.Exec(favourite.Address, favourite.Name)
	return err
}

func (fm *FavouriteManager) DeleteFavourite(address common.Address) error {
	_, err := fm.db.Exec("DELETE FROM favourites WHERE address = ?", address)
	return err
}
