package sign

import "github.com/ethereum/go-ethereum/common"

// Result is a result of a signing request, error or successful
type Result struct {
	Hash  common.Hash
	Error error
}
