package main

/*
#include <stddef.h>
#include <stdbool.h>
extern bool GethServiceSignalEvent( const char *jsonEvent );
*/
import "C"
import (
    "encoding/json"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/whisper"
)

var(
    whisperFilters []int
)


func onWhisperMessage(message *whisper.Message) {
    event := GethEvent{
        Type: "whisper",
        Event: WhisperMessageEvent{
            Payload: string(message.Payload),
            From:    common.ToHex(crypto.FromECDSAPub(message.Recover())),
            To:      common.ToHex(crypto.FromECDSAPub(message.To)),
            Sent:    message.Sent.Unix(),
            TTL:     int64(message.TTL / time.Second),
            Hash:    common.ToHex(message.Hash.Bytes()),
        },
    }
    body, _ := json.Marshal(&event)
    C.GethServiceSignalEvent(C.CString(string(body)))
}

func doAddWhisperFilter(args whisper.NewFilterArgs) int {
    var id int
    filter := whisper.Filter{
        To:     crypto.ToECDSAPub(common.FromHex(args.To)),
        From:   crypto.ToECDSAPub(common.FromHex(args.From)),
        Topics: whisper.NewFilterTopics(args.Topics...),
        Fn: onWhisperMessage,
    }

    id = whisperService.Watch(filter)
    whisperFilters = append(whisperFilters, id)
    return id
}

func doRemoveWhisperFilter(idFilter int) {
    whisperService.Unwatch(idFilter)
}

func doClearWhisperFilters() {
    for _, idFilter := range whisperFilters {
        doRemoveWhisperFilter(idFilter)
    }
    whisperFilters = nil
}