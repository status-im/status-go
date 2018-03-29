package main

import (
	"encoding/json"
	"time"
)

// PanicAfter throws panic() after waitSeconds, unless abort channel receives
// notification.
func PanicAfter(waitSeconds time.Duration, abort chan struct{}, desc string) {
	go func() {
		select {
		case <-abort:
			return
		case <-time.After(waitSeconds):
			panic("whatever you were doing takes toooo long: " + desc)
		}
	}()
}

// ParseJSONArray parses JSON array into Go array of string.
func ParseJSONArray(items string) ([]string, error) {
	var parsedItems []string
	err := json.Unmarshal([]byte(items), &parsedItems)
	if err != nil {
		return nil, err
	}

	return parsedItems, nil
}
