package pairing

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	udpp2p "github.com/schollz/peerdiscovery"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
)

var (
	pk = []byte{0xbf, 0x3b, 0x37, 0x04, 0x30, 0x04, 0x32, 0x15, 0x72, 0xb0, 0x7f, 0x56, 0x72, 0x30, 0xae, 0x5b, 0x41, 0xf4, 0x4b, 0x42, 0x4a, 0xa2, 0x33, 0x53, 0x76, 0xed, 0x7a, 0xb9, 0x2d, 0x40, 0x37, 0x73}
	k  = &ecdsa.PrivateKey{}
)

func init() {
	k = server.ToECDSA(pk)
}

func sign(h *protobuf.LocalPairingPeerHello, k *ecdsa.PrivateKey) error {
	dHash := sha256.Sum256(append(h.PeerId, []byte(h.DeviceName+h.DeviceType)...))
	s, err := ecdsa.SignASN1(rand.Reader, k, dHash[:])
	if err != nil {
		return err
	}

	h.Signature = s
	return nil
}

func verify(h *protobuf.LocalPairingPeerHello, k *ecdsa.PrivateKey) bool {
	dHash := sha256.Sum256(append(h.PeerId, []byte(h.DeviceName+h.DeviceType)...))
	return ecdsa.VerifyASN1(&k.PublicKey, dHash[:], h.Signature)
}

func makeDevicePayload(name, deviceType string, k *ecdsa.PrivateKey) (*protobuf.LocalPairingPeerHello, error) {
	h := new(protobuf.LocalPairingPeerHello)

	randId := make([]byte, 32)
	_, err := rand.Read(randId)
	if err != nil {
		return nil, err
	}

	h.PeerId = randId
	h.DeviceName = name
	h.DeviceType = deviceType

	err = sign(h, k)
	if err != nil {
		return nil, err
	}

	return h, nil
}

type UDPNotifier struct {
	notifyOutput func(*protobuf.LocalPairingPeerHello)
}

func (u *UDPNotifier) notify(d udpp2p.Discovered) {
	h := new(protobuf.LocalPairingPeerHello)
	err := proto.Unmarshal(d.Payload, h)
	if err != nil {
		// TODO add logging rather than dumping
		spew.Dump(err)
	}

	ok := verify(h, k)
	if !ok {
		return
	}

	u.notifyOutput(h)
}

func (u *UDPNotifier) makeUDPP2PSettings(h *protobuf.LocalPairingPeerHello) (*udpp2p.Settings, error) {
	mh, err := proto.Marshal(h)
	if err != nil {
		return nil, err
	}

	return &udpp2p.Settings{
		Limit:     4,
		AllowSelf: true,
		Notify:    u.notify,
		Payload:   mh,
	}, nil
}

func Search() {
	discoveries, _ := udpp2p.Discover(udpp2p.Settings{Limit: 1, AllowSelf: true})
	for _, d := range discoveries {
		fmt.Printf("discovered '%s'\n", d.Address)
	}
}
