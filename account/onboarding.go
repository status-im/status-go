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

type OnboardingAccount struct {
	ID       string `json:"id"`
	mnemonic string
	Info     Info `json:"info"`
}

type Onboarding struct {
	accounts map[string]*OnboardingAccount
}

func NewOnboarding(accountsCount, mnemonicPhraseLength int) (*Onboarding, error) {
	onboarding := &Onboarding{
		accounts: make(map[string]*OnboardingAccount),
	}

	for i := 0; i < accountsCount; i++ {
		account, err := onboarding.generateAccount(mnemonicPhraseLength)
		if err != nil {
			return nil, err
		}
		uuid := uuid.NewRandom().String()
		account.ID = uuid
		onboarding.accounts[uuid] = account
	}

	return onboarding, nil
}

func (o *Onboarding) Accounts() []*OnboardingAccount {
	accounts := make([]*OnboardingAccount, 0)
	for _, a := range o.accounts {
		accounts = append(accounts, a)
	}

	return accounts
}

func (o *Onboarding) Account(id string) (*OnboardingAccount, error) {
	account, ok := o.accounts[id]
	if !ok {
		return nil, errors.New("id not found")
	}

	return account, nil
}

func (o *Onboarding) generateAccount(mnemonicPhraseLength int) (*OnboardingAccount, error) {
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

	walletAddress, walletPubKey, err := o.deriveAccount(masterExtendedKey, extkeys.KeyPurposeWallet, 0)
	if err != nil {
		return nil, err
	}

	info := Info{
		WalletAddress: walletAddress,
		WalletPubKey:  walletPubKey,
		ChatAddress:   walletAddress,
		ChatPubKey:    walletPubKey,
	}

	account := &OnboardingAccount{
		mnemonic: mnemonicPhrase,
		Info:     info,
	}

	return account, nil
}

func (o *Onboarding) deriveAccount(masterExtendedKey *extkeys.ExtendedKey, purpose extkeys.KeyPurpose, index uint32) (string, string, error) {
	extendedKey, err := masterExtendedKey.ChildForPurpose(purpose, index)
	if err != nil {
		return "", "", err
	}

	privateKeyECDSA := extendedKey.ToECDSA()
	address := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	publicKeyHex := hexutil.Encode(crypto.FromECDSAPub(&privateKeyECDSA.PublicKey))

	return address.Hex(), publicKeyHex, nil
}

func mnemonicPhraseLengthToEntropyStrenght(length int) (extkeys.EntropyStrength, error) {
	if length < 12 || length > 24 || length%3 != 0 {
		return 0, errInvalidMnemonicPhraseLength
	}

	bitsLength := length * 11
	checksumLength := bitsLength % 32

	return extkeys.EntropyStrength(bitsLength - checksumLength), nil
}
