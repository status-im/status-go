package api

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/transactions"
)

// StatusBackend defines the contract for the Status.im service
type StatusBackend interface {
	// IsNodeRunning() bool                       // NOTE: Only used in tests
	StartNode(config *params.NodeConfig) error // NOTE: Only used in canary
	StartNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error
	StartNodeWithAccount(acc multiaccounts.Account, password string) error
	StartNodeWithAccountAndConfig(account multiaccounts.Account, password string, settings accounts.Settings, conf *params.NodeConfig, subaccs []accounts.Account) error
	StopNode() error
	// RestartNode() error // NOTE: Only used in tests

	UpdateRootDataDir(datadir string)

	// SelectAccount(loginParams account.LoginParams) error
	OpenAccounts() error
	GetAccounts() ([]multiaccounts.Account, error)
	// SaveAccount(account multiaccounts.Account) error
	SaveAccountAndStartNodeWithKey(acc multiaccounts.Account, password string, settings accounts.Settings, conf *params.NodeConfig, subaccs []accounts.Account, keyHex string) error
	Recover(rpcParams personal.RecoverParams) (types.Address, error)
	Logout() error

	CallPrivateRPC(inputJSON string) (string, error)
	CallRPC(inputJSON string) (string, error)
	GetNodesFromContract(rpcEndpoint string, contractAddress string) ([]string, error)
	HashTransaction(sendArgs transactions.SendTxArgs) (transactions.SendTxArgs, types.Hash, error)
	HashTypedData(typed typeddata.TypedData) (types.Hash, error)
	ResetChainData() error
	SendTransaction(sendArgs transactions.SendTxArgs, password string) (hash types.Hash, err error)
	SendTransactionWithSignature(sendArgs transactions.SendTxArgs, sig []byte) (hash types.Hash, err error)
	SignHash(hexEncodedHash string) (string, error)
	SignMessage(rpcParams personal.SignParams) (types.HexBytes, error)
	SignTypedData(typed typeddata.TypedData, address string, password string) (types.HexBytes, error)

	ConnectionChange(typ string, expensive bool)
	AppStateChange(state string)

	InjectChatAccount(chatKeyHex, encryptionKeyHex string) error // NOTE: Only used in lib and in tests
	ExtractGroupMembershipSignatures(signaturePairs [][2]string) ([]string, error)
	SignGroupMembership(content string) (string, error)
}
