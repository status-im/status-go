package utils

import (
	"net"
	"strings"
)

func IsIPv4(str string) bool {
	ip := net.ParseIP(str)
	return ip != nil && !strings.Contains(str, ":")
}

func IsIPv6(str string) bool {
	ip := net.ParseIP(str)
	return ip != nil && strings.Contains(str, ":")
}
