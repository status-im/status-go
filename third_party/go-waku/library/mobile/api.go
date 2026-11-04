// Package gowaku implements gomobile bindings for go-waku. Contains a set of functions that
// are exported when go-waku is exported as libraries for mobile devices
package gowaku

import (
	"github.com/waku-org/go-waku/library"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

// NewNode initializes a waku node.
// Receives a JSON string containing the configuration, and use default values for those config items not specified
// Returns an instance id
func NewNode(instanceID uint, configJSON string) string {
	instance := library.Init()
	err := library.NewNode(instance, configJSON)
	if err != nil {
		_ = library.Free(instance)
	}
	return prepareJSONResponse(instance.ID, err)
}

// Start starts the waku node
func Start(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.Start(instance)
	return makeJSONResponse(err)
}

// Stop stops a waku node
func Stop(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.Stop(instance)
	return makeJSONResponse(err)
}

// Release resources allocated to a waku node
func Free(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.Free(instance)
	return makeJSONResponse(err)
}

// IsStarted is used to determine is a node is started or not
func IsStarted(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	return prepareJSONResponse(library.IsStarted(instance), nil)
}

// PeerID is used to obtain the peer ID of the waku node
func PeerID(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	peerID, err := library.PeerID(instance)
	return prepareJSONResponse(peerID, err)
}

// ListenAddresses returns the multiaddresses the wakunode is listening to
func ListenAddresses(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	addresses, err := library.ListenAddresses(instance)
	return prepareJSONResponse(addresses, err)
}

// AddPeer adds a node multiaddress and protocol to the wakunode peerstore
func AddPeer(instanceID uint, address string, protocolID string) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	peerID, err := library.AddPeer(instance, address, protocolID)
	return prepareJSONResponse(peerID, err)
}

// Connect is used to connect to a peer at multiaddress. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func Connect(instanceID uint, address string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.Connect(instance, address, ms)
	return makeJSONResponse(err)
}

// ConnectPeerID is usedd to connect to a known peer by peerID. if ms > 0, cancel the function execution if it takes longer than N milliseconds
func ConnectPeerID(instanceID uint, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.ConnectPeerID(instance, peerID, ms)
	return makeJSONResponse(err)
}

// Disconnect closes a connection to a known peer by peerID
func Disconnect(instanceID uint, peerID string) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.Disconnect(instance, peerID)
	return makeJSONResponse(err)
}

// PeerCnt returns the number of connected peers
func PeerCnt(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	peerCnt, err := library.PeerCnt(instance)
	return prepareJSONResponse(peerCnt, err)
}

// ContentTopic creates a content topic string according to RFC 23
func ContentTopic(applicationName string, applicationVersion string, contentTopicName string, encoding string) string {
	contentTopic, _ := protocol.NewContentTopic(applicationName, applicationVersion, contentTopicName, encoding)
	return contentTopic.String()
}

// DefaultPubsubTopic returns the default pubsub topic used in waku2: /waku/2/default-waku/proto
func DefaultPubsubTopic() string {
	return library.DefaultPubsubTopic()
}

// Peers retrieves the list of peers known by the waku node
func Peers(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	peers, err := library.Peers(instance)
	return prepareJSONResponse(peers, err)
}
