/**
* libcodex.h - C Interface for Example Library
*
* This header provides the public API for libcodex
*
* To see the auto-generated header by Nim, run `make libcodex` from the
* repository root. The generated file will be created at:
* nimcache/release/libcodex/libcodex.h
*/

#ifndef __libcodex__
#define __libcodex__

#include <stddef.h>
#include <stdint.h>

// The possible returned values for the functions that return int
#define RET_OK                0
#define RET_ERR               1
#define RET_MISSING_CALLBACK  2
#define RET_PROGRESS          3

#ifdef __cplusplus
extern "C" {
#endif

typedef void (*CodexCallback) (int callerRet, const char* msg, size_t len, void* userData);

void* codex_new(
             const char* configJson,
             CodexCallback callback,
             void* userData);

int codex_version(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_revision(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_repo(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_debug(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_spr(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_peer_id(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_log_level(
                void* ctx,
                const char* logLevel,
                CodexCallback callback,
                void* userData);

int codex_connect(
                void* ctx,
                const char* peerId,
                const char** peerAddresses,
                size_t peerAddressesSize,
                CodexCallback callback,
                void* userData);

int codex_peer_debug(
                void* ctx,
                const char* peerId,
                CodexCallback callback,
                void* userData);


int codex_upload_init(
                void* ctx,
                const char* filepath,
                size_t chunkSize,
                CodexCallback callback,
                void* userData);

int codex_upload_chunk(
                void* ctx,
                const char* sessionId,
                const uint8_t* chunk,
                size_t len,
                CodexCallback callback,
                void* userData);

int codex_upload_finalize(
                void* ctx,
                const char* sessionId,
                CodexCallback callback,
                void* userData);

int codex_upload_cancel(
                void* ctx,
                const char* sessionId,
                CodexCallback callback,
                void* userData);

int codex_upload_file(
                void* ctx,
                const char* sessionId,
                CodexCallback callback,
                void* userData);

int codex_download_stream(
                void* ctx,
                const char* cid,
                size_t chunkSize,
                bool local,
                const char* filepath,
                CodexCallback callback,
                void* userData);

int codex_download_init(
                void* ctx,
                const char* cid,
                size_t chunkSize,
                bool local,
                CodexCallback callback,
                void* userData);

int codex_download_chunk(
                void* ctx,
                const char* cid,
                CodexCallback callback,
                void* userData);

int codex_download_cancel(
                void* ctx,
                const char* cid,
                CodexCallback callback,
                void* userData);

int codex_download_manifest(
                void* ctx,
                const char* cid,
                CodexCallback callback,
                void* userData);

int codex_storage_list(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_storage_space(
                void* ctx,
                CodexCallback callback,
                void* userData);

int codex_storage_delete(
                void* ctx,
                const char* cid,
                CodexCallback callback,
                void* userData);

int codex_storage_fetch(
                void* ctx,
                const char* cid,
                CodexCallback callback,
                void* userData);

int codex_start(void* ctx,
               CodexCallback callback,
               void* userData);

int codex_stop(void* ctx,
              CodexCallback callback,
              void* userData);

int codex_close(void* ctx,
              CodexCallback callback,
              void* userData);

// Destroys an instance of a codex node created with codex_new
int codex_destroy(void* ctx,
                  CodexCallback callback,
                 void* userData);

void codex_set_event_callback(void* ctx,
                             CodexCallback callback,
                             void* userData);

#ifdef __cplusplus
}
#endif

#endif /* __libcodex__ */