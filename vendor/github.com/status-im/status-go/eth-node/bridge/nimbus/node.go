// +build nimbus

package nimbusbridge

// https://golang.org/cmd/cgo/

/*
#cgo LDFLAGS: -Wl,-rpath,'$ORIGIN' -L${SRCDIR} -lnimbus -lm
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include <libnimbus.h>
*/
import "C"
import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
)

type nimbusNodeWrapper struct {
	mu sync.Mutex

	routineQueue      *RoutineQueue
	tid               int
	nodeStarted       bool
	cancelPollingChan chan struct{}

	w types.Whisper
}

type Node interface {
	types.Node

	StartNimbus(privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error
	Stop()
}

func NewNodeBridge() Node {
	c := make(chan Node, 1)
	go func(c chan<- Node, delay time.Duration) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		n := &nimbusNodeWrapper{
			routineQueue:      NewRoutineQueue(),
			tid:               syscall.Gettid(),
			cancelPollingChan: make(chan struct{}, 1),
		}
		c <- n

		for {
			select {
			case <-time.After(delay):
				n.poll()
			case <-n.cancelPollingChan:
				return
			}
		}
	}(c, 50*time.Millisecond)

	return <-c
}

func (n *nimbusNodeWrapper) StartNimbus(privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	return n.routineQueue.Send(func(c chan<- callReturn) {
		c <- callReturn{err: startNimbus(privateKey, listenAddr, staging)}
		n.nodeStarted = true
	}).err
}

func (n *nimbusNodeWrapper) Stop() {
	if n.cancelPollingChan != nil {
		close(n.cancelPollingChan)
		n.nodeStarted = false
		n.cancelPollingChan = nil
	}
}

func (n *nimbusNodeWrapper) NewENSVerifier(_ *zap.Logger) enstypes.ENSVerifier {
	panic("not implemented")
}

func (n *nimbusNodeWrapper) GetWhisper(ctx interface{}) (types.Whisper, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.w == nil {
		n.w = NewNimbusWhisperWrapper(n.routineQueue)
	}
	return n.w, nil
}

func (w *nimbusNodeWrapper) GetWaku(ctx interface{}) (types.Waku, error) {
	panic("not implemented")
}

func (n *nimbusNodeWrapper) AddPeer(url string) error {
	urlC := C.CString(url)
	defer C.free(unsafe.Pointer(urlC))
	if !C.nimbus_add_peer(urlC) {
		return fmt.Errorf("failed to add peer: %s", url)
	}

	return nil
}

func (n *nimbusNodeWrapper) RemovePeer(url string) error {
	panic("TODO: RemovePeer")
}

func (n *nimbusNodeWrapper) poll() {
	if syscall.Gettid() != n.tid {
		panic("poll called from wrong thread")
	}

	if n.nodeStarted {
		C.nimbus_poll()
	}

	n.routineQueue.HandleEvent()
}

func startNimbus(privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	C.NimMain()

	if listenAddr == "" {
		listenAddr = ":30304"
	}
	addrParts := strings.Split(listenAddr, ":")
	port, err := strconv.Atoi(addrParts[len(addrParts)-1])
	if err != nil {
		return fmt.Errorf("failed to parse port number from %s", listenAddr)
	}

	var privateKeyC unsafe.Pointer
	if privateKey != nil {
		privateKeyC = C.CBytes(crypto.FromECDSA(privateKey))
		defer C.free(privateKeyC)
	}
	if !C.nimbus_start(C.ushort(port), true, false, 0.002, (*C.uchar)(privateKeyC), C.bool(staging)) {
		return errors.New("failed to start Nimbus node")
	}

	return nil
}
