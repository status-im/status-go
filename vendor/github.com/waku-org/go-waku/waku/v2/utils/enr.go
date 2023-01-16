package utils

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// WakuENRField is the name of the ENR field that contains information about which protocols are supported by the node
const WakuENRField = "waku2"

// MultiaddrENRField is the name of the ENR field that will contain multiaddresses that cannot be described using the
// already available ENR fields (i.e. in the case of websocket connections)
const MultiaddrENRField = "multiaddrs"

// WakuEnrBitfield is a8-bit flag field to indicate Waku capabilities. Only the 4 LSBs are currently defined according to RFC31 (https://rfc.vac.dev/spec/31/).
type WakuEnrBitfield = uint8

// NewWakuEnrBitfield creates a WakuEnrBitField whose value will depend on which protocols are enabled in the node
func NewWakuEnrBitfield(lightpush, filter, store, relay bool) WakuEnrBitfield {
	var v uint8 = 0

	if lightpush {
		v |= (1 << 3)
	}

	if filter {
		v |= (1 << 2)
	}

	if store {
		v |= (1 << 1)
	}

	if relay {
		v |= (1 << 0)
	}

	return v
}

// GetENRandIP returns a enr Node and TCP address obtained from a multiaddress. priv key and protocols supported
func GetENRandIP(addrs []multiaddr.Multiaddr, wakuFlags WakuEnrBitfield, privK *ecdsa.PrivateKey) (*enode.Node, error) {
	r := &enr.Record{}
	for _, addr := range addrs {
		storeInMultiaddrsKey := false
		var multiaddrItems []multiaddr.Multiaddr
		_, err := addr.ValueForProtocol(multiaddr.P_WS)
		if err == nil {
			storeInMultiaddrsKey = true
			multiaddrItems = append(multiaddrItems, addr)
		}

		_, err = addr.ValueForProtocol(multiaddr.P_WSS)
		if err == nil {
			storeInMultiaddrsKey = true
			multiaddrItems = append(multiaddrItems, addr)
		}

		if !storeInMultiaddrsKey {
			var ip string
			dns4, err := addr.ValueForProtocol(multiaddr.P_DNS4)
			if err != nil {
				ip, err = addr.ValueForProtocol(multiaddr.P_IP4)
				if err != nil {
					return nil, err
				}
			} else {
				netIP, err := net.ResolveIPAddr("ip4", dns4)
				if err != nil {
					return nil, err
				}
				ip = netIP.String()
			}

			portStr, err := addr.ValueForProtocol(multiaddr.P_TCP)
			if err != nil {
				return nil, err
			}

			port, err := strconv.Atoi(portStr)
			if err != nil {
				return nil, err
			}

			if port > 0 && port <= math.MaxUint16 {
				r.Set(enr.TCP(uint16(port))) // lgtm [go/incorrect-integer-conversion]
			} else {
				return nil, fmt.Errorf("could not set port %d", port)
			}

			r.Set(enr.IP(net.ParseIP(ip)))
		} else {
			p2p, err := addr.ValueForProtocol(multiaddr.P_P2P)
			if err != nil {
				return nil, err
			}

			p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + p2p)
			if err != nil {
				return nil, fmt.Errorf("could not create p2p addr: %w", err)
			}

			var fieldRaw []byte
			for _, ma := range multiaddrItems {
				maRaw := ma.Decapsulate(p2pAddr).Bytes()
				maSize := make([]byte, 2)
				binary.BigEndian.PutUint16(maSize, uint16(len(maRaw)))

				fieldRaw = append(fieldRaw, maSize...)
				fieldRaw = append(fieldRaw, maRaw...)
			}

			if len(fieldRaw) != 0 {
				r.Set(enr.WithEntry(MultiaddrENRField, fieldRaw))
			}
		}
	}

	r.Set(enr.WithEntry(WakuENRField, wakuFlags))
	err := enode.SignV4(r, privK)
	if err != nil {
		return nil, err
	}

	node, err := enode.New(enode.ValidSchemes, r)

	return node, err
}

// EnodeToMultiaddress converts an enode into a multiaddress
func enodeToMultiAddr(node *enode.Node) (multiaddr.Multiaddr, error) {
	pubKey := EcdsaPubKeyToSecp256k1PublicKey(node.Pubkey())
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	return multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", node.IP(), node.TCP(), peerID))
}

// Multiaddress is used to extract all the multiaddresses that are part of a ENR record
func Multiaddress(node *enode.Node) ([]multiaddr.Multiaddr, error) {
	pubKey := EcdsaPubKeyToSecp256k1PublicKey(node.Pubkey())
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	var result []multiaddr.Multiaddr

	addr, err := enodeToMultiAddr(node)
	if err != nil {
		return nil, err
	}
	result = append(result, addr)

	var multiaddrRaw []byte
	if err := node.Record().Load(enr.WithEntry(MultiaddrENRField, &multiaddrRaw)); err != nil {
		if !enr.IsNotFound(err) {
			return nil, err
		} else {
			// No multiaddr entry on enr
			return result, nil
		}
	}

	hostInfo, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID.Pretty()))
	if err != nil {
		return nil, err
	}

	offset := 0
	for {
		maSize := binary.BigEndian.Uint16(multiaddrRaw[offset : offset+2])
		if len(multiaddrRaw) < offset+2+int(maSize) {
			return nil, errors.New("invalid multiaddress field length")
		}
		maRaw := multiaddrRaw[offset+2 : offset+2+int(maSize)]
		addr, err := multiaddr.NewMultiaddrBytes(maRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid multiaddress field length")
		}

		result = append(result, addr.Encapsulate(hostInfo))

		offset += 2 + int(maSize)
		if offset >= len(multiaddrRaw) {
			break
		}
	}

	return result, nil
}

// EnodeToPeerInfo extracts the peer ID and multiaddresses defined in an ENR
func EnodeToPeerInfo(node *enode.Node) (*peer.AddrInfo, error) {
	addresses, err := Multiaddress(node)
	if err != nil {
		return nil, err
	}

	res, err := peer.AddrInfosFromP2pAddrs(addresses...)
	if err != nil {
		return nil, err
	}

	return &res[0], nil
}
