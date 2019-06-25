package statusaccounts

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/extkeys"
)

type account struct {
	privateKey  *ecdsa.PrivateKey
	extendedKey *extkeys.ExtendedKey
}

func (a *account) toAccountInfo() AccountInfo {
	publicKeyHex := hexutil.Encode(crypto.FromECDSAPub(&a.privateKey.PublicKey))
	addressHex := crypto.PubkeyToAddress(a.privateKey.PublicKey).Hex()

	return AccountInfo{
		PublicKey: publicKeyHex,
		Address:   addressHex,
	}
}

func (a *account) toIdentifiedAccountInfo(id string) IdentifiedAccountInfo {
	info := a.toAccountInfo()
	return IdentifiedAccountInfo{
		AccountInfo: info,
		ID:          id,
	}
}

func (a *account) toCreatedAccountInfo(id string, mnemonic string) CreatedAccountInfo {
	idInfo := a.toIdentifiedAccountInfo(id)
	return CreatedAccountInfo{
		IdentifiedAccountInfo: idInfo,
		Mnemonic:              mnemonic,
	}
}

type AccountInfo struct {
	PublicKey string `json:"publicKey"`
	Address   string `json:"address"`
}

type IdentifiedAccountInfo struct {
	AccountInfo
	ID string `json:"id"`
}

type CreatedAccountInfo struct {
	IdentifiedAccountInfo
	Mnemonic string `json:"mnemonic"`
}
