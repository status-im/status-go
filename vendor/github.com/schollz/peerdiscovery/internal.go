package peerdiscovery

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// initialize returns a new peerDiscovery object which can be used to discover peers.
// The settings are optional. If any setting is not supplied, then defaults are used.
// See the Settings for more information.
func initialize(settings Settings) (p *PeerDiscovery, err error) {
	p = new(PeerDiscovery)
	p.Lock()
	defer p.Unlock()

	// initialize settings
	p.settings = settings

	// defaults
	if p.settings.Port == "" {
		p.settings.Port = "9999"
	}
	if p.settings.IPVersion == 0 {
		p.settings.IPVersion = IPv4
	}
	if p.settings.MulticastAddress == "" {
		if p.settings.IPVersion == IPv4 {
			p.settings.MulticastAddress = "239.255.255.250"
		} else {
			p.settings.MulticastAddress = "ff02::c"
		}
	}
	if len(p.settings.Payload) == 0 {
		p.settings.Payload = []byte("hi")
	}
	if p.settings.Delay == 0 {
		p.settings.Delay = 1 * time.Second
	}
	if p.settings.TimeLimit == 0 {
		p.settings.TimeLimit = 10 * time.Second
	}
	if p.settings.StopChan == nil {
		p.settings.StopChan = make(chan struct{})
	}
	p.received = make(map[string]*PeerState)
	p.settings.multicastAddressNumbers = net.ParseIP(p.settings.MulticastAddress)
	if p.settings.multicastAddressNumbers == nil {
		err = fmt.Errorf(
			"multicast address %s could not be converted to an IP",
			p.settings.MulticastAddress,
		)

		return
	}
	p.settings.portNum, err = strconv.Atoi(p.settings.Port)
	if err != nil {
		return
	}
	return
}

// filterInterfaces returns a list of valid network interfaces
func filterInterfaces(useIpv4 bool) (ifaces []net.Interface, err error) {
	allIfaces, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, iface := range allIfaces {
		// Interface must be up and either support multicast or be a loopback interface.
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagMulticast == 0 {
			continue
		}

		addrs, addrsErr := iface.Addrs()
		if addrsErr != nil {
			err = addrsErr
			return
		}

		supported := false
		for j := range addrs {
			addr, ok := addrs[j].(*net.IPNet)
			if !ok {
				continue
			}
			if addr == nil || addr.IP == nil {
				continue
			}

			// An IP can either be an IPv4 or an IPv6 address.
			// Check if the desired familiy is used.
			familiyMatches := (addr.IP.To4() != nil) == useIpv4
			if familiyMatches {
				supported = true
				break
			}
		}

		if supported {
			ifaces = append(ifaces, iface)
		}
	}

	return
}

// getLocalIPs returns the local ip address
func getLocalIPs() (ips map[string]struct{}) {
	ips = make(map[string]struct{})
	ips["localhost"] = struct{}{}
	ips["127.0.0.1"] = struct{}{}
	ips["::1"] = struct{}{}

	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, address := range addrs {
			ip, _, err := net.ParseCIDR(address.String())
			if err != nil {
				// log.Printf("Failed to parse %s: %v", address.String(), err)
				continue
			}

			ips[ip.String()+"%"+iface.Name] = struct{}{}
			ips[ip.String()] = struct{}{}
		}
	}
	return
}

func broadcast(p2 NetPacketConn, payload []byte, ifaces []net.Interface, dst net.Addr) {
	for i := range ifaces {
		if errMulticast := p2.SetMulticastInterface(&ifaces[i]); errMulticast != nil {
			continue
		}
		p2.SetMulticastTTL(2)
		if _, errMulticast := p2.WriteTo([]byte(payload), dst); errMulticast != nil {
			continue
		}
	}
}
