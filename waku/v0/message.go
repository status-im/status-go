package v0

import (
	"bytes"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/status-go/waku/common"
)

// MultiVersionResponse allows to decode response into chosen version.
type MultiVersionResponse struct {
	Version  uint
	Response rlp.RawValue
}

// DecodeResponse1 decodes response into first version of the messages response.
func (m MultiVersionResponse) DecodeResponse1() (resp common.MessagesResponse, err error) {
	return resp, rlp.DecodeBytes(m.Response, &resp)
}

// Version1MessageResponse first version of the message response.
type Version1MessageResponse struct {
	Version  uint
	Response common.MessagesResponse
}

// NewMessagesResponse returns instance of the version messages response.
func NewMessagesResponse(batch gethcommon.Hash, errors []common.EnvelopeError) Version1MessageResponse {
	return Version1MessageResponse{
		Version: 1,
		Response: common.MessagesResponse{
			Hash:   batch,
			Errors: errors,
		},
	}
}

func sendBundle(rw p2p.MsgWriter, bundle []*common.Envelope) (rst gethcommon.Hash, err error) {
	data, err := rlp.EncodeToBytes(bundle)
	if err != nil {
		return
	}
	err = rw.WriteMsg(p2p.Msg{
		Code:    MessagesCode,
		Size:    uint32(len(data)),
		Payload: bytes.NewBuffer(data),
	})
	if err != nil {
		return
	}
	return crypto.Keccak256Hash(data), nil
}
