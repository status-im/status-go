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

// AccountInfo contains a PublicKey and an Address of an account.
type AccountInfo struct {
	PublicKey string `json:"publicKey"`
	Address   string `json:"address"`
}

// IdentifiedAccountInfo contains AccountInfo and the ID of an account.
type IdentifiedAccountInfo struct {
	AccountInfo
	ID string `json:"id"`
}

// CreatedAccountInfo contains IdentifiedAccountInfo and the mnemonic of an account.
type CreatedAccountInfo struct {
	IdentifiedAccountInfo
	Mnemonic string `json:"mnemonic"`
}
