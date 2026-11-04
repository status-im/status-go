package utils

import (
	"errors"
	"net"
	"strconv"

	"github.com/multiformats/go-multiaddr"
)

func ExtractIPAddressForENR(addr multiaddr.Multiaddr) (*net.TCPAddr, error) {
	// It's a p2p-circuit address. We shouldnt use these
	// for building the ENR record default keys
	_, err := addr.ValueForProtocol(multiaddr.P_CIRCUIT)
	if err == nil {
		return nil, errors.New("can't use IP address from a p2p-circuit address")
	}

	// ws and wss addresses are handled by the multiaddr key
	// they shouldnt be used for building the ENR record default keys
	_, err = addr.ValueForProtocol(multiaddr.P_WS)
	if err == nil {
		return nil, errors.New("can't use IP address from a ws address")
	}
	_, err = addr.ValueForProtocol(multiaddr.P_WSS)
	if err == nil {
		return nil, errors.New("can't use IP address from a wss address")
	}

	var ipStr string
	dns4, err := addr.ValueForProtocol(multiaddr.P_DNS4)
	if err != nil {
		ipStr, err = addr.ValueForProtocol(multiaddr.P_IP4)
		if err != nil {
			return nil, err
		}
	} else {
		netIP, err := net.ResolveIPAddr("ip4", dns4)
		if err != nil {
			return nil, err
		}
		ipStr = netIP.String()
	}

	portStr, err := addr.ValueForProtocol(multiaddr.P_TCP)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	return &net.TCPAddr{
		IP:   net.ParseIP(ipStr),
		Port: port,
	}, nil
}

func SelectMostExternalAddress(addresses []multiaddr.Multiaddr) (*net.TCPAddr, error) {
	var ipAddrs []*net.TCPAddr
	for _, addr := range addresses {
		ipAddr, err := ExtractIPAddressForENR(addr)
		if err != nil {
			continue
		}
		ipAddrs = append(ipAddrs, ipAddr)
	}

	externalIPs := filterIP(ipAddrs, isExternal)
	if len(externalIPs) > 0 {
		return externalIPs[0], nil
	}

	privateIPs := filterIP(ipAddrs, isPrivate)
	if len(privateIPs) > 0 {
		return privateIPs[0], nil
	}

	loopback := filterIP(ipAddrs, isLoopback)
	if len(loopback) > 0 {
		return loopback[0], nil
	}

	return nil, errors.New("could not obtain ip address")
}

func isPrivate(addr *net.TCPAddr) bool {
	return addr.IP.IsPrivate()
}

func isExternal(addr *net.TCPAddr) bool {
	return !isPrivate(addr) && !addr.IP.IsLoopback() && !addr.IP.IsUnspecified()
}

func isLoopback(addr *net.TCPAddr) bool {
	return addr.IP.IsLoopback()
}

func filterIP(ss []*net.TCPAddr, fn func(*net.TCPAddr) bool) (ret []*net.TCPAddr) {
	for _, s := range ss {
		if fn(s) {
			ret = append(ret, s)
		}
	}
	return
}
