package node

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func (w *WakuNode) newLocalnode(priv *ecdsa.PrivateKey) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, err
	}
	return enode.NewLocalNode(db, priv), nil
}

func (w *WakuNode) updateLocalNode(localnode *enode.LocalNode, wsAddr []ma.Multiaddr, ipAddr *net.TCPAddr, udpPort uint, wakuFlags utils.WakuEnrBitfield, advertiseAddr *net.IP, shouldAutoUpdate bool, log *zap.Logger) error {
	localnode.SetFallbackUDP(int(udpPort))
	localnode.Set(enr.WithEntry(utils.WakuENRField, wakuFlags))
	localnode.SetFallbackIP(net.IP{127, 0, 0, 1})

	if udpPort > math.MaxUint16 {
		return errors.New("invalid udp port number")
	}

	if advertiseAddr != nil {
		// An advertised address disables libp2p address updates
		// and discv5 predictions
		localnode.SetStaticIP(*advertiseAddr)
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // TODO: ipv6?
	} else if !shouldAutoUpdate {
		// We received a libp2p address update. Autoupdate is disabled
		// Using a static ip will disable endpoint prediction.
		localnode.SetStaticIP(ipAddr.IP)
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // TODO: ipv6?
	} else {
		// We received a libp2p address update, but we should still
		// allow discv5 to update the enr record. We set the localnode
		// keys manually. It's possible that the ENR record might get
		// updated automatically
		ip4 := ipAddr.IP.To4()
		ip6 := ipAddr.IP.To16()
		if ip4 != nil && !ip4.IsUnspecified() {
			localnode.Set(enr.IPv4(ip4))
			localnode.Set(enr.TCP(uint16(ipAddr.Port)))
		} else {
			localnode.Delete(enr.IPv4{})
			localnode.Delete(enr.TCP(0))
		}

		if ip6 != nil && !ip6.IsUnspecified() {
			localnode.Set(enr.IPv6(ip6))
			localnode.Set(enr.TCP6(ipAddr.Port))
		} else {
			localnode.Delete(enr.IPv6{})
			localnode.Delete(enr.TCP6(0))
		}
	}

	// Adding websocket multiaddresses
	var fieldRaw []byte

	for _, addr := range wsAddr {
		p2p, err := addr.ValueForProtocol(ma.P_P2P)
		if err != nil {
			return err
		}

		p2pAddr, err := ma.NewMultiaddr("/p2p/" + p2p)
		if err != nil {
			return fmt.Errorf("could not create p2p addr: %w", err)
		}

		maRaw := addr.Decapsulate(p2pAddr).Bytes()
		maSize := make([]byte, 2)
		binary.BigEndian.PutUint16(maSize, uint16(len(maRaw)))

		fieldRaw = append(fieldRaw, maSize...)
		fieldRaw = append(fieldRaw, maRaw...)
	}

	if len(fieldRaw) != 0 {
		localnode.Set(enr.WithEntry(utils.MultiaddrENRField, fieldRaw))
	}

	return nil
}

func isPrivate(addr candidateAddr) bool {
	return addr.ip.IP.IsPrivate()
}

func isExternal(addr candidateAddr) bool {
	return !isPrivate(addr) && !addr.ip.IP.IsLoopback() && !addr.ip.IP.IsUnspecified()
}

func isLoopback(addr candidateAddr) bool {
	return addr.ip.IP.IsLoopback()
}

func filterIP(ss []candidateAddr, fn func(candidateAddr) bool) (ret []candidateAddr) {
	for _, s := range ss {
		if fn(s) {
			ret = append(ret, s)
		}
	}
	return
}

type candidateAddr struct {
	ip    *net.TCPAddr
	maddr ma.Multiaddr
}

func extractIP(addr ma.Multiaddr) (*net.TCPAddr, error) {
	var ipStr string
	dns4, err := addr.ValueForProtocol(ma.P_DNS4)
	if err != nil {
		ipStr, err = addr.ValueForProtocol(ma.P_IP4)
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

	portStr, err := addr.ValueForProtocol(ma.P_TCP)
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

func selectMostExternalAddress(addresses []ma.Multiaddr) (ma.Multiaddr, *net.TCPAddr, error) {
	var ipAddrs []candidateAddr

	for _, addr := range addresses {
		ipAddr, err := extractIP(addr)
		if err != nil {
			continue
		}

		ipAddrs = append(ipAddrs, candidateAddr{
			ip:    ipAddr,
			maddr: addr,
		})
	}

	externalIPs := filterIP(ipAddrs, isExternal)
	if len(externalIPs) > 0 {
		return externalIPs[0].maddr, externalIPs[0].ip, nil
	}

	privateIPs := filterIP(ipAddrs, isPrivate)
	if len(privateIPs) > 0 {
		return privateIPs[0].maddr, privateIPs[0].ip, nil
	}

	loopback := filterIP(ipAddrs, isLoopback)
	if len(loopback) > 0 {
		return loopback[0].maddr, loopback[0].ip, nil
	}

	return nil, nil, errors.New("could not obtain ip address")
}

func selectWSListenAddress(addresses []ma.Multiaddr, extAddr ma.Multiaddr) ([]ma.Multiaddr, error) {
	extAddrDNS, err := extAddr.ValueForProtocol(ma.P_DNS4)
	var extAddrIP string
	if err != nil {
		extAddrIP, err = extAddr.ValueForProtocol(ma.P_IP4)
		if err != nil {
			return nil, err
		}
	}

	var result []ma.Multiaddr
	for _, addr := range addresses {
		// Filter addresses that match the extAddr
		if extAddrDNS != "" {
			dns4, err := addr.ValueForProtocol(ma.P_DNS4)
			if err != nil {
				continue
			}
			if dns4 != extAddrDNS {
				continue
			}
		} else {
			ip4, err := addr.ValueForProtocol(ma.P_IP4)
			if err != nil {
				continue
			}
			if ip4 != extAddrIP {
				continue
			}
		}

		_, err := addr.ValueForProtocol(ma.P_WS)
		if err == nil {
			result = append(result, addr)
		}

		_, err = addr.ValueForProtocol(ma.P_WSS)
		if err == nil {
			result = append(result, addr)
		}
	}

	return result, nil
}

func (w *WakuNode) setupENR(ctx context.Context, addrs []ma.Multiaddr) error {
	extAddr, ipAddr, err := selectMostExternalAddress(addrs)
	if err != nil {
		w.log.Error("obtaining external address", zap.Error(err))
		return err
	}

	wsAddresses, err := selectWSListenAddress(addrs, extAddr)
	if err != nil {
		w.log.Error("obtaining websocket addresses", zap.Error(err))
		return err
	}

	err = w.updateLocalNode(w.localNode, wsAddresses, ipAddr, w.opts.udpPort, w.wakuFlag, w.opts.advertiseAddr, w.opts.discV5autoUpdate, w.log)
	if err != nil {
		w.log.Error("obtaining ENR record from multiaddress", logging.MultiAddrs("multiaddr", extAddr), zap.Error(err))
		return err
	}

	return nil
}
