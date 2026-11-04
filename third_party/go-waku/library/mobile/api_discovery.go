package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// DNSDiscovery executes dns discovery on an url and returns a list of nodes
func DNSDiscovery(url string, nameserver string, ms int) string {
	response, err := library.DNSDiscovery(url, nameserver, ms)
	return prepareJSONResponse(response, err)
}

// StartDiscoveryV5 starts discv5 discovery
func StartDiscoveryV5(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.StartDiscoveryV5(instance)
	return makeJSONResponse(err)
}

// StopDiscoveryV5 stops discv5 discovery
func StopDiscoveryV5(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.StopDiscoveryV5(instance)
	return makeJSONResponse(err)
}

// SetBootnodes is used to update the bootnodes receiving a JSON array of ENRs
func SetBootnodes(instanceID uint, bootnodes string) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.SetBootnodes(instance, bootnodes)
	return makeJSONResponse(err)
}
