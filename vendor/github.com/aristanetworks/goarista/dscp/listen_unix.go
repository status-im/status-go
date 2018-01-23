// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package dscp

import (
	"net"
	"os"
	"reflect"

	"golang.org/x/sys/unix"
)

func listenTCPWithTOS(address *net.TCPAddr, tos byte) (*net.TCPListener, error) {
	lsnr, err := net.ListenTCP("tcp", address)
	if err != nil {
		return nil, err
	}
	// This works for the UNIX implementation of netFD, i.e. not on Windows and Plan9.
	// This kludge is needed until https://github.com/golang/go/issues/9661 is fixed.
	fd := int(reflect.ValueOf(lsnr).Elem().FieldByName("fd").Elem().FieldByName("sysfd").Int())
	var proto, optname int
	if address.IP.To4() != nil {
		proto = unix.IPPROTO_IP
		optname = unix.IP_TOS
	} else {
		proto = unix.IPPROTO_IPV6
		optname = unix.IPV6_TCLASS
	}
	err = unix.SetsockoptInt(fd, proto, optname, int(tos))
	if err != nil {
		lsnr.Close()
		return nil, os.NewSyscallError("setsockopt", err)
	}

	return lsnr, nil
}
