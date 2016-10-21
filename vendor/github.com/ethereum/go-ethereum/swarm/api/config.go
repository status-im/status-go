// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/services/swap"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	port = "8500"
)

//  by default ens root is  north internal
var (
	toyNetEnsRoot = common.HexToAddress("0xd344889e0be3e9ef6c26b0f60ef66a32e83c1b69")
)

// separate bzz directories
// allow several bzz nodes running in parallel
type Config struct {
	// serialised/persisted fields
	*storage.StoreParams
	*storage.ChunkerParams
	*network.HiveParams
	Swap *swap.SwapParams
	*network.SyncParams
	Path      string
	Port      string
	PublicKey string
	BzzKey    string
	EnsRoot   common.Address
}

// config is agnostic to where private key is coming from
// so managing accounts is outside swarm and left to wrappers
func NewConfig(path string, contract common.Address, prvKey *ecdsa.PrivateKey) (self *Config, err error) {

	address := crypto.PubkeyToAddress(prvKey.PublicKey) // default beneficiary address
	dirpath := filepath.Join(path, common.Bytes2Hex(address.Bytes()))
	err = os.MkdirAll(dirpath, os.ModePerm)
	if err != nil {
		return
	}
	confpath := filepath.Join(dirpath, "config.json")
	var data []byte
	pubkey := crypto.FromECDSAPub(&prvKey.PublicKey)
	pubkeyhex := common.ToHex(pubkey)
	keyhex := crypto.Sha3Hash(pubkey).Hex()

	self = &Config{
		SyncParams:    network.NewSyncParams(dirpath),
		HiveParams:    network.NewHiveParams(dirpath),
		ChunkerParams: storage.NewChunkerParams(),
		StoreParams:   storage.NewStoreParams(dirpath),
		Port:          port,
		Path:          dirpath,
		Swap:          swap.DefaultSwapParams(contract, prvKey),
		PublicKey:     pubkeyhex,
		BzzKey:        keyhex,
		EnsRoot:       toyNetEnsRoot,
	}
	data, err = ioutil.ReadFile(confpath)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
		// file does not exist
		// write out config file
		err = self.Save()
		if err != nil {
			err = fmt.Errorf("error writing config: %v", err)
		}
		return
	}
	// file exists, deserialise
	err = json.Unmarshal(data, self)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %v", err)
	}
	// check public key
	if pubkeyhex != self.PublicKey {
		return nil, fmt.Errorf("public key does not match the one in the config file %v != %v", pubkeyhex, self.PublicKey)
	}
	if keyhex != self.BzzKey {
		return nil, fmt.Errorf("bzz key does not match the one in the config file %v != %v", keyhex, self.BzzKey)
	}
	self.Swap.SetKey(prvKey)

	if (self.EnsRoot == common.Address{}) {
		self.EnsRoot = toyNetEnsRoot
	}

	return
}

func (self *Config) Save() error {
	data, err := json.MarshalIndent(self, "", "    ")
	if err != nil {
		return err
	}
	err = os.MkdirAll(self.Path, os.ModePerm)
	if err != nil {
		return err
	}
	confpath := filepath.Join(self.Path, "config.json")
	return ioutil.WriteFile(confpath, data, os.ModePerm)
}
