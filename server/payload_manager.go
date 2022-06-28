package server

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Payload struct {
	plain     []byte
	encrypted []byte
}

type PayloadManager struct {
	aesKey   []byte
	toSend   *Payload
	received *Payload
}

func NewPayloadManager(pk *ecdsa.PrivateKey) (*PayloadManager, error) {
	ek, err := makeEncryptionKey(pk)
	if err != nil {
		return nil, err
	}

	return &PayloadManager{ek, new(Payload), new(Payload)}, nil
}

func (pm *PayloadManager) Mount(data []byte) error {
	ep, err := common.Encrypt(data, pm.aesKey, rand.Reader)
	if err != nil {
		return err
	}

	pm.toSend.plain = data
	pm.toSend.encrypted = ep
	return nil
}

func (pm *PayloadManager) Receive(data []byte) error {
	pd, err := common.Decrypt(data, pm.aesKey)
	if err != nil {
		return err
	}

	pm.received.encrypted = data
	pm.received.plain = pd
	return nil
}

func (pm *PayloadManager) ToSend() []byte {
	return pm.toSend.encrypted
}

func (pm *PayloadManager) Received() []byte {
	return pm.received.plain
}

func (pm *PayloadManager) ResetPayload() {
	pm.toSend = new(Payload)
	pm.received = new(Payload)
}

// PayloadMarshaller is responsible for loading, parsing, marshalling, unmarshalling and storing of PairingServer
// payload data
type PayloadMarshaller struct {
	multiaccountDB *multiaccounts.Database

	keys         map[string][]byte
	multiaccount *multiaccounts.Account
	password     string
}

func NewPayloadMarshaller(db *multiaccounts.Database) *PayloadMarshaller {
	return &PayloadMarshaller{multiaccountDB: db}
}

func (pm *PayloadMarshaller) LoadPayloads(keystorePath, keyUID, password string) error {
	err := pm.loadKeys(keystorePath)
	if err != nil {
		return err
	}

	pm.multiaccount, err = pm.multiaccountDB.GetAccount(keyUID)
	if err != nil {
		return err
	}
	pm.password = password

	return nil
}

func (pm *PayloadMarshaller) loadKeys(keyStorePath string) error {
	pm.keys = make(map[string][]byte)

	fileWalker := func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileInfo.IsDir() || filepath.Dir(path) != keyStorePath {
			return nil
		}

		rawKeyFile, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("invalid account key file: %v", err)
		}

		accountKey := new(keystore.EncryptedKeyJSONV3)
		if err := json.Unmarshal(rawKeyFile, &accountKey); err != nil {
			return fmt.Errorf("failed to read key file: %s", err)
		}

		if len(accountKey.Address) != 40 {
			return fmt.Errorf("account key address has invalid length '%s'", accountKey.Address)
		}

		pm.keys[fileInfo.Name()] = rawKeyFile

		return nil
	}

	err := filepath.Walk(keyStorePath, fileWalker)
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	return nil
}

func (pm *PayloadMarshaller) StorePayloads(keystorePath, password string) error {
	err := pm.validateKeys(password)
	if err != nil {
		return err
	}

	err = pm.storeKeys(keystorePath)
	if err != nil {
		return err
	}

	err = pm.storeMultiAccount()
	if err != nil {
		return err
	}

	// TODO install PublicKey into settings, probably do this outside of StorePayloads
	return nil
}

