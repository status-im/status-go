package walletconnect

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// TODO #12434: respond async
// func sendResponseEvent(eventFeed *event.Feed, eventType walletevent.EventType, payloadObj interface{}, resErr error) {
// 	payload, err := json.Marshal(payloadObj)
// 	if err != nil {
// 		log.Error("Error marshaling WC response: %v; result error: %w", err, resErr)
// 	} else {
// 		err = resErr
// 	}

// 	log.Debug("wallet.api.wc RESPONSE", "eventType", eventType, "error", err, "payload.len", len(payload))

// 	event := walletevent.Event{
// 		Type:    eventType,
// 		Message: string(payload),
// 	}

// 	eventFeed.Send(event)
// }

func parseCaip2ChainID(str string) (uint64, error) {
	caip2 := strings.Split(str, ":")
	if len(caip2) != 2 {
		return 0, errors.New("CAIP-2 string is not valid")
	}

	chainIDStr := caip2[1]
	chainID, err := strconv.ParseUint(chainIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("CAIP-2 second value not valid Chain ID: %w", err)
	}
	return chainID, nil
}
