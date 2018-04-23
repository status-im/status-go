package sdk

import (
	"errors"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
)

type AccountStorer interface {
	GetAddress(string) (string, error)
	SetAddress(string, string) error
}

type AccountStore struct {
	keyAddress string
}

func (a *AccountStore) GetAddress(keyAddress string) (string, error) {
	cwd, _ := os.Getwd()
	db, err := leveldb.OpenFile(cwd+"/data", nil)
	if err != nil {
		return "", errors.New("Can't open levelDB file. ERR: " + err.Error())
	}
	defer db.Close()

	addressBytes, err := db.Get([]byte(keyAddress), nil)
	if err != nil {
		return "", errors.New("Error while getting address: " + err.Error())
	}

	return string(addressBytes), nil
}

func (a *AccountStore) SetAddress(keyAddress string, address string) error {
	cwd, _ := os.Getwd()
	db, err := leveldb.OpenFile(cwd+"/data", nil)
	if err != nil {
		return errors.New("can't open levelDB file. ERR: " + err.Error())
	}
	defer db.Close()

	db.Put([]byte(keyAddress), []byte(address), nil)

	return nil
}
