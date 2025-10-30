package codex

/*
   #include "bridge.h"
   #include <stdlib.h>

   static int cGoCodexDownloadInit(void* codexCtx, char* cid, size_t chunkSize, bool local, void* resp) {
      return codex_download_init(codexCtx, cid, chunkSize, local, (CodexCallback) callback, resp);
   }

   static int cGoCodexDownloadChunk(void* codexCtx, char* cid, void* resp) {
      return codex_download_chunk(codexCtx, cid, (CodexCallback) callback, resp);
   }

   static int cGoCodexDownloadStream(void* codexCtx, char* cid, size_t chunkSize, bool local, const char* filepath, void* resp) {
      return codex_download_stream(codexCtx, cid, chunkSize, local, filepath, (CodexCallback) callback, resp);
   }

   static int cGoCodexDownloadCancel(void* codexCtx, char* cid, void* resp) {
      return codex_download_cancel(codexCtx, cid, (CodexCallback) callback, resp);
   }

   static int cGoCodexDownloadManifest(void* codexCtx, char* cid, void* resp) {
      return codex_download_manifest(codexCtx, cid, (CodexCallback) callback, resp);
   }
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"unsafe"
)

type OnDownloadProgressFunc func(read, total int, percent float64, err error)

// DownloadStreamOptions is used to download a file
// in a streaming manner in Codex.
type DownloadStreamOptions = struct {
	// Filepath is the path destination used by DownloadStream.
	// If it is set, the content will be written into the specified
	// path.
	Filepath string

	// ChunkSize is the size of each downloaded chunk. Default is to 64 KB.
	ChunkSize ChunkSize

	// OnProgress is a callback function that is called after each chunk is download with:
	//   - read: the number of bytes downloaded for the last chunk.
	//   - total: the total number of bytes downloaded so far.
	//   - percent: the percentage of the total file size that has been downloaded. It is
	//     determined from `datasetSize`.
	//   - err: an error, if one occurred.
	OnProgress OnDownloadProgressFunc

	// Writer is the path destination used by DownloadStream.
	// If it is set, the content will be written into the specified
	// Writer.
	Writer io.Writer

	// Local defines the way to download the content.
	// If true, the content will be downloaded from the
	// Local node.
	// If false (default), the content will be downloaded
	// from the network.
	Local bool

	// DatasetSize is the total size of the dataset being downloaded.
	DatasetSize int

	// DatasetSizeAuto if true, will fetch the manifest before starting
	// the downloaded to retrive the size of the data.
	DatasetSizeAuto bool
}

// DownloadInitOptions is used to create a download session.
type DownloadInitOptions = struct {
	// Local defines the way to download the content.
	// If true, the content will be downloaded from the
	// local node.
	// If false (default), the content will be downloaded
	// from the network.
	Local bool

	// ChunkSize is the size of each downloaded chunk. Default is to 64 KB.
	ChunkSize ChunkSize
}

// Manifest is the object containing the information of
// a file in Codex.
type Manifest struct {
	// Cid is the content identifier over the network
	Cid string

	// TreeCid is the root of the merkle tree
	TreeCid string `json:"treeCid"`

	// DatasetSize is the total size of all blocks
	DatasetSize int `json:"datasetSize"`

	// BlockSize is the size of each contained block
	BlockSize int `json:"blockSize"`

	// Filename is the name of the file (optional)
	Filename string `json:"filename"`

	// Mimetype is the MIME type of the file (optional)
	Mimetype string `json:"mimetype"`

	// Protected datasets have erasure coded info
	Protected bool `json:"protected"`
}

// DownloadManifest retrieves the Codex manifest from its cid.
// The session identifier is the cid, i.e you cannot have multiple
// sessions for a cid.
func (node CodexNode) DownloadManifest(cid string) (Manifest, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	if C.cGoCodexDownloadManifest(node.ctx, cCid, bridge.resp) != C.RET_OK {
		return Manifest{}, bridge.callError("cGoCodexDownloadManifest")
	}

	val, err := bridge.wait()
	if err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{Cid: cid}
	err = json.Unmarshal([]byte(val), &manifest)
	if err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

// DownloadStream download the data corresponding to a cid.
// If options.datasetSizeAuto is true, the manifest will be fetched first
// to get the dataset size.
// If options.filepath is set, the data will be written into that path.
// If options.writer is set, the data will be written into that writer.
// The options filepath and writer are not mutually exclusive, i.e you can write
// in different places in a same call.
func (node CodexNode) DownloadStream(ctx context.Context, cid string, options DownloadStreamOptions) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	if options.DatasetSizeAuto {
		manifest, err := node.DownloadManifest(cid)

		if err != nil {
			return err
		}

		options.DatasetSize = manifest.DatasetSize
	}

	total := 0
	bridge.onProgress = func(read int, chunk []byte) {
		if read == 0 {
			return
		}

		if options.Writer != nil {
			w := options.Writer
			if _, err := w.Write(chunk); err != nil {
				if options.OnProgress != nil {
					options.OnProgress(0, 0, 0.0, err)
				}
			}
		}

		total += read

		if options.OnProgress != nil {
			var percent = 0.0
			if options.DatasetSize > 0 {
				percent = float64(total) / float64(options.DatasetSize) * 100.0
			}

			options.OnProgress(read, total, percent, nil)
		}
	}

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	err := node.DownloadInit(cid, DownloadInitOptions{
		ChunkSize: options.ChunkSize,
		Local:     options.Local,
	})
	if err != nil {
		return err
	}

	defer node.DownloadCancel(cid)

	var cFilepath = C.CString(options.Filepath)
	defer C.free(unsafe.Pointer(cFilepath))

	var cLocal = C.bool(options.Local)

	if C.cGoCodexDownloadStream(node.ctx, cCid, options.ChunkSize.toSizeT(), cLocal, cFilepath, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexDownloadLocal")
	}

	// Create a done channel to signal the goroutine to stop
	// when the download is complete and avoid goroutine leaks.
	done := make(chan struct{})
	defer close(done)

	channelError := make(chan error, 1)
	var cancelled atomic.Bool
	go func() {
		select {
		case <-ctx.Done():
			channelError <- node.DownloadCancel(cid)
			cancelled.Store(true)
		case <-done:
			// Nothing to do, download finished
		}
	}()

	_, err = bridge.wait()

	// Extract the potential cancellation error
	var cancelError error
	select {
	case cancelError = <-channelError:
	default:
	}

	if err != nil {
		if cancelError != nil {
			return fmt.Errorf("context canceled: %v, but failed to cancel download session: %v", ctx.Err(), cancelError)
		}

		if cancelled.Load() {
			return context.Canceled
		}

		return err
	}

	return cancelError
}

// DownloadInit initializes the download process for a specific CID.
// This method should be used if you want to manage the download session
// and the chunk downloads manually.
func (node CodexNode) DownloadInit(cid string, options DownloadInitOptions) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	var cLocal = C.bool(options.Local)

	if C.cGoCodexDownloadInit(node.ctx, cCid, options.ChunkSize.toSizeT(), cLocal, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexDownloadInit")
	}

	_, err := bridge.wait()
	return err
}

// DownloadChunk downloads a chunk from its cid.
// You HAVE TO call `DownloadInit` before using this method.
// When using this method, you are managing at your own
// the total size downloaded (use DownloadManifest to get the
// datasetSize).
// When the download is complete, you need to call `CodexDownloadCancel`
// to free the resources.
func (node CodexNode) DownloadChunk(cid string) ([]byte, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	var bytes []byte

	bridge.onProgress = func(read int, chunk []byte) {
		bytes = chunk
	}

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	if C.cGoCodexDownloadChunk(node.ctx, cCid, bridge.resp) != C.RET_OK {
		return nil, bridge.callError("cGoCodexDownloadChunk")
	}

	if _, err := bridge.wait(); err != nil {
		return nil, err
	}

	return bytes, nil
}

// DownloadCancel cancels a download session.
// It can be only if the download session is managed manually.
// It doesn't work with DownloadStream.
func (node CodexNode) DownloadCancel(cid string) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cCid = C.CString(cid)
	defer C.free(unsafe.Pointer(cCid))

	if C.cGoCodexDownloadCancel(node.ctx, cCid, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexDownloadCancel")
	}

	_, err := bridge.wait()
	return err
}
