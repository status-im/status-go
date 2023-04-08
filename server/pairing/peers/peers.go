package peers

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"runtime"

	"github.com/golang/protobuf/proto"
	udpp2p "github.com/schollz/peerdiscovery"
	"go.uber.org/zap"

	"github.com/status-im/status-go/server"
)

var (
	// pk Yes this is an actual ECDSA **private** key committed to a public repository visible to anyone.
	// DO NOT use this key for anything other than signing udp "hellos". The key's value is in giving other Status
	// installations CONFIDENCE, NOT proof, that the sender of the UDP pings is another Status device.
	// We do not rely on UDP message information to orchestrate connections or swap secrets. The use case is purely
	// to make preflight checks which ADVISE the application and the user.
	//
	// A signature is more robust and flexible than an application identifier, and serves the same role as an ID, while
	// securing the payload against tampering.
	pk = []byte{0xbf, 0x3b, 0x37, 0x04, 0x30, 0x04, 0x32, 0x15, 0x72, 0xb0, 0x7f, 0x56, 0x72, 0x30, 0xae, 0x5b, 0x41, 0xf4, 0x4b, 0x42, 0x4a, 0xa2, 0x33, 0x53, 0x76, 0xed, 0x7a, 0xb9, 0x2d, 0x40, 0x37, 0x73}
	k  = &ecdsa.PrivateKey{}
)

func init() {
	k = server.ToECDSA(pk)
}

type UDPNotifier struct {
	logger       *zap.Logger
	id           []byte
	notifyOutput func(*LocalPairingPeerHello)
}

func NewUDPNotifier(logger *zap.Logger, outputFunc func(*LocalPairingPeerHello)) (*UDPNotifier, error) {
	randId := make([]byte, 32)
	_, err := rand.Read(randId)
	if err != nil {
		return nil, err
	}

	n := new(UDPNotifier)
	n.logger = logger
	n.id = randId
	n.notifyOutput = outputFunc
	return n, nil
}

func (u *UDPNotifier) MakePayload(deviceName string) (*LocalPairingPeerHello, error) {
	return NewLocalPairingPeerHello(u.id, deviceName, runtime.GOOS, k)
}

func (u *UDPNotifier) notify(d udpp2p.Discovered) {
	h := new(LocalPairingPeerHello)
	err := proto.Unmarshal(d.Payload, &h.LocalPairingPeerHello)
	if err != nil {
		u.logger.Error("notify unmarshalling of payload failed", zap.Error(err))
		return
	}

	ok := h.verify(&k.PublicKey)
	if !ok {
		u.logger.Error("verification of unmarshalled payload failed", zap.Any("LocalPairingPeerHello", h))
		return
	}

	h.Discovered = d
	u.notifyOutput(h)
}

func (u *UDPNotifier) MakeUDPP2PSettings(deviceName string) (*udpp2p.Settings, error) {
	if u.notifyOutput == nil {
		return nil, fmt.Errorf("UDPNotifier has no notiftOutput function defined")
	}

	h, err := u.MakePayload(deviceName)
	if err != nil {
		return nil, err
	}

	mh, err := proto.Marshal(&h.LocalPairingPeerHello)
	if err != nil {
		return nil, err
	}

	return &udpp2p.Settings{
		Notify:  u.notify,
		Payload: mh,
	}, nil
}

func Search() {
	discoveries, _ := udpp2p.Discover(udpp2p.Settings{Limit: 1, AllowSelf: true})
	for _, d := range discoveries {
		fmt.Printf("discovered '%s'\n", d.Address)
	}
}
