package settings

import (
	"encoding/json"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
)

func BoolHandler(value interface{}) (interface{}, error) {
	_, ok := value.(bool)
	if !ok {
		return value, errors.ErrInvalidConfig
	}

	return value, nil
}

func JSONBlobHandler(value interface{}) (interface{}, error) {
	return &sqlite.JSONBlob{Data: value}, nil
}

func AddressHandler(value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if ok {
		value = types.HexToAddress(str)
	} else {
		return value, errors.ErrInvalidConfig
	}
	return value, nil
}

func NodeConfigHandler(value interface{}) (interface{}, error){
	jsonString, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var nodeConfig params.NodeConfig
	err = json.Unmarshal(jsonString, &nodeConfig)
	if err != nil {
		return nil, err
	}

	return nodeConfig, nil
}
