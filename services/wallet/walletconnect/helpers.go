package walletconnect

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// func sendResponseEvent(eventFeed *event.Feed, eventType walletevent.EventType, payloadObj interface{}, resErr error) {
// 	payload, err := json.Marshal(payloadObj)
// 	if err != nil {
// 		log.Error("Error marshaling WC response: %v; result error: %w", err, resErr)
// 	} else {
// 		err = resErr
// 	}

// 	event := walletevent.Event{
// 		Type:    eventType,
// 		Message: string(payload),
// 	}

// 	sentCount := eventFeed.Send(event)

// 	log.Debug("wallet.api.wc RESPONSE", "eventType", eventType, "error", err, "payload.len", len(payload), "sentCount", sentCount)
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

// JSONProxyType provides a generic way of changing the JSON value before unmarshalling it into the target.
// transform function is called before unmarshalling.
type JSONProxyType struct {
	target    interface{}
	transform func([]byte) ([]byte, error)
}

func (b *JSONProxyType) UnmarshalJSON(input []byte) error {
	if b.transform == nil {
		return errors.New("transform function is not set")
	}

	output, err := b.transform(input)
	if err != nil {
		return err
	}

	return json.Unmarshal(output, b.target)
}
