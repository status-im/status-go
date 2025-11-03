package codex

import (
	"encoding/json"
	"unsafe"
)

/*
   #include "bridge.h"
   #include <stdlib.h>

   static int cGoCodexStorageList(void* codexCtx, void* resp) {
       return codex_storage_list(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexStorageFetch(void* codexCtx, char* cid, void* resp) {
       return codex_storage_fetch(codexCtx, cid, (CodexCallback) callback, resp);
   }

   static int cGoCodexStorageSpace(void* codexCtx, void* resp) {
       return codex_storage_space(codexCtx, (CodexCallback) callback, resp);
   }

   static int cGoCodexStorageDelete(void* codexCtx, char* cid, void* resp) {
       return codex_storage_delete(codexCtx, cid, (CodexCallback) callback, resp);
   }

   static int cGoCodexStorageExists(void* codexCtx, char* cid, void* resp) {
       return codex_storage_exists(codexCtx, cid, (CodexCallback) callback, resp);
   }
*/
import "C"

type manifestWithCid struct {
	Cid      string   `json:"cid"`
	Manifest Manifest `json:"manifest"`
}

type Space struct {
	// TotalBlocks is the number of blocks stored by the node
	TotalBlocks int `json:"totalBlocks"`

	// QuotaMaxBytes is the maximum storage space (in bytes) available
	// for the node in Codex's local repository.
	QuotaMaxBytes int64 `json:"quotaMaxBytes"`

	// QuotaUsedBytes is the mount of storage space (in bytes) currently used
	// for storing files in Codex's local repository.
	QuotaUsedBytes int64 `json:"quotaUsedBytes"`

	// QuotaReservedBytes is the amount of storage reserved (in bytes) in the
	// Codex's local repository for future use when storage requests will be picked
	// up and hosted by the node using node's availabilities.
	// This does not include the storage currently in use.
	QuotaReservedBytes int64 `json:"quotaReservedBytes"`
}

// Manifests returns the list of all manifests stored by the Codex node.
func (node CodexNode) Manifests() ([]Manifest, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexStorageList(node.ctx, bridge.resp) != C.RET_OK {
		return nil, bridge.callError("cGoCodexStorageList")
	}
	value, err := bridge.wait()
	if err != nil {
		return nil, err
	}

	var items []manifestWithCid
	err = json.Unmarshal([]byte(value), &items)
	if err != nil {
		return nil, err
	}

	var list []Manifest
	for _, item := range items {
		item.Manifest.Cid = item.Cid
		list = append(list, item.Manifest)
	}

	return list, err
}

// Fetch download a file from the network and store it to the local node.
func (node CodexNode) Fetch(cid string) (Manifest, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	if C.cGoCodexStorageFetch(node.ctx, cCid, bridge.resp) != C.RET_OK {
		return Manifest{}, bridge.callError("cGoCodexStorageFetch")
	}

	value, err := bridge.wait()
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	err = json.Unmarshal([]byte(value), &manifest)
	if err != nil {
		return Manifest{}, err
	}

	manifest.Cid = cid
	return manifest, nil
}

// Space returns information about the storage space used and available.
func (node CodexNode) Space() (Space, error) {
	var space Space

	bridge := newBridgeCtx()
	defer bridge.free()

	if C.cGoCodexStorageSpace(node.ctx, bridge.resp) != C.RET_OK {
		return space, bridge.callError("cGoCodexStorageSpace")
	}

	value, err := bridge.wait()
	if err != nil {
		return space, err
	}

	err = json.Unmarshal([]byte(value), &space)
	return space, err
}

// Deletes either a single block or an entire dataset
// from the local node. Does nothing if the dataset is not locally available.
func (node CodexNode) Delete(cid string) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	if C.cGoCodexStorageDelete(node.ctx, cCid, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexStorageDelete")
	}

	_, err := bridge.wait()
	return err
}

// Exists checks if a given cid exists in the local storage.
func (node CodexNode) Exists(cid string) (bool, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	if C.cGoCodexStorageExists(node.ctx, cCid, bridge.resp) != C.RET_OK {
		return false, bridge.callError("cGoCodexStorageExists")
	}

	result, err := bridge.wait()
	return result == "true", err
}
