package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// StoreQuery is used to retrieve historic messages using waku store protocol.
func StoreQuery(instanceID uint, queryJSON string, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	response, err := library.StoreQuery(instance, queryJSON, peerID, ms)
	return prepareJSONResponse(response, err)
}

// StoreLocalQuery is used to retrieve historic messages stored in the localDB using waku store protocol.
func StoreLocalQuery(instanceID uint, queryJSON string) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	response, err := library.StoreLocalQuery(instance, queryJSON)
	return prepareJSONResponse(response, err)
}
