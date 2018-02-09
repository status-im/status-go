package api

import (
	"fmt"
)

// ConnectionState represents device connection state and type,
// as reported by mobile framework.
//
// Zero value represents default assumption about network
// connection until first update — online, on cellular and not expensive.
type ConnectionState struct {
	Offline   bool           `json:"offline"`
	Type      ConnectionType `json:"type"`
	Expensive bool           `json:"expensive"`
}

// ConnectionType represents description of available
// connection types as reported by React Native (see
// https://facebook.github.io/react-native/docs/netinfo.html)
// We're interested mainly in 'wifi' and 'cellular', but
// other types are also may be used.
type ConnectionType byte

// NewConnectionType creates new ConnectionType from string.
func NewConnectionType(s string) ConnectionType {
	switch s {
	case "cellular":
		return ConnectionCellular
	case "wifi":
		return ConnectionWifi
	}

	return ConnectionUnknown
}

const (
	ConnectionCellular = iota // default value, LTE, 4G, 3G, EDGE, etc.
	ConnectionWifi            // WIFI or iOS simulator
	ConnectionUnknown
)

// String formats ConnectionState for logs. Implements Stringer.
func (c ConnectionState) String() string {
	if c.Offline {
		return "offline"
	}

	var typ string
	switch c.Type {
	case ConnectionWifi:
		typ = "wifi"
	case ConnectionCellular:
		typ = "cellular"
	default:
		typ = "unknown"
	}

	if c.Expensive {
		return fmt.Sprintf("%s (expensive)", typ)
	}

	return typ
}
