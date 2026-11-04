package library

import "encoding/json"

func marshalJSON(result interface{}) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
