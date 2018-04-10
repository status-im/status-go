package sign

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Response is a byte payload returned by the signed function
type Response []byte

// Hex returns a string representation of the response
func (r Response) Hex() string {
	return hexutil.Encode(r[:])
}

// Bytes returns a byte representation of the response
func (r Response) Bytes() []byte {
	return []byte(r)
}

// Hash converts response to a hash.
func (r Response) Hash() common.Hash {
	return common.BytesToHash(r.Bytes())
}

// EmptyResponse is returned when an error occures
var EmptyResponse = Response([]byte{})

// Result is a result of a signing request, error or successful
type Result struct {
	Response Response
	Error    error
}

// newErrResult creates a result based on an empty response and an error
func newErrResult(err error) Result {
	return Result{
		Response: EmptyResponse,
		Error:    err,
	}
}
