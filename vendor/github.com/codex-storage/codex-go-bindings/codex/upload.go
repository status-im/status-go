package codex

/*
   #include "bridge.h"
   #include <stdlib.h>

   static int cGoCodexUploadInit(void* codexCtx, char* filepath, size_t chunkSize, void* resp) {
      return codex_upload_init(codexCtx, filepath, chunkSize, (CodexCallback) callback, resp);
   }

   static int cGoCodexUploadChunk(void* codexCtx, char* sessionId, const uint8_t* chunk, size_t len, void* resp) {
      return codex_upload_chunk(codexCtx, sessionId, chunk, len, (CodexCallback) callback, resp);
   }

   static int cGoCodexUploadFinalize(void* codexCtx, char* sessionId, void* resp) {
      return codex_upload_finalize(codexCtx, sessionId, (CodexCallback) callback, resp);
   }

   static int cGoCodexUploadCancel(void* codexCtx, char* sessionId, void* resp) {
      return codex_upload_cancel(codexCtx, sessionId, (CodexCallback) callback, resp);
   }

   static int cGoCodexUploadFile(void* codexCtx, char* sessionId, void* resp) {
      return codex_upload_file(codexCtx, sessionId, (CodexCallback) callback, resp);
   }
*/
import "C"
import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"unsafe"
)

const defaultBlockSize = 1024 * 64

type OnUploadProgressFunc func(read, total int, percent float64, err error)

type UploadOptions struct {
	// Filepath can be the full path when using UploadFile
	// otherwise the file name.
	// It is used to detect the mimetype.
	Filepath string

	// ChunkSize is the size of each upload chunk, passed as `blockSize` to the Codex node
	// store. Default is to 64 KB.
	ChunkSize ChunkSize

	// OnProgress is a callback function that is called after each chunk is uploaded with:
	//   - read: the number of bytes read in the last chunk.
	//   - total: the total number of bytes read so far.
	//   - percent: the percentage of the total file size that has been uploaded. It is
	//     determined from a `stat` call if it is a file and from the length of the buffer
	// 	   if it is a buffer. Otherwise, it is 0.
	//   - err: an error, if one occurred.
	//
	// If the chunk size is more than the `chunkSize` parameter, the callback is called
	// after the block is actually stored in the block store. Otherwise, it is called
	// after the chunk is sent to the stream.
	OnProgress OnUploadProgressFunc
}

func getReaderSize(r io.Reader) int64 {
	switch v := r.(type) {
	case *os.File:
		stat, err := v.Stat()
		if err != nil {
			return 0
		}
		return stat.Size()
	case *bytes.Buffer:
		return int64(v.Len())
	default:
		return 0
	}
}

// UploadInit initializes a new upload session.
// It returns a session ID that can be used for subsequent upload operations.
// This function is called by UploadReader and UploadFile internally.
// You should use this function only if you need to manage the upload session manually.
func (node CodexNode) UploadInit(options *UploadOptions) (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cFilename = C.CString(options.Filepath)
	defer C.free(unsafe.Pointer(cFilename))

	if C.cGoCodexUploadInit(node.ctx, cFilename, options.ChunkSize.toSizeT(), bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexUploadInit")
	}

	return bridge.wait()
}

// UploadChunk uploads a chunk of data to the Codex node.
// It takes the session ID returned by UploadInit
// and a byte slice containing the chunk data.
// This function is called by UploadReader internally.
// You should use this function only if you need to manage the upload session manually.
func (node CodexNode) UploadChunk(sessionId string, chunk []byte) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cSessionId = C.CString(sessionId)
	defer C.free(unsafe.Pointer(cSessionId))

	var cChunkPtr *C.uint8_t
	if len(chunk) > 0 {
		cChunkPtr = (*C.uint8_t)(unsafe.Pointer(&chunk[0]))
	}

	if C.cGoCodexUploadChunk(node.ctx, cSessionId, cChunkPtr, C.size_t(len(chunk)), bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexUploadChunk")
	}

	_, err := bridge.wait()
	return err
}

// UploadFinalize finalizes the upload session and returns the CID of the uploaded file.
// It takes the session ID returned by UploadInit.
// This function is called by UploadReader and UploadFile internally.
// You should use this function only if you need to manage the upload session manually.
func (node CodexNode) UploadFinalize(sessionId string) (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cSessionId = C.CString(sessionId)
	defer C.free(unsafe.Pointer(cSessionId))

	if C.cGoCodexUploadFinalize(node.ctx, cSessionId, bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexUploadFinalize")
	}

	return bridge.wait()
}

// UploadCancel cancels an ongoing upload session.
// It can be only if the upload session is managed manually.
// It doesn't work with UploadFile.
func (node CodexNode) UploadCancel(sessionId string) error {
	bridge := newBridgeCtx()
	defer bridge.free()

	var cSessionId = C.CString(sessionId)
	defer C.free(unsafe.Pointer(cSessionId))

	if C.cGoCodexUploadCancel(node.ctx, cSessionId, bridge.resp) != C.RET_OK {
		return bridge.callError("cGoCodexUploadCancel")
	}

	_, err := bridge.wait()
	return err
}

