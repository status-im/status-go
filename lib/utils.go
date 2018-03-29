package main

import (
	"encoding/json"
)

// ParseJSONArray parses JSON array into Go array of string.
func ParseJSONArray(items string) ([]string, error) {
	var parsedItems []string
	err := json.Unmarshal([]byte(items), &parsedItems)
	if err != nil {
		return nil, err
	}

	return parsedItems, nil
}
