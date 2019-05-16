package account

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/extkeys"
)

var errInvalidMnemonicPhraseLength = errors.New("invalid mnemonic phrase length")

// OnboardingAccount is returned during onboarding and contains its ID and the mnemonic to re-generate the same account Info keys.
type OnboardingAccount struct {
	ID       string `json:"id"`
	mnemonic string
	Info     Info `json:"info"`
}

// Onboarding is a struct contains a slice of OnboardingAccount.
type Onboarding struct {
	accounts map[string]*OnboardingAccount
}

// NewOnboarding returns a new onboarding struct generating n accounts.
func NewOnboarding(n, mnemonicPhraseLength int) (*Onboarding, error) {
	onboarding := &Onboarding{
		accounts: make(map[string]*OnboardingAccount),
	}

	for i := 0; i < n; i++ {
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

// Accounts return the list of OnboardingAccount generated.
func (o *Onboarding) Accounts() []*OnboardingAccount {
	accounts := make([]*OnboardingAccount, 0)
	for _, a := range o.accounts {
		accounts = append(accounts, a)
	}

	return accounts
}

// Account returns an OnboardingAccount by id.
func (o *Onboarding) Account(id string) (*OnboardingAccount, error) {
	account, ok := o.accounts[id]
	if !ok {
		return nil, ErrOnboardingAccountNotFound
	}

	return account, nil
}

func (o *Onboarding) generateAccount(mnemonicPhraseLength int) (*OnboardingAccount, error) {
	entropyStrength, err := mnemonicPhraseLengthToEntropyStrenght(mnemonicPhraseLength)
	if err != nil {
		return nil, err
	}

	mnemonic := extkeys.NewMnemonic()
	mnemonicPhrase, err := mnemonic.MnemonicPhrase(entropyStrength, extkeys.EnglishLanguage)
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