// UploadReader uploads data from an io.Reader to the Codex node.
// It takes the upload options and the reader as parameters.
// It returns the CID of the uploaded file or an error.
//
// Internally, it calls:
// - UploadInit to create the upload session.
// - UploadChunk to upload a chunk to codex.
// - UploadFinalize to finalize the upload session.
// - UploadCancel if an error occurs.
func (node CodexNode) UploadReader(ctx context.Context, options UploadOptions, r io.Reader) (string, error) {
	sessionId, err := node.UploadInit(&options)
	if err != nil {
		return "", err
	}
	defer node.UploadCancel(sessionId)

	buf := make([]byte, options.ChunkSize.valOrDefault())
	total := 0

	var size int64
	if options.OnProgress != nil {
		size = getReaderSize(r)
	}

	for {
		select {
		case <-ctx.Done():
			if cancelErr := node.UploadCancel(sessionId); cancelErr != nil {
				return "", fmt.Errorf("upload canceled: %v, but failed to cancel upload session: %v", ctx.Err(), cancelErr)
			}
			return "", context.Canceled
		default:
			// continue
		}

		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}

		if err != nil {
			if cancelErr := node.UploadCancel(sessionId); cancelErr != nil {
				return "", fmt.Errorf("failed to upload chunk %v and failed to cancel upload session %v", err, cancelErr)
			}

			return "", err
		}

		if n == 0 {
			break
		}

		if err := node.UploadChunk(sessionId, buf[:n]); err != nil {
			if cancelErr := node.UploadCancel(sessionId); cancelErr != nil {
				return "", fmt.Errorf("failed to upload chunk %v and failed to cancel upload session %v", err, cancelErr)
			}

			return "", err
		}

		total += n
		if options.OnProgress != nil && size > 0 {
			percent := float64(total) / float64(size) * 100.0
			// The last block could be a bit over the size due to padding
			// on the chunk size.
			if percent > 100.0 {
				percent = 100.0
			}
			options.OnProgress(n, total, percent, nil)
		} else if options.OnProgress != nil {
			options.OnProgress(n, total, 0, nil)
		}
	}

	return node.UploadFinalize(sessionId)
}

// UploadReaderAsync is the asynchronous version of UploadReader using a goroutine.
func (node CodexNode) UploadReaderAsync(ctx context.Context, options UploadOptions, r io.Reader, onDone func(cid string, err error)) {
	go func() {
		cid, err := node.UploadReader(ctx, options, r)
		onDone(cid, err)
	}()
}

// UploadFile uploads a file to the Codex node.
// It takes the upload options as parameter.
// It returns the CID of the uploaded file or an error.
//
// The options parameter contains the following fields:
// - filepath: the full path of the file to upload.
// - chunkSize: the size of each upload chunk, passed as `blockSize` to the Codex node
// store. Default is to 64 KB.
// - onProgress: a callback function that is called after each chunk is uploaded with:
//   - read: the number of bytes read in the last chunk.
//   - total: the total number of bytes read so far.
//   - percent: the percentage of the total file size that has been uploaded. It is
//     determined from a `stat` call.
//   - err: an error, if one occurred.
//
// If the chunk size is more than the `chunkSize` parameter, the callback is called after
// the block is actually stored in the block store. Otherwise, it is called after the chunk
// is sent to the stream.
//
// Internally, it calls UploadInit to create the upload session.
func (node CodexNode) UploadFile(ctx context.Context, options UploadOptions) (string, error) {
	bridge := newBridgeCtx()
	defer bridge.free()

	if options.OnProgress != nil {
		stat, err := os.Stat(options.Filepath)
		if err != nil {
			return "", err
		}

		size := stat.Size()
		total := 0

		if size > 0 {
			bridge.onProgress = func(read int, _ []byte) {
				if read == 0 {
					return
				}

				total += read
				percent := float64(total) / float64(size) * 100.0
				// The last block could be a bit over the size due to padding
				// on the chunk size.
				if percent > 100.0 {
					percent = 100.0
				}

				options.OnProgress(read, int(size), percent, nil)
			}
		}
	}

	sessionId, err := node.UploadInit(&options)
	if err != nil {
		return "", err
	}
	defer node.UploadCancel(sessionId)

	var cSessionId = C.CString(sessionId)
	defer C.free(unsafe.Pointer(cSessionId))

	if C.cGoCodexUploadFile(node.ctx, cSessionId, bridge.resp) != C.RET_OK {
		return "", bridge.callError("cGoCodexUploadFile")
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
			channelError <- node.UploadCancel(sessionId)
			cancelled.Store(true)
		case <-done:
			// Nothing to do, upload finished
		}
	}()

	_, err = bridge.wait()

	// Extract the potential cancellation error
	var cancelErr error
	select {
	case cancelErr = <-channelError:
	default:
	}

	if err != nil {
		if cancelErr != nil {
			return "", fmt.Errorf("context canceled: %v, but failed to cancel upload session: %v", ctx.Err(), cancelErr)
		}

		if cancelled.Load() {
			return "", context.Canceled
		}

		return "", err
	}

	return bridge.result, cancelErr
}

// UploadFileAsync is the asynchronous version of UploadFile using a goroutine.
func (node CodexNode) UploadFileAsync(ctx context.Context, options UploadOptions, onDone func(cid string, err error)) {
	go func() {
		cid, err := node.UploadFile(ctx, options)
		onDone(cid, err)
	}()
}
