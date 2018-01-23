// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// Package dscp provides helper functions to apply DSCP / ECN / CoS flags to sockets.
package dscp

import (
	"net"
)

// ListenTCPWithTOS is similar to net.ListenTCP but with the socket configured
// to the use the given ToS (Type of Service), to specify DSCP / ECN / class
// of service flags to use for incoming connections.
func ListenTCPWithTOS(address *net.TCPAddr, tos byte) (*net.TCPListener, error) {
	return listenTCPWithTOS(address, tos)
}
