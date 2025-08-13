package api

import (
	"crypto/ecdsa"

	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

// StatusBackend defines the contract for the Status.im service
type StatusBackend interface {
	StartNode(config *params.NodeConfig) error // NOTE: Only used in canary
	StartNodeWithKey(acc multiaccounts.Account, password string, keyHex string, conf *params.NodeConfig) error
	StartNodeWithAccount(acc multiaccounts.Account, password string, conf *params.NodeConfig, chatKey *ecdsa.PrivateKey) error
	StartNodeWithChatKeyOrMnemonic(
		request *requests.CreateAccount,
		mnemonic string, // empty mnemonic is used for keycard account, not empty for regular account
		keycardData *requests.KeycardData, // nil for regular account, not nil for keycard account
		restoreAccount bool,
		fetchBackup bool,
	) (*multiaccounts.Account, error)
	StartNodeWithAccountAndInitialConfig(
		multiAccount *multiaccounts.Account,
		password string,
		settings settings.Settings,
		nodecfg *params.NodeConfig,
		keypair *accounts.Keypair,
		chatKey *ecdsa.PrivateKey) error
	StopNode() error

	GetNodeConfig() (*params.NodeConfig, error)
	UpdateRootDataDir(datadir string)

	SelectAccount(loginParams LoginParams, privateKey *ecdsa.PrivateKey) error
	OpenAccounts() error
	GetAccounts() ([]multiaccounts.Account, error)
	LocalPairingStarted() error
	SaveAccount(account multiaccounts.Account) error
	Recover(rpcParams personal.RecoverParams) (types.Address, error)
	Logout() error

	CallPrivateRPC(inputJSON string) (string, error)
	CallRPC(inputJSON string) (string, error)
	HashTransaction(sendArgs wallettypes.SendTxArgs) (wallettypes.SendTxArgs, types.Hash, error)
	HashTypedData(typed typeddata.TypedData) (types.Hash, error)
	HashTypedDataV4(typed signercore.TypedData) (types.Hash, error)
	SendTransaction(sendArgs wallettypes.SendTxArgs, password string) (hash types.Hash, err error)
	SendTransactionWithChainID(chainID uint64, sendArgs wallettypes.SendTxArgs, password string) (hash types.Hash, err error)
	SendTransactionWithSignature(sendArgs wallettypes.SendTxArgs, sig []byte) (hash types.Hash, err error)
	SignHash(hexEncodedHash string) (string, error)
	SignMessage(rpcParams personal.SignParams) (types.HexBytes, error)
	SignTypedData(typed typeddata.TypedData, address string, password string) (types.HexBytes, error)
	SignTypedDataV4(typed signercore.TypedData, address string, password string) (types.HexBytes, error)

	ConnectionChange(typ string, expensive bool)
	AppStateChange(state AppState)

	ExtractGroupMembershipSignatures(signaturePairs [][2]string) ([]string, error)
	SignGroupMembership(content string) (string, error)
}
