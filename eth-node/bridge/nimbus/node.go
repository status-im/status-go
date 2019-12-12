// +build nimbus

package nimbusbridge

// https://golang.org/cmd/cgo/

/*
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

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"go.uber.org/zap"
)

func Init() {
	runtime.LockOSThread()
}

func StartNimbus(privateKey *ecdsa.PrivateKey, listenAddr string, staging bool) error {
	C.NimMain()

	port, err := strconv.Atoi(strings.Split(listenAddr, ":")[1])
	if err != nil {
		return fmt.Errorf("failed to parse port number from %s", listenAddr)
	}

	privateKeyC := C.CBytes(crypto.FromECDSA(privateKey))
	defer C.free(privateKeyC)
	if !C.nimbus_start(C.ushort(port), true, false, 0.002, (*C.uchar)(privateKeyC), C.bool(staging)) {
		return errors.New("failed to start Nimbus node")
	}

	return nil
}

type nimbusNodeWrapper struct {
	w types.Whisper
}

func NewNodeBridge() types.Node {
	return &nimbusNodeWrapper{w: NewNimbusWhisperWrapper()}
}

func (w *nimbusNodeWrapper) NewENSVerifier(_ *zap.Logger) enstypes.ENSVerifier {
	panic("not implemented")
}

func (w *nimbusNodeWrapper) GetWhisper(ctx interface{}) (types.Whisper, error) {
	return w.w, nil
}

func (w *nimbusNodeWrapper) AddPeer(url string) error {
	urlC := C.CString(url)
	defer C.free(unsafe.Pointer(urlC))
	if !C.nimbus_add_peer(urlC) {
		return fmt.Errorf("failed to add peer: %s", url)
	}

	return nil
}

func (w *nimbusNodeWrapper) RemovePeer(url string) error {
	panic("TODO: RemovePeer")
}
