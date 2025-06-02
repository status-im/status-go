package common

import (
	"fmt"
	"math"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/services/wallet/common"
)

type PayloadType = int

// Beware: please update multiTransactionTypeToActivityType if changing this enum
const (
	MultiTransactionPT PayloadType = iota + 1
	SimpleTransactionPT
	PendingTransactionPT
)

type TransferType = int64

const (
	TransferTypeEth TransferType = iota + 1
	TransferTypeErc20
	TransferTypeErc721
	TransferTypeErc1155
)

const (
	L1FinalizationDuration  = 960    // A block on layer 1 is every 12s, finalization require 64 blocks. A buffer of 16 blocks is added to not create false positives.
	BSCFinalizationDuration = 9      // BSC uses Fast Finality, finalization time is 7.5 seconds approx. A buffer of 1.5 seconds is added to not create false positives.
	L2FinalizationDuration  = 648000 // 7.5 days in seconds for layer 2 finalization. 0.5 day is buffer to not create false positive.
)

const (
	NoLimit = 0
)

type Type uint64

const (
	SendAT Type = iota
	ReceiveAT
	BuyAT
	SwapAT
	BridgeAT
	ContractDeploymentAT
	MintAT
	ApproveAT
	UnknownAT = math.MaxUint64
)

type Status int

const (
	FailedAS    Status = iota // failed status or at least one failed transaction for multi-transactions
	PendingAS                 // in pending DB or at least one transaction in pending for multi-transactions
	CompleteAS                // success status
	FinalizedAS               // all multi-transactions have success status
)

type TokenType int

const (
	Native TokenType = iota
	Erc20
	Erc721
	Erc1155
)

type TransactionIdentity struct {
	ChainID common.ChainID `json:"chainId"`
	Hash    eth.Hash       `json:"hash"`
	Address eth.Address    `json:"address"`
}

func (tid *TransactionIdentity) Key() string {
	return fmt.Sprintf("%d-%s-%s", tid.ChainID, tid.Hash.Hex(), tid.Address.Hex())
}

// Token supports all tokens. Some fields might be optional, depending on the TokenType
type Token struct {
	TokenType TokenType `json:"tokenType"`
	// ChainID is used for TokenType.Native only to lookup the symbol, all chains will be included in the token filter
	ChainID common.ChainID `json:"chainId"`
	Address eth.Address    `json:"address,omitempty"`
	TokenID *hexutil.Big   `json:"tokenId,omitempty"`
}

type EntryData struct {
	PayloadType               PayloadType                    `json:"payloadType"`
	Key                       string                         `json:"key"`
	Transaction               *TransactionIdentity           `json:"transaction,omitempty"`
	ID                        *common.MultiTransactionIDType `json:"id,omitempty"`
	Transactions              []*TransactionIdentity         `json:"transactions,omitempty"`
	Timestamp                 *int64                         `json:"timestamp,omitempty"`
	ActivityType              *Type                          `json:"activityType,omitempty"`
	ActivityStatus            *Status                        `json:"activityStatus,omitempty"`
	AmountOut                 *hexutil.Big                   `json:"amountOut,omitempty"`
	AmountIn                  *hexutil.Big                   `json:"amountIn,omitempty"`
	TokenOut                  *Token                         `json:"tokenOut,omitempty"`
	TokenIn                   *Token                         `json:"tokenIn,omitempty"`
	SymbolOut                 *string                        `json:"symbolOut,omitempty"`
	SymbolIn                  *string                        `json:"symbolIn,omitempty"`
	Sender                    *eth.Address                   `json:"sender,omitempty"`
	Recipient                 *eth.Address                   `json:"recipient,omitempty"`
	ChainIDOut                *common.ChainID                `json:"chainIdOut,omitempty"`
	ChainIDIn                 *common.ChainID                `json:"chainIdIn,omitempty"`
	TransferType              *TransferType                  `json:"transferType,omitempty"`
	ContractAddress           *eth.Address                   `json:"contractAddress,omitempty"`
	CommunityID               *string                        `json:"communityId,omitempty"`
	InteractedContractAddress *eth.Address                   `json:"interactedContractAddress,omitempty"`
	ApprovalSpender           *eth.Address                   `json:"approvalSpender,omitempty"`

	IsNew *bool `json:"isNew,omitempty"`

	NftName *string `json:"nftName,omitempty"`
	NftURL  *string `json:"nftUrl,omitempty"`
}
