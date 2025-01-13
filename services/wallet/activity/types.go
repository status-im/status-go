package activity

import (
	"encoding/json"
	"fmt"
	"math/big"

	// used for embedding the sql query in the binary
	_ "embed"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/transfer"
)

type Entry struct {
	payloadType               ac.PayloadType
	transaction               *transfer.TransactionIdentity   // ID for SimpleTransactionPT and PendingTransactionPT.	Origin transaction for MultiTransactionPT
	id                        common.MultiTransactionIDType   // ID for MultiTransactionPT
	transactions              []*transfer.TransactionIdentity // List of transactions for MultiTransactionPT
	timestamp                 int64
	activityType              ac.Type
	activityStatus            ac.Status
	amountOut                 *hexutil.Big // Used for activityType SendAT, SwapAT, BridgeAT
	amountIn                  *hexutil.Big // Used for activityType ReceiveAT, BuyAT, SwapAT, BridgeAT, ApproveAT
	tokenOut                  *ac.Token    // Used for activityType SendAT, SwapAT, BridgeAT
	tokenIn                   *ac.Token    // Used for activityType ReceiveAT, BuyAT, SwapAT, BridgeAT, ApproveAT
	symbolOut                 *string
	symbolIn                  *string
	sender                    *eth.Address
	recipient                 *eth.Address
	chainIDOut                *common.ChainID
	chainIDIn                 *common.ChainID
	transferType              *ac.TransferType
	contractAddress           *eth.Address // Used for contract deployment
	communityID               *string
	interactedContractAddress *eth.Address
	approvalSpender           *eth.Address

	isNew bool // isNew is used to indicate if the entry is newer than session start (changed state also)
}

func (e *Entry) Key() string {
	if e.payloadType == ac.MultiTransactionPT {
		key := fmt.Sprintf("%d", e.id)
		for _, t := range e.transactions {
			key += fmt.Sprintf("-%s", t.Key())
		}
		return key
	}
	return e.transaction.Key()
}

// Only used for JSON marshalling

func (e *Entry) MarshalJSON() ([]byte, error) {
	data := ac.EntryData{
		Key:                       e.Key(),
		Timestamp:                 &e.timestamp,
		ActivityType:              &e.activityType,
		ActivityStatus:            &e.activityStatus,
		AmountOut:                 e.amountOut,
		AmountIn:                  e.amountIn,
		TokenOut:                  e.tokenOut,
		TokenIn:                   e.tokenIn,
		SymbolOut:                 e.symbolOut,
		SymbolIn:                  e.symbolIn,
		Sender:                    e.sender,
		Recipient:                 e.recipient,
		ChainIDOut:                e.chainIDOut,
		ChainIDIn:                 e.chainIDIn,
		TransferType:              e.transferType,
		ContractAddress:           e.contractAddress,
		CommunityID:               e.communityID,
		InteractedContractAddress: e.interactedContractAddress,
		ApprovalSpender:           e.approvalSpender,
	}

	if e.payloadType == ac.MultiTransactionPT {
		data.ID = common.NewAndSet(e.id)
		data.Transactions = e.transactions
	} else {
		data.Transaction = e.transaction
	}

	data.PayloadType = e.payloadType
	if e.isNew {
		data.IsNew = &e.isNew
	}

	return json.Marshal(data)
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	aux := ac.EntryData{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.payloadType = aux.PayloadType
	e.transaction = aux.Transaction
	if aux.ID != nil {
		e.id = *aux.ID
	}
	e.transactions = aux.Transactions
	if aux.Timestamp != nil {
		e.timestamp = *aux.Timestamp
	}
	if aux.ActivityType != nil {
		e.activityType = *aux.ActivityType
	}
	if aux.ActivityStatus != nil {
		e.activityStatus = *aux.ActivityStatus
	}
	e.amountOut = aux.AmountOut
	e.amountIn = aux.AmountIn
	e.tokenOut = aux.TokenOut
	e.tokenIn = aux.TokenIn
	e.symbolOut = aux.SymbolOut
	e.symbolIn = aux.SymbolIn
	e.sender = aux.Sender
	e.recipient = aux.Recipient
	e.chainIDOut = aux.ChainIDOut
	e.chainIDIn = aux.ChainIDIn
	e.transferType = aux.TransferType
	e.communityID = aux.CommunityID
	e.interactedContractAddress = aux.InteractedContractAddress
	e.approvalSpender = aux.ApprovalSpender

	e.isNew = aux.IsNew != nil && *aux.IsNew

	return nil
}

func (e *Entry) PayloadType() ac.PayloadType {
	return e.payloadType
}

func (e *Entry) isNFT() bool {
	tt := e.transferType
	return tt != nil && (*tt == ac.TransferTypeErc721 || *tt == ac.TransferTypeErc1155) && ((e.tokenIn != nil && e.tokenIn.TokenID != nil) || (e.tokenOut != nil && e.tokenOut.TokenID != nil))
}

func tokenIDToWalletBigInt(tokenID *hexutil.Big) *bigint.BigInt {
	if tokenID == nil {
		return nil
	}

	bi := new(big.Int).Set((*big.Int)(tokenID))
	return &bigint.BigInt{Int: bi}
}

func (e *Entry) anyIdentity() *thirdparty.CollectibleUniqueID {
	if e.tokenIn != nil {
		return &thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				ChainID: e.tokenIn.ChainID,
				Address: e.tokenIn.Address,
			},
			TokenID: tokenIDToWalletBigInt(e.tokenIn.TokenID),
		}
	} else if e.tokenOut != nil {
		return &thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				ChainID: e.tokenOut.ChainID,
				Address: e.tokenOut.Address,
			},
			TokenID: tokenIDToWalletBigInt(e.tokenOut.TokenID),
		}
	}
	return nil
}

func (e *Entry) getIdentity() EntryIdentity {
	return EntryIdentity{
		payloadType: e.payloadType,
		id:          e.id,
		transaction: e.transaction,
	}
}
