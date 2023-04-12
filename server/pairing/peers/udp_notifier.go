package peers

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	udpp2p "github.com/schollz/peerdiscovery"
	"go.uber.org/zap"
)

type NotifyHandler func(*LocalPairingPeerHello)

type UDPNotifier struct {
	logger       *zap.Logger
	id           []byte
	notifyOutput NotifyHandler
}

func NewUDPNotifier(logger *zap.Logger, outputFunc NotifyHandler) (*UDPNotifier, error) {
	randID := make([]byte, 32)
	_, err := rand.Read(randID)
	if err != nil {
		return nil, err
	}

	n := new(UDPNotifier)
	n.logger = logger
	n.id = randID
	n.notifyOutput = outputFunc
	return n, nil
}

func (u *UDPNotifier) makePayload(deviceName, deviceType string) (*LocalPairingPeerHello, error) {
	return NewLocalPairingPeerHello(u.id, deviceName, deviceType, k)
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

func (u *UDPNotifier) MakeUDPP2PSettings(deviceName, deviceType string) (*udpp2p.Settings, error) {
	if u.notifyOutput == nil {
		return nil, fmt.Errorf("UDPNotifier has no notiftOutput function defined")
	}

	h, err := u.makePayload(deviceName, deviceType)
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

func Search(deviceName, deviceType string, notify NotifyHandler, stop chan struct{}, logger *zap.Logger) error {
	un, err := NewUDPNotifier(logger, notify)
	if err != nil {
		return err
	}

	settings, err := un.MakeUDPP2PSettings(deviceName, deviceType)
	if err != nil {
		return err
	}

	settings.Delay = 500 * time.Millisecond
	settings.TimeLimit = 2 * time.Minute
	settings.StopChan = stop

	go func() {
		_, err = udpp2p.Discover(*settings)
		logger.Error("error while discovering udp peers", zap.Error(err))
	}()
	return nil
}
