package account

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/extkeys"
)

type address string

var errInvalidMnemonicPhraseLength = errors.New("invalid mnemonic phrase length")

type keyPair struct {
	purpose   extkeys.KeyPurpose
	address   string
	publicKey string
}

type userAccount struct {
	mnemonic          string
	masterExtendedKey *extkeys.ExtendedKey
	keyPairs          []*keyPair
}

type Onboarding struct {
	userAccounts map[string]*userAccount
}

func New(accountsCount int, mnemonicPhraseLength int) (*Onboarding, error) {
	onboarding := &Onboarding{
		userAccounts: make(map[string]*userAccount),
	}

	for i := 0; i < accountsCount; i++ {
		userAccount, err := onboarding.generateUserAccount(mnemonicPhraseLength)
		if err != nil {
			return nil, err
		}
		uuid := uuid.NewRandom()
		onboarding.userAccounts[uuid.String()] = userAccount
	}

	return onboarding, nil
}

func (o *Onboarding) Choose(id, password string) error {
	if _, ok := o.userAccounts[id]; !ok {
		return errors.New("id not found")
	}

	// return address, pubKey too?
	_, _, err = m.importExtendedKey(extkeys.KeyPurposeWallet, extKey, password)

	return err
}

func (o *Onboarding) generateUserAccount(mnemonicPhraseLength int) (*userAccount, error) {
	entropyStrength, err := mnemonicPhraseLengthToEntropyStrenght(mnemonicPhraseLength)
	if err != nil {
		return nil, err
	}

	mnemonic := extkeys.NewMnemonic()
	mnemonicPhrase, err := mnemonic.MnemonicPhrase(extkeys.EntropyStrength(entropyStrength), extkeys.EnglishLanguage)
	if err != nil {
		return nil, fmt.Errorf("can not create mnemonic seed: %v", err)
	}

	masterExtendedKey, err := extkeys.NewMaster(mnemonic.MnemonicSeed(mnemonicPhrase, ""))
	if err != nil {
		return nil, fmt.Errorf("can not create master extended key: %v", err)
	}

	walletKeyPair, err := o.deriveKeyPair(masterExtendedKey, extkeys.KeyPurposeWallet, 0)
	if err != nil {
		return nil, err
	}

	chatKeyPair := walletKeyPair

	kps := []*keyPair{walletKeyPair, chatKeyPair}

	userAccount := &userAccount{
		mnemonic:          mnemonicPhrase,
		masterExtendedKey: masterExtendedKey,
		keyPairs:          kps,
	}

	return userAccount, nil
}

func (o *Onboarding) deriveKeyPair(masterExtendedKey *extkeys.ExtendedKey, purpose extkeys.KeyPurpose, index uint32) (*keyPair, error) {
	extendedKey, err := masterExtendedKey.ChildForPurpose(purpose, index)
	if err != nil {
		return nil, err
	}

	privateKeyECDSA := extendedKey.ToECDSA()
	address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	publicKeyHex := hexutil.Encode(crypto.FromECDSAPub(&privateKeyECDSA.PublicKey))

	return &keyPair{
		purpose:   purpose,
		address:   address.Hex(),
		publicKey: publicKeyHex,
	}, nil
}

func mnemonicPhraseLengthToEntropyStrenght(length int) (extkeys.EntropyStrength, error) {
	if length < 12 || length > 24 || length%3 != 0 {
		return 0, errInvalidMnemonicPhraseLength
	}

	bitsLength := length * 11
	checksumLength := bitsLength % 32

	return extkeys.EntropyStrength(bitsLength - checksumLength), nil
}
