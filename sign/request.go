package sign

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/account"
)

// CompleteFunc is a function that is called after the sign request is approved.
type CompleteFunc func(account *account.SelectedExtKey, password string, completeArgs *TxArgs) (Response, error)

// Meta represents any metadata that could be attached to a signing request.
// It will be JSON-serialized and used in notifications to the API consumer.
type Meta interface{}

// Request is a single signing request.
type Request struct {
	ID           string
	Method       string
	Meta         Meta
	context      context.Context
	locked       bool
	completeFunc CompleteFunc
	result       chan Result
}

// TxArgs represents the arguments to submit when signing a transaction
type TxArgs struct {
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
}

func newRequest(ctx context.Context, method string, meta Meta, completeFunc CompleteFunc) *Request {
	return &Request{
		ID:           uuid.New(),
		Method:       method,
		Meta:         meta,
		context:      ctx,
		locked:       false,
		completeFunc: completeFunc,
		result:       make(chan Result, 1),
	}
}
