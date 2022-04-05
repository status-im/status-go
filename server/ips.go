package server

import (
	"net"
)

func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "255.255.255.255:8080")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
