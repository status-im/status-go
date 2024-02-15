package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"unsafe"

	"github.com/vitvly/lc-proxy-wrapper/types"
)

/*
#include <stdlib.h>
#include "verifproxy.h"

typedef void (*callback_type)(char *, int);
void goCallback_cgo(char *, int);

*/
import "C"

type Web3UrlType struct {
	Kind    string
	Web3Url string
}
type Config struct {
	Eth2Network      string
	TrustedBlockRoot string
	Web3Url          string
	RpcAddress       string
	RpcPort          uint16
	LogLevel         string
}

var proxyEventChan chan *types.ProxyEvent

var nimContextPtr unsafe.Pointer

//export goCallback
func goCallback(json *C.char, cbType int) {
	//C.free(unsafe.Pointer(json))
	//fmt.Println("### goCallback " + goStr)
	var goStr string
	if json != nil {
		goStr = C.GoString(json)
	}
	if proxyEventChan != nil {
		if cbType == 0 { // finalized header
			proxyEventChan <- &types.ProxyEvent{types.FinalizedHeader, goStr}
		} else if cbType == 1 { // optimistic header
			proxyEventChan <- &types.ProxyEvent{types.OptimisticHeader, goStr}
		} else if cbType == 2 { // stopped
			proxyEventChan <- &types.ProxyEvent{types.Stopped, goStr}
			close(proxyEventChan)
			proxyEventChan = nil
			nimContextPtr = nil
		} else if cbType == 3 { // error
			proxyEventChan <- &types.ProxyEvent{types.Error, goStr}
			close(proxyEventChan)
			proxyEventChan = nil
			nimContextPtr = nil
		}
	}
}

func StartVerifProxy(cfg *Config) (chan *types.ProxyEvent, error) {
	if nimContextPtr != nil {
		// Other instance running
		return nil, errors.New("Nimbux proxy already (still) running")
	}
	proxyEventChan = make(chan *types.ProxyEvent, 10)
	cb := (C.callback_type)(unsafe.Pointer(C.goCallback_cgo))

	jsonBytes, _ := json.Marshal(cfg)
	jsonStr := string(jsonBytes)
	fmt.Println("### jsonStr: ", jsonStr)
	configCStr := C.CString(jsonStr)
	nimContextPtr = unsafe.Pointer(C.startVerifProxy(configCStr, cb))
	fmt.Println("ptr: %p", nimContextPtr)
	fmt.Println("inside go-func after startLcViaJson")

	return proxyEventChan, nil

}

func StopVerifProxy() error {
	if nimContextPtr != nil {
		C.stopVerifProxy((*C.struct_VerifProxyContext)(nimContextPtr))
		return nil
	} else {
		return errors.New("Nimbux proxy not running")
	}
}
