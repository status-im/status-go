package codex

/*
   #include "bridge.h"
   #include <stdlib.h>

   static int cGoCodexDebug(void* codexCtx, void* resp) {
       return codex_debug(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexLogLevel(void* codexCtx, char* logLevel, void* resp) {
       return codex_log_level(codexCtx, logLevel, (CodexCallback) callback, resp);
   }

   static int cGoCodexPeerDebug(void* codexCtx, char* peerId, void* resp) {
       return codex_peer_debug(codexCtx, peerId, (CodexCallback) callback, resp);
   }
*/
import "C"
import (
	"encoding/json"
	"unsafe"
)

type Node struct {
	NodeId  string  `json:"nodeId"`
	PeerId  string  `json:"peerId"`
	Record  string  `json:"record"`
	Address *string `json:"address"`
	Seen    bool    `json:"seen"`
}

type RoutingTable struct {
	LocalNode Node   `json:"localNode"`
	Nodes     []Node `json:"nodes"`
}

type DebugInfo struct {
	// Peer ID
	ID string `json:"id"`

	// Peer info addresses
	// Specified with `ListenAddresses` in `CodexConfig`
	Addrs []string `json:"addrs"`

	Spr               string       `json:"spr"`
	AnnounceAddresses []string     `json:"announceAddresses"`
	PeersTable        RoutingTable `json:"table"`
}

type PeerRecord struct {
	PeerId    string   `json:"peerId"`
	SeqNo     int      `json:"seqNo"`
	Addresses []string `json:"addresses,omitempty"`
}

// Debug retrieves debugging information from the Codex node.
func (node CodexNode) Debug() (DebugInfo, error) {
	var info DebugInfo

	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexDebug(node.ctx, bridge.resp) != C.RET_OK {
		return info, bridge.callError("cGoCodexDebug")
	}

	value, err := bridge.wait()
	if err != nil {
		return info, err
	}

	err = json.Unmarshal([]byte(value), &info)
	return info, err
}

// UpdateLogLevel updates Chronicles’ runtime logging configuration.
// You can pass a plain level: TRACE, DEBUG, INFO, NOTICE, WARN, ERROR, FATAL.
// The default level is TRACE.
// You can also use Chronicles topic directives. So for example if you want
// to update the general level to INFO but want to see TRACE logs for the codexlib
// topic, you can pass "INFO,codexlib:TRACE".
func (node CodexNode) UpdateLogLevel(logLevel string) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cLogLevel = C.CString(string(logLevel))
	defer C.free(unsafe.Pointer(cLogLevel))

	if C.cGoCodexLogLevel(node.ctx, cLogLevel, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexLogLevel")
	}

	_, err := bridge.wait()
	return err
}

// CodexPeerDebug retrieves the peer record for a given peer ID.
// This function is available only if the flag
// -d:codex_enable_api_debug_peers=true was set at build time.
func (node CodexNode) CodexPeerDebug(peerId string) (PeerRecord, error) {
	var record PeerRecord

	bridge := newBridgeCtx()
	defer bridge.free()

	var cPeerId = C.CString(peerId)
	defer C.free(unsafe.Pointer(cPeerId))

	if C.cGoCodexPeerDebug(node.ctx, cPeerId, bridge.resp) != C.RET_OK {
		return record, bridge.callError("cGoCodexPeerDebug")
	}

	value, err := bridge.wait()
	if err != nil {
		return record, err
	}

	err = json.Unmarshal([]byte(value), &record)
	return record, err
}
