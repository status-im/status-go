package sign

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/geth/account"
)

type completeFunc func(*account.SelectedExtKey) (common.Hash, error)

// Meta represents any metadata that could be attached to a signing request.
// It will be JSON-serialized and used in notifications to the API consumer.
type Meta interface{}

// Request is a single signing request.
type Request struct {
	ID           string
	Meta         Meta
	context      context.Context
	locked       bool
	completeFunc completeFunc
	result       chan Result
}

func newRequest(ctx context.Context, meta Meta, completeFunc completeFunc) *Request {
	return &Request{
		ID:           uuid.New(),
		Meta:         meta,
		context:      ctx,
		locked:       false,
		completeFunc: completeFunc,
		result:       make(chan Result, 1),
	}
}
