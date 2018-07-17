package chat

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type SendPublicMessageRPC struct {
	Sig     string
	Chat    string
	Payload hexutil.Bytes
}

type SendDirectMessageRPC struct {
	Sig     string
	Payload hexutil.Bytes
	PubKey  hexutil.Bytes
}
