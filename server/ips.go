package server

import (
	"net"
)

var (
	DefaultIP = net.IP{127, 0, 0, 1}
	Localhost = "Localhost"
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
