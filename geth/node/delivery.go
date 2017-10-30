package node

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethmessage "github.com/ethereum/go-ethereum/common/message"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
)

// LogDeliveryService implements the whisper.DeliveryServer which logs out
// stats of whisper.MessageState to the log.
type LogDeliveryService struct{}

// SendState logs incoming whisper.MesssageState into the log.
func (ld LogDeliveryService) SendState(state whisper.MessageState) {
	var stat common.MessageState
	var protocol string
	var payload []byte
	var from, to string

	if state.IsP2P {
		protocol = "P2P"
	} else {
		protocol = "RPC"
	}

	switch state.Direction {
	case gethmessage.IncomingMessage:
		payload = state.Received.Payload

		if state.Received.Src != nil {
			from = gethcommon.ToHex(crypto.FromECDSAPub(state.Received.Src))
		}

		if state.Received.Dst != nil {
			to = gethcommon.ToHex(crypto.FromECDSAPub(state.Received.Dst))
		}

	case gethmessage.OutgoingMessage:
		from = state.Source.Sig

		if len(state.Source.PublicKey) == 0 {
			to = string(state.Source.PublicKey)
		} else {
			to = state.Source.TargetPeer
		}
	}

	stat.Protocol = protocol
	stat.Payload = payload
	stat.FromDevice = from
	stat.ToDevice = to
	stat.Source = state.Source
	stat.RejectionReason = state.Reason
	stat.Envelope = state.Envelope.Data
	stat.Status = state.Status.String()
	stat.Type = state.Direction.String()
	stat.Hash = state.Envelope.Hash().String()
	stat.TimeSent = state.Envelope.Expiry - state.Envelope.TTL

	statdata, err := json.Marshal(stat)
	if err != nil {
		log.Warn("failed to marshal common.MessageStat", "err", err)
		return
	}

	encodedStat := base64.StdEncoding.EncodeToString(statdata)
	log.Info(fmt.Sprintf("%s : %s : %s : %s : %+s", params.MessageStatHeader, stat.Protocol, stat.Type, stat.Status, encodedStat))
}