func (pm *PayloadMarshaller) validateKeys(password string) error {
	for _, key := range pm.keys {
		k, err := keystore.DecryptKey(key, password)
		if err != nil {
			return err
		}

		err = generator.ValidateKeystoreExtendedKey(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pm *PayloadMarshaller) storeKeys(keyStorePath string) error {
	for name, data := range pm.keys {
		accountKey := new(keystore.EncryptedKeyJSONV3)
		if err := json.Unmarshal(data, &accountKey); err != nil {
			return fmt.Errorf("failed to read key file: %s", err)
		}

		if len(accountKey.Address) != 40 {
			return fmt.Errorf("account key address has invalid length '%s'", accountKey.Address)
		}

		err := ioutil.WriteFile(filepath.Join(keyStorePath, name), data, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pm *PayloadMarshaller) storeMultiAccount() error {
	return pm.multiaccountDB.SaveAccount(*pm.multiaccount)
}

func (pm *PayloadMarshaller) MarshalToProtobuf() ([]byte, error) {
	return proto.Marshal(&protobuf.LocalPairingPayload{
		Keys:         pm.accountKeysToProtobuf(),
		Multiaccount: pm.multiaccountToProtobuf(),
		Password:     pm.password,
	})
}

func (pm *PayloadMarshaller) accountKeysToProtobuf() []*protobuf.LocalPairingPayload_Key {
	var keys []*protobuf.LocalPairingPayload_Key
	for name, data := range pm.keys {
		keys = append(keys, &protobuf.LocalPairingPayload_Key{Name: name, Data: data})
	}
	return keys
}

func (pm *PayloadMarshaller) multiaccountToProtobuf() *protobuf.MultiAccount {
	var colourHashes []*protobuf.MultiAccount_ColourHash
	for _, index := range pm.multiaccount.ColorHash {
		var i []int64
		for _, is := range index {
			i = append(i, int64(is))
		}

		colourHashes = append(colourHashes, &protobuf.MultiAccount_ColourHash{Index: i})
	}

	var identityImages []*protobuf.MultiAccount_IdentityImage
	for _, ii := range pm.multiaccount.Images {
		identityImages = append(identityImages, &protobuf.MultiAccount_IdentityImage{
			KeyUid:       ii.KeyUID,
			Name:         ii.Name,
			Payload:      ii.Payload,
			Width:        int64(ii.Width),
			Height:       int64(ii.Height),
			Filesize:     int64(ii.FileSize),
			ResizeTarget: int64(ii.ResizeTarget),
			Clock:        ii.Clock,
		})
	}

	return &protobuf.MultiAccount{
		Name:           pm.multiaccount.Name,
		Timestamp:      pm.multiaccount.Timestamp,
		Identicon:      pm.multiaccount.Identicon,
		ColorHash:      colourHashes,
		ColorId:        pm.multiaccount.ColorID,
		KeycardPairing: pm.multiaccount.KeycardPairing,
		KeyUid:         pm.multiaccount.KeyUID,
		Images:         identityImages,
	}
}

func (pm *PayloadMarshaller) UnmarshalProtobuf(data []byte) error {
	pb := new(protobuf.LocalPairingPayload)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	pm.accountKeysFromProtobuf(pb.Keys)
	pm.multiaccountFromProtobuf(pb.Multiaccount)
	pm.password = pb.Password
	return nil
}

func (pm *PayloadMarshaller) accountKeysFromProtobuf(pbKeys []*protobuf.LocalPairingPayload_Key) {
	if pm.keys == nil {
		pm.keys = make(map[string][]byte)
	}

	for _, key := range pbKeys {
		pm.keys[key.Name] = key.Data
	}
}

func (pm *PayloadMarshaller) multiaccountFromProtobuf(pbMultiAccount *protobuf.MultiAccount) {
	var colourHash [][]int
	for _, index := range pbMultiAccount.ColorHash {
		var i []int
		for _, is := range index.Index {
			i = append(i, int(is))
		}

		colourHash = append(colourHash, i)
	}

	var identityImages []images.IdentityImage
	for _, ii := range pbMultiAccount.Images {
		identityImages = append(identityImages, images.IdentityImage{
			KeyUID:       ii.KeyUid,
			Name:         ii.Name,
			Payload:      ii.Payload,
			Width:        int(ii.Width),
			Height:       int(ii.Height),
			FileSize:     int(ii.Filesize),
			ResizeTarget: int(ii.ResizeTarget),
			Clock:        ii.Clock,
		})
	}

	pm.multiaccount = &multiaccounts.Account{
		Name:           pbMultiAccount.Name,
		Timestamp:      pbMultiAccount.Timestamp,
		Identicon:      pbMultiAccount.Identicon,
		ColorHash:      colourHash,
		ColorID:        pbMultiAccount.ColorId,
		KeycardPairing: pbMultiAccount.KeycardPairing,
		KeyUID:         pbMultiAccount.KeyUid,
		Images:         identityImages,
	}
}
