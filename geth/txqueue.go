package geth

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent( const char *jsonEvent );
*/
import "C"

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
)

const (
	EventTransactionQueued = "transaction.queued"
)

func onSendTransactionRequest(queuedTx status.QueuedTx) {
	event := GethEvent{
		Type: EventTransactionQueued,
		Event: SendTransactionEvent{
			Id:   string(queuedTx.Id),
			Args: queuedTx.Args,
		},
	}

	body, _ := json.Marshal(&event)
	C.StatusServiceSignalEvent(C.CString(string(body)))
}

func CompleteTransaction(id, password string) (common.Hash, error) {
	lightEthereum, err := GetNodeManager().LightEthereumService()
	if err != nil {
		return common.Hash{}, err
	}

	backend := lightEthereum.StatusBackend

	return backend.CompleteQueuedTransaction(status.QueuedTxId(id), password)
}
