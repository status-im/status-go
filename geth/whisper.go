package geth

import (
	"time"

	"github.com/teslapatrick/go-ethereum/common"
	"github.com/teslapatrick/go-ethereum/crypto"
	whisper "github.com/teslapatrick/go-ethereum/whisper/whisperv2"
)

var (
	whisperFilters []int
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

func AddWhisperFilter(args whisper.NewFilterArgs) int {
	whisperService, err := NodeManagerInstance().WhisperService()
	if err != nil {
		return -1
	}

	filter := whisper.Filter{
		To:     crypto.ToECDSAPub(common.FromHex(args.To)),
		From:   crypto.ToECDSAPub(common.FromHex(args.From)),
		Topics: whisper.NewFilterTopics(args.Topics...),
		Fn:     onWhisperMessage,
	}

	id := whisperService.Watch(filter)
	whisperFilters = append(whisperFilters, id)
	return id
}

func RemoveWhisperFilter(idFilter int) {
	whisperService, err := NodeManagerInstance().WhisperService()
	if err != nil {
		return
	}
	whisperService.Unwatch(idFilter)
}

func ClearWhisperFilters() {
	for _, idFilter := range whisperFilters {
		RemoveWhisperFilter(idFilter)
	}
	whisperFilters = nil
}
