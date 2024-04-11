// +build !windows

package tcp

import (
	"context"
	"fmt"
	"net"
	"runtime"

	"golang.org/x/sys/unix"
)


// parseSockAddr resolves given addr to unix.Sockaddr
func parseSockAddr(addr string) (unix.Sockaddr, error) {
	const bootstrapDNS = "8.8.8.8:53"
	var dialer net.Dialer
	net.DefaultResolver = &net.Resolver{
		PreferGo: false,
		Dial: func(context context.Context, _, _ string) (net.Conn, error) {
			conn, err := dialer.DialContext(context, "udp", bootstrapDNS)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
	fmt.Println("Attempting to parseSckAddr ->", addr)
	tAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		fmt.Println("Error in net.ResolveTCPAddr ->", err)
		return nil, err
	}
	var addr4 [4]byte
	if tAddr.IP != nil {
		copy(addr4[:], tAddr.IP.To4()) // copy last 4 bytes of slice to array
	}
	fmt.Println("Resolved Port is ->", tAddr.Port, "Resolved Addr is -> ", addr4)
	return &unix.SockaddrInet4{Port: tAddr.Port, Addr: addr4}, nil
}

// connect calls the connect syscall with error handled.
func connect(fd int, addr unix.Sockaddr) (success bool, err error) {
	// Connect() sends the actual SYN
	switch serr := unix.Connect(fd, addr); serr {
	case unix.EALREADY, unix.EINPROGRESS, unix.EINTR:
		// Connection could not be made immediately but asynchronously.
		success = false
		err = nil
	case nil, unix.EISCONN:
		// The specified socket is already connected.
		success = true
		err = nil
	case unix.EINVAL:
		// On Solaris we can see EINVAL if the socket has
		// already been accepted and closed by the server.
		// Treat this as a successful connection--writes to
		// the socket will see EOF.  For details and a test
		// case in C see https://golang.org/issue/6828.
		if runtime.GOOS == "solaris" {
			success = true
			err = nil
		} else {
			// error must be reported
			success = false
			err = serr
		}
	default:
		// Connect error.
		success = false
		err = serr
	}
	return success, err
}
