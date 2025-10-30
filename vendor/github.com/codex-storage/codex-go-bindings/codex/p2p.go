package codex

/*
   #include "bridge.h"
   #include <stdlib.h>

   static int cGoCodexConnect(void* codexCtx, char* peerId, const char** peerAddresses, uintptr_t peerAddressesSize,  void* resp) {
       return codex_connect(codexCtx, peerId, peerAddresses, peerAddressesSize, (CodexCallback) callback, resp);
   }
*/
import "C"
import (
	"unsafe"
)

// Connect connects to a peer using its peer ID and optional multiaddresses.
// If `peerAddresses` param is supplied, it will be used to  dial the peer,
// otherwise the `peerId` is used to invoke peer discovery, if it succeeds
// the returned addresses will be used to dial.
// `peerAddresses` the listening addresses of the peers to dial,
// eg the one specified with `ListenAddresses` in `CodexConfig`.
func (node CodexNode) Connect(peerId string, peerAddresses []string) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cPeerId = C.CString(peerId)
	defer C.free(unsafe.Pointer(cPeerId))

	if len(peerAddresses) > 0 {
		var cAddresses = make([]*C.char, len(peerAddresses))
		for i, addr := range peerAddresses {
			cAddresses[i] = C.CString(addr)
			defer C.free(unsafe.Pointer(cAddresses[i]))
		}

		if C.cGoCodexConnect(node.ctx, cPeerId, &cAddresses[0], C.uintptr_t(len(peerAddresses)), bridge.resp) != C.RET_OK {
			return bridge.callError("cGoCodexConnect")
		}
	} else {
		if C.cGoCodexConnect(node.ctx, cPeerId, nil, 0, bridge.resp) != C.RET_OK {
			return bridge.callError("cGoCodexConnect")
		}
	}

	_, err := bridge.wait()
	return err
}
