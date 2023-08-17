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

// addrToIPNet casts addr to IPNet.
// Returns nil if addr is not of IPNet type.
func addrToIPNet(addr net.Addr) *net.IPNet {
	switch v := addr.(type) {
	case *net.IPNet:
		return v
	default:
		return nil
	}
}

// FilterAddressesForPairingServer filters private unicast addresses.
// ips is a 2-dimensional array, where each sub-array is a list of IP
// addresses for a single network interface.
func FilterAddressesForPairingServer(ips [][]net.IP) []net.IP {
	var result []net.IP

	for _, niIps := range ips {
		var ipv4, ipv6 []net.IP

		for _, ip := range niIps {

			// Only take private global unicast addrs
			if !ip.IsGlobalUnicast() || !ip.IsPrivate() {
				continue
			}

			if v := ip.To4(); v != nil {
				ipv4 = append(ipv4, ip)
			} else {
				ipv6 = append(ipv6, ip)
			}
		}

		// Prefer IPv4 over IPv6 for shorter connection string
		if len(ipv4) == 0 {
			result = append(result, ipv6...)
		} else {
			result = append(result, ipv4...)
		}
	}

	return result
}

// GetLocalAddresses returns an array of all addresses
// of all available network interfaces.
func GetLocalAddresses() ([][]net.IP, error) {
	nis, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips [][]net.IP

	for _, ni := range nis {
		var niIps []net.IP

		addrs, err := ni.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {

			var ip net.IP
			if ipNet := addrToIPNet(addr); ipNet == nil {
				continue
			} else {
				ip = ipNet.IP
			}

			niIps = append(niIps, ip)
		}

		if len(niIps) > 0 {
			ips = append(ips, niIps)
		}
	}

	return ips, nil
}

// GetLocalAddressesForPairingServer is a high-level func
// that returns a list of addresses to be used by local pairing server.
func GetLocalAddressesForPairingServer() ([]net.IP, error) {
	ips, err := GetLocalAddresses()
	if err != nil {
		return nil, err
	}
	return FilterAddressesForPairingServer(ips), nil
}

// FindReachableAddresses returns a filtered remoteIps array,
// in which each IP matches one or more of given localNets.
func FindReachableAddresses(remoteIps []net.IP, localNets []net.IPNet) []net.IP {
	var result []net.IP
	for _, localNet := range localNets {
		for _, remoteIP := range remoteIps {
			if localNet.Contains(remoteIP) {
				result = append(result, remoteIP)
			}
		}
	}
	return result
}

// GetAllAvailableNetworks collects all networks
// from available network interfaces.
func GetAllAvailableNetworks() ([]net.IPNet, error) {
	var localNets []net.IPNet

	nis, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, ni := range nis {
		addrs, err := ni.Addrs()
		if err != nil {
			return nil, err
		}

		for _, localAddr := range addrs {
			localNets = append(localNets, *addrToIPNet(localAddr))
		}
	}
	return localNets, nil
}

// FindReachableAddressesForPairingClient is a high-level func
// that returns a reachable server's address to be used by local pairing client.
func FindReachableAddressesForPairingClient(serverIps []net.IP) ([]net.IP, error) {
	nets, err := GetAllAvailableNetworks()
	if err != nil {
		return nil, err
	}
	return FindReachableAddresses(serverIps, nets), nil
}
