package protocol

import (
	"github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/types"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

//go:generate go tool mockgen -package=mock_protocol_accounts_manager -source=accounts_manager_interface.go -destination=mock/messenger_accounts_manager.go

// AccountsManager interface for mocking purposes
type AccountsManager interface {
	GetVerifiedWalletAccount(address cryptotypes.Address, password string) (*generator.Account, error)
	DeleteAccount(address cryptotypes.Address, password string, clock uint64) (account *types.Account, err error)
	DeleteKeypair(keyUID string, password string, clock uint64) (keypair *types.Keypair, err error)
	MakeSeedPhraseKeypairFullyOperable(mnemonic string, password string, clock uint64) (string, error)
	MakePrivateKeyKeypairFullyOperable(privateKey string, password string, clock uint64) (string, error)
	SaveOrUpdateKeycard(keycard *types.Keycard, password string, clock uint64) error
	AddAccounts(keyUID string, accounts []*types.Account, password string) error
	CreateKeypairFromMnemonicAndStore(mnemonic string, password string, keypairName string,
		walletAccount *types.AccountCreationDetails, profile bool, clock uint64) (keypair *types.Keypair, err error)
	AddKeypairStoredToKeycard(keyUID string, masterAddress string, name string,
		xpub string, coldWallet types.ColdWalletType, walletAccounts []*types.Account, clock uint64) (keypair *types.Keypair, err error)
	CreateKeypairFromPrivateKeyAndStore(privateKey string, password string, keypairName string,
		walletAccount *types.AccountCreationDetails, clock uint64) (keypair *types.Keypair, err error)
	MigrateNonProfileKeycardKeypairToApp(mnemonic string, password string, clock uint64) (string, error)
}
