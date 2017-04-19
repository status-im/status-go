package geth

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv2"
)

func onWhisperMessage(message *whisper.Message) {
	SendSignal(SignalEnvelope{
		Type: "whisper",
		Event: WhisperMessageEvent{
			Payload: string(message.Payload),
			From:    common.ToHex(crypto.FromECDSAPub(message.Recover())),
			To:      common.ToHex(crypto.FromECDSAPub(message.To)),
			Sent:    message.Sent.Unix(),
			TTL:     int64(message.TTL / time.Second),
			Hash:    common.ToHex(message.Hash.Bytes()),
		},
	})
}
