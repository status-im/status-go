package sdk

import (
	"log"
	"os"

	"github.com/syndtr/goleveldb/leveldb"
)

// TODO : this should be received as an input
const KEY_ADDRESS = "hnny.address.lol"

func getAccountAddress() string {
	cwd, _ := os.Getwd()
	println(cwd + "/data")
	db, err := leveldb.OpenFile(cwd+"/data", nil)
	if err != nil {
		log.Fatal("can't open levelDB file. ERR: ", err)
	}
	defer db.Close()

	addressBytes, err := db.Get([]byte(KEY_ADDRESS), nil)
	if err != nil {
		log.Printf("Error while getting address: %v", err)
		return ""
	}
	return string(addressBytes)
}

func saveAccountAddress(address string) {
	cwd, _ := os.Getwd()
	db, err := leveldb.OpenFile(cwd+"/data", nil)
	if err != nil {
		log.Fatal("can't open levelDB file. ERR: ", err)
	}
	defer db.Close()

	db.Put([]byte(KEY_ADDRESS), []byte(address), nil)
}
