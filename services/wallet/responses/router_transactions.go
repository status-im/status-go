package responses

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

type SendDetails struct {
	Uuid                string                `json:"uuid"`
	SendType            int                   `json:"sendType"`
	FromAddress         types.Address         `json:"fromAddress"`
	ToAddress           types.Address         `json:"toAddress"`
	FromToken           string                `json:"fromToken"`
	ToToken             string                `json:"toToken"`
	FromAmount          string                `json:"fromAmount"` // total amount
	ToAmount            string                `json:"toAmount"`
	OwnerTokenBeingSent bool                  `json:"ownerTokenBeingSent"`
	ErrorResponse       *errors.ErrorResponse `json:"errorResponse,omitempty"`

	Username  string `json:"username"`
	PublicKey string `json:"publicKey"`
	PackID    string `json:"packId"`
}

type SigningDetails struct {
	Address       types.Address `json:"address"`
	AddressPath   string        `json:"addressPath"`
	KeyUid        string        `json:"keyUid"`
	SignOnKeycard bool          `json:"signOnKeycard"`
	Hashes        []types.Hash  `json:"hashes"`
}

type RouterTransactionsForSigning struct {
	SendDetails    *SendDetails    `json:"sendDetails"`
	SigningDetails *SigningDetails `json:"signingDetails"`
}

type RouterSentTransaction struct {
	FromAddress types.Address `json:"fromAddress"`
	ToAddress   types.Address `json:"toAddress"`
	FromChain   uint64        `json:"fromChain"`
	ToChain     uint64        `json:"toChain"`
	FromToken   string        `json:"fromToken"`
	ToToken     string        `json:"toToken"`
	Amount      string        `json:"amount"`    // amount sent
	AmountIn    string        `json:"amountIn"`  // amount that is "data" of tx (important for erc20 tokens)
	AmountOut   string        `json:"amountOut"` // amount that will be received
	Hash        types.Hash    `json:"hash"`
	ApprovalTx  bool          `json:"approvalTx"`
}

type RouterSentTransactions struct {
	SendDetails      *SendDetails             `json:"sendDetails"`
	SentTransactions []*RouterSentTransaction `json:"sentTransactions"`
}

func NewRouterSentTransaction(sendArgs *wallettypes.SendTxArgs, hash types.Hash, approvalTx bool) *RouterSentTransaction {
	addr := types.Address{}
	if sendArgs.To != nil {
		addr = *sendArgs.To
	}
	if sendArgs.Value == nil {
		sendArgs.Value = (*hexutil.Big)(big.NewInt(0))
	}
	if sendArgs.ValueIn == nil {
		sendArgs.ValueIn = (*hexutil.Big)(big.NewInt(0))
	}
	if sendArgs.ValueOut == nil {
		sendArgs.ValueOut = (*hexutil.Big)(big.NewInt(0))
	}
	return &RouterSentTransaction{
		FromAddress: sendArgs.From,
		ToAddress:   addr,
		FromChain:   sendArgs.FromChainID,
		ToChain:     sendArgs.ToChainID,
		FromToken:   sendArgs.FromTokenID,
		ToToken:     sendArgs.ToTokenID,
		Amount:      sendArgs.Value.String(),
		AmountIn:    sendArgs.ValueIn.String(),
		AmountOut:   sendArgs.ValueOut.String(),
		Hash:        hash,
		ApprovalTx:  approvalTx,
	}
}

func (sd *SendDetails) UpdateFields(inputParams requests.RouteInputParams) {
	sd.SendType = int(inputParams.SendType)
	sd.FromAddress = types.Address(inputParams.AddrFrom)
	sd.ToAddress = types.Address(inputParams.AddrTo)
	sd.FromToken = inputParams.TokenID
	sd.ToToken = inputParams.ToTokenID
	if inputParams.AmountIn != nil {
		sd.FromAmount = inputParams.AmountIn.String()
	}
	if inputParams.AmountOut != nil {
		sd.ToAmount = inputParams.AmountOut.String()
	}
	sd.OwnerTokenBeingSent = inputParams.TokenIDIsOwnerToken
	sd.Username = inputParams.Username
	sd.PublicKey = inputParams.PublicKey
	if inputParams.PackID != nil {
		sd.PackID = inputParams.PackID.String()
	}
}
