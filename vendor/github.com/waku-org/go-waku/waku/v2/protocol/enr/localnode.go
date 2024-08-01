package enr

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"net"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

func NewLocalnode(priv *ecdsa.PrivateKey) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, err
	}
	return enode.NewLocalNode(db, priv), nil
}

type ENROption func(*enode.LocalNode) error

func WithMultiaddress(multiaddrs ...multiaddr.Multiaddr) ENROption {
	return func(localnode *enode.LocalNode) (err error) {
		// Randomly shuffle multiaddresses
		rand.Shuffle(len(multiaddrs), func(i, j int) { multiaddrs[i], multiaddrs[j] = multiaddrs[j], multiaddrs[i] })

		// Testing how many multiaddresses we can write before we exceed the limit
		// By simulating what the localnode does when signing the enr, but without
		// causing a panic

		privk, err := crypto.GenerateKey()
		if err != nil {
			return err
		}

		// Adding extra multiaddresses. Should probably not exceed the enr max size of 300bytes
		failedOnceWritingENR := false
		couldWriteENRatLeastOnce := false
		successIdx := -1
		for i := len(multiaddrs); i > 0; i-- {
			cpy := localnode.Node().Record() // Record() creates a copy for the current iteration
			// Copy all the entries that might not have been written in the ENR record due to the
			// async nature of localnode.Set
			for _, entry := range localnode.Entries() {
				cpy.Set(entry)
			}
			cpy.Set(enr.WithEntry(MultiaddrENRField, marshalMultiaddress(multiaddrs[0:i])))
			cpy.SetSeq(localnode.Seq() + 1)
			err = enode.SignV4(cpy, privk)
			if err == nil {
				couldWriteENRatLeastOnce = true
				successIdx = i
				break
			}
			failedOnceWritingENR = true
		}

		if failedOnceWritingENR && couldWriteENRatLeastOnce {
			// Could write a subset of multiaddresses but not all
			writeMultiaddressField(localnode, multiaddrs[0:successIdx])
		}

		return nil
	}
}

func WithCapabilities(lightpush, filter, store, relay bool) ENROption {
	return func(localnode *enode.LocalNode) (err error) {
		wakuflags := NewWakuEnrBitfield(lightpush, filter, store, relay)
		return WithWakuBitfield(wakuflags)(localnode)
	}
}

func WithWakuBitfield(flags WakuEnrBitfield) ENROption {
	return func(localnode *enode.LocalNode) (err error) {
		localnode.Set(enr.WithEntry(WakuENRField, flags))
		return nil
	}
}

func WithIP(ipAddr *net.TCPAddr) ENROption {
	return func(localnode *enode.LocalNode) (err error) {
		if ipAddr.Port == 0 {
			return ErrNoPortAvailable
		}

		localnode.SetStaticIP(ipAddr.IP)
		localnode.Set(enr.TCP(uint16(ipAddr.Port))) // TODO: ipv6?
		return nil
	}
}

func WithUDPPort(udpPort uint) ENROption {
	return func(localnode *enode.LocalNode) (err error) {
		if udpPort == 0 {
			return nil
		}

		if udpPort > math.MaxUint16 {
			return errors.New("invalid udp port number")
		}
		localnode.SetFallbackUDP(int(udpPort))
		return nil
	}
}

func Update(logger *zap.Logger, localnode *enode.LocalNode, enrOptions ...ENROption) error {
	for _, opt := range enrOptions {
		err := opt(localnode)
		if err != nil {
			if errors.Is(err, ErrNoPortAvailable) {
				logger.Warn("no tcp port available. ENR will not contain tcp key")
			} else {
				return err
			}
		}
	}
	return nil
}

func marshalMultiaddress(addrAggr []multiaddr.Multiaddr) []byte {
	var fieldRaw []byte
	for _, addr := range addrAggr {
		maRaw := addr.Bytes()
		maSize := make([]byte, 2)
		binary.BigEndian.PutUint16(maSize, uint16(len(maRaw)))

		fieldRaw = append(fieldRaw, maSize...)
		fieldRaw = append(fieldRaw, maRaw...)
	}
	return fieldRaw
}

func writeMultiaddressField(localnode *enode.LocalNode, addrAggr []multiaddr.Multiaddr) {
	fieldRaw := marshalMultiaddress(addrAggr)
	localnode.Set(enr.WithEntry(MultiaddrENRField, fieldRaw))
}

func DeleteField(localnode *enode.LocalNode, field string) {
	localnode.Delete(enr.WithEntry(field, struct{}{}))
}
