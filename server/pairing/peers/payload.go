package peers

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"

	udpp2p "github.com/schollz/peerdiscovery"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server/pairing/versioning"
)

type LocalPairingPeerHello struct {
	protobuf.LocalPairingPeerHello
	Discovered udpp2p.Discovered
}

func NewLocalPairingPeerHello(id []byte, name, deviceType string, k *ecdsa.PrivateKey) (*LocalPairingPeerHello, error) {
	h := new(LocalPairingPeerHello)

	h.PairingVersion = int32(versioning.LatestLocalPairingVer)
	h.PeerId = id
	h.DeviceName = name
	h.DeviceType = deviceType

	err := h.sign(k)
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *LocalPairingPeerHello) MarshalJSON() ([]byte, error) {
	alias := struct {
		PairingVersion int32
		PeerID         []byte
		DeviceName     string
		DeviceType     string
		Address        string
	}{
		PairingVersion: h.PairingVersion,
		PeerID:         h.PeerId,
		DeviceName:     h.DeviceName,
		DeviceType:     h.DeviceType,
		Address:        h.Discovered.Address,
	}

	return json.Marshal(alias)
}

func (h *LocalPairingPeerHello) hash() []byte {
	dHash := sha256.Sum256(append(h.PeerId, []byte(h.DeviceName+h.DeviceType)...))
	return dHash[:]
}

func (h *LocalPairingPeerHello) sign(k *ecdsa.PrivateKey) error {
	s, err := ecdsa.SignASN1(rand.Reader, k, h.hash())
	if err != nil {
		return err
	}

	h.Signature = s
	return nil
}

func (h *LocalPairingPeerHello) verify(k *ecdsa.PublicKey) bool {
	return ecdsa.VerifyASN1(k, h.hash(), h.Signature)
}
