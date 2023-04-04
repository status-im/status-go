package pairing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
	"math/big"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/schollz/peerdiscovery"
)

var (
	pk = []byte{0xbf, 0x3b, 0x37, 0x04, 0x30, 0x04, 0x32, 0x15, 0x72, 0xb0, 0x7f, 0x56, 0x72, 0x30, 0xae, 0x5b, 0x41, 0xf4, 0x4b, 0x42, 0x4a, 0xa2, 0x33, 0x53, 0x76, 0xed, 0x7a, 0xb9, 0x2d, 0x40, 0x37, 0x73}
	k  = &ecdsa.PrivateKey{}
)

func init() {
	k = buildECKey()
}

func notify(d peerdiscovery.Discovered) {
	h := new(protobuf.LocalPairingPeerHello)
	err := proto.Unmarshal(d.Payload, h)
	if err != nil {
		spew.Dump(err)
	}

	fmt.Printf("payload - %+v\n", h)

	dHash := sha256.Sum256([]byte(h.Name))
	ok := ecdsa.VerifyASN1(&k.PublicKey, dHash[:], h.Signature)
	spew.Dump("verified", ok)
}

func buildECKey() *ecdsa.PrivateKey {
	k := big.NewInt(0).SetBytes(pk)

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = elliptic.P256()
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = elliptic.P256().ScalarBaseMult(k.Bytes())
	return priv
}

func makeDevicePayload(name string, k *ecdsa.PrivateKey) (*protobuf.LocalPairingPeerHello, error) {
	d := new(protobuf.LocalPairingPeerHello)
	dHash := sha256.Sum256([]byte(name))

	s, err := ecdsa.SignASN1(rand.Reader, k, dHash[:])
	if err != nil {
		return nil, err
	}

	d.Name = name
	d.Signature = s

	return d, nil
}

func Test(t *testing.T) {
	n, err := server.GetDeviceName()
	if err != nil {
		spew.Dump(err)
	}

	d1, err := makeDevicePayload(n+" - device 1", k)
	if err != nil {
		spew.Dump(err)
	}

	d2, err := makeDevicePayload(n+" - device 2", k)
	if err != nil {
		spew.Dump(err)
	}

	md1, err := proto.Marshal(d1)
	if err != nil {
		spew.Dump(err)
	}

	md2, err := proto.Marshal(d2)
	if err != nil {
		spew.Dump(err)
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		_, err := peerdiscovery.Discover(peerdiscovery.Settings{
			Limit:     4,
			AllowSelf: true,
			Notify:    notify,
			Payload:   md1,
		})
		if err != nil {
			spew.Dump("error 1", err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		_, err := peerdiscovery.Discover(peerdiscovery.Settings{
			Limit:     4,
			AllowSelf: true,
			Notify:    notify,
			Payload:   md2,
		})
		if err != nil {
			spew.Dump("error 2", err)
		}
		wg.Done()
	}()

	wg.Wait()
	spew.Dump("done")
}
