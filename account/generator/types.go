package generator

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"github.com/status-im/extkeys"

	"github.com/status-im/status-go/account/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

var (
	ErrInvalidKeystoreExtendedKey = errors.New("PrivateKey and ExtendedKey are different")
)

type Account struct {
	privateKey  *ecdsa.PrivateKey
	extendedKey *extkeys.ExtendedKey
}

func NewAccount(privateKey *ecdsa.PrivateKey, extKey *extkeys.ExtendedKey) *Account {
	if privateKey == nil && extKey != nil {
		privateKey = extKey.ToECDSA()
	}

	return &Account{
		privateKey:  privateKey,
		extendedKey: extKey,
	}
}

func (a *Account) ExtendedKey() *extkeys.ExtendedKey {
	return a.extendedKey
}

func (a *Account) PrivateKey() *ecdsa.PrivateKey {
	return a.privateKey
}

func (a *Account) PrivateKeyHex() string {
	return types.EncodeHex(crypto.FromECDSA(a.privateKey))
}

func (a *Account) PublicKey() *ecdsa.PublicKey {
	return &a.privateKey.PublicKey
}

func (a *Account) PublicKeyHex() string {
	if a.privateKey == nil {
		return ""
	}
	return types.EncodeHex(crypto.FromECDSAPub(&a.privateKey.PublicKey))
}

func (a *Account) Address() types.Address {
	if a.privateKey == nil {
		return types.Address{}
	}
	return crypto.PubkeyToAddress(a.privateKey.PublicKey)
}

func (a *Account) HasExtendedKey() bool {
	return a.extendedKey != nil && !a.extendedKey.IsZeroed()
}

func (a *Account) KeyUID() string {
	if a.privateKey == nil {
		return ""
	}
	keyUID := sha256.Sum256(crypto.FromECDSAPub(&a.privateKey.PublicKey))
	return types.EncodeHex(keyUID[:])
}

func (a *Account) ToAccountInfo() AccountInfo {
	if a.privateKey == nil {
		return AccountInfo{}
	}
	privateKeyHex := types.EncodeHex(crypto.FromECDSA(a.privateKey))
	publicKeyHex := types.EncodeHex(crypto.FromECDSAPub(&a.privateKey.PublicKey))
	addressHex := crypto.PubkeyToAddress(a.privateKey.PublicKey).Hex()

	return AccountInfo{
		PrivateKey: privateKeyHex,
		PublicKey:  publicKeyHex,
		Address:    addressHex,
	}
}

func (a *Account) ToIdentifiedAccountInfo() IdentifiedAccountInfo {
	if a.privateKey == nil {
		return IdentifiedAccountInfo{}
	}
	info := a.ToAccountInfo()
	keyUID := sha256.Sum256(crypto.FromECDSAPub(&a.privateKey.PublicKey))
	keyUIDHex := types.EncodeHex(keyUID[:])
	return IdentifiedAccountInfo{
		AccountInfo: info,
		KeyUID:      keyUIDHex,
	}
}

func (a *Account) ToGeneratedAccountInfo(mnemonic string) GeneratedAccountInfo {
	idInfo := a.ToIdentifiedAccountInfo()
	return GeneratedAccountInfo{
		IdentifiedAccountInfo: idInfo,
		Mnemonic:              mnemonic,
	}
}

// ValidateExtendedKey validates the keystore keys, checking that ExtendedKey is the extended key of PrivateKey
func (a *Account) ValidateExtendedKey() error {
	if a.extendedKey == nil || a.extendedKey.IsZeroed() {
		return nil
	}

	privKey := crypto.FromECDSA(a.privateKey)
	extKey := crypto.FromECDSA(a.extendedKey.ToECDSA())
	if !bytes.Equal(privKey, extKey) {
		return ErrInvalidKeystoreExtendedKey
	}

	return nil
}

type AccountInfo struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address"`
}

func (a AccountInfo) MarshalJSON() ([]byte, error) {
	type Alias AccountInfo
	ext, err := common.ExtendStructWithPubKeyData(a.PublicKey, Alias(a))
	if err != nil {
		return nil, err
	}
	return json.Marshal(ext)
}

type IdentifiedAccountInfo struct {
	AccountInfo
	KeyUID string `json:"keyUid"`
}

func (a *IdentifiedAccountInfo) ToGeneratedAccountInfo(mnemonic string) GeneratedAccountInfo {
	return GeneratedAccountInfo{
		IdentifiedAccountInfo: *a,
		Mnemonic:              mnemonic,
	}
}

func (i IdentifiedAccountInfo) MarshalJSON() ([]byte, error) {
	accountInfoJSON, err := i.AccountInfo.MarshalJSON()
	if err != nil {
		return nil, err
	}
	type info struct {
		KeyUID string `json:"keyUid"`
	}
	infoJSON, err := json.Marshal(info{
		KeyUID: i.KeyUID,
	})
	if err != nil {
		return nil, err
	}
	infoJSON[0] = ','
	return append(accountInfoJSON[:len(accountInfoJSON)-1], infoJSON...), nil
}

type GeneratedAccountInfo struct {
	IdentifiedAccountInfo
	Mnemonic string `json:"mnemonic"`
}

func (g GeneratedAccountInfo) MarshalJSON() ([]byte, error) {
	accountInfoJSON, err := g.IdentifiedAccountInfo.MarshalJSON()
	if err != nil {
		return nil, err
	}
	type info struct {
		Mnemonic string `json:"mnemonic"`
	}
	infoJSON, err := json.Marshal(info{
		Mnemonic: g.Mnemonic,
	})
	if err != nil {
		return nil, err
	}
	infoJSON[0] = ','
	return append(accountInfoJSON[:len(accountInfoJSON)-1], infoJSON...), nil
}

type GeneratedAndDerivedAccountInfo struct {
	GeneratedAccountInfo
	Derived map[string]AccountInfo `json:"derived"`
}

func (g GeneratedAndDerivedAccountInfo) MarshalJSON() ([]byte, error) {
	accountInfoJSON, err := g.GeneratedAccountInfo.MarshalJSON()
	if err != nil {
		return nil, err
	}
	type info struct {
		Derived map[string]AccountInfo `json:"derived"`
	}
	infoJSON, err := json.Marshal(info{
		Derived: g.Derived,
	})
	if err != nil {
		return nil, err
	}
	infoJSON[0] = ','
	return append(accountInfoJSON[:len(accountInfoJSON)-1], infoJSON...), nil
}
