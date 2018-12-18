// TODO: These types should be defined using protobuf, but protoc can only emit []byte instead of hexutil.Bytes,
// which causes issues when marshalong to JSON on the react side. Let's do that once the chat protocol is moved to the go repo.

package chat

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// SendPublicMessageRPC represents the RPC payload for the SendPublicMessage RPC method
type SendPublicMessageRPC struct {
	Sig     string
	Chat    string
	Payload hexutil.Bytes
}

// SendDirectMessageRPC represents the RPC payload for the SendDirectMessage RPC method
type SendDirectMessageRPC struct {
	Sig     string
	Chat    string
	Payload hexutil.Bytes
	PubKey  hexutil.Bytes
}

// SendGroupMessageRPC represents the RPC payload for the SendGroupMessage RPC method
type SendGroupMessageRPC struct {
	Sig     string
	Payload hexutil.Bytes
	PubKeys []hexutil.Bytes
}
