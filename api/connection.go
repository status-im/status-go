package api

import (
	"fmt"
)

// connectionState represents device connection state and type,
// as reported by mobile framework.
//
// Zero value represents default assumption about network (online and unknown type).
type connectionState struct {
	Offline   bool           `json:"offline"`
	Type      connectionType `json:"type"`
	Expensive bool           `json:"expensive"`
}

// connectionType represents description of available
// connection types as reported by React Native (see
// https://facebook.github.io/react-native/docs/netinfo.html)
// We're interested mainly in 'wifi' and 'cellular', but
// other types are also may be used.
type connectionType byte

const (
	offline  = "offline"
	wifi     = "wifi"
	cellular = "cellular"
	unknown  = "unknown"
	none     = "none"
)

// newConnectionType creates new connectionType from string.
func newConnectionType(s string) connectionType {
	switch s {
	case cellular:
		return connectionCellular
	case wifi:
		return connectionWifi
	}

	return connectionUnknown
}

// ConnectionType constants
const (
	connectionUnknown  connectionType = iota
	connectionCellular                // cellular, LTE, 4G, 3G, EDGE, etc.
	connectionWifi                    // WIFI or iOS simulator
)

// string formats ConnectionState for logs. Implements Stringer.
func (c connectionState) String() string {
	if c.Offline {
		return offline
	}

	var typ string
	switch c.Type {
	case connectionWifi:
		typ = wifi
	case connectionCellular:
		typ = cellular
	default:
		typ = unknown
	}

	if c.Expensive {
		return fmt.Sprintf("%s (expensive)", typ)
	}

	return typ
}
