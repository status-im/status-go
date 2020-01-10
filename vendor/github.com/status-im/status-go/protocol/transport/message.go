package transport

import "github.com/status-im/status-go/eth-node/types"

const defaultPowTime = 1

func DefaultMessage() types.NewMessage {
	msg := types.NewMessage{}

	msg.TTL = 10
	msg.PowTarget = 0.002
	msg.PowTime = defaultPowTime

	return msg
}
