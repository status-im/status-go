package pairing

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	udpp2p "github.com/schollz/peerdiscovery"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/server"
)

func testSignal(h *protobuf.LocalPairingPeerHello) {
	fmt.Printf("peer message - %s : %s\n", h.DeviceName, h.DeviceType)
}

func Test(t *testing.T) {
	n, err := server.GetDeviceName()
	if err != nil {
		t.Error(err)
	}

	d1, err := makeDevicePayload(n+" - device 1", runtime.GOOS, k)
	if err != nil {
		t.Error(err)
	}

	d2, err := makeDevicePayload(n+" - device 2", runtime.GOOS, k)
	if err != nil {
		t.Error(err)
	}

	u := new(UDPNotifier)
	u.notifyOutput = testSignal

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		s, err := u.makeUDPP2PSettings(d1)
		if err != nil {
			t.Error("1 -", err)
		}

		_, err = udpp2p.Discover(*s)
		if err != nil {
			t.Error("1 -", err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		s, err := u.makeUDPP2PSettings(d2)
		if err != nil {
			t.Error("2 -", err)
		}

		_, err = udpp2p.Discover(*s)
		if err != nil {
			t.Error("2 -", err)
		}
		wg.Done()
	}()

	wg.Wait()
}
