//go:build !lint

package sds

/*
	#include <stdint.h>
	#include <stdlib.h>
	#include <string.h>

	// The raw libsds exports. libsds.h is not included: its generated helpers
	// need TinyCBOR, and these bindings encode CBOR in Go instead.
	#define RET_OK         0
	#define RET_ERR        1
	#define RET_STALE_WARN 3

	typedef void (*FFICallback)(int ret, const char* msg, size_t len, void* userData);
	typedef void (*SdsRetrievalHintProvider)(const char* messageId, char** hint, size_t* hintLen, void* userData);
	typedef int (*SdsPersistenceCallback)(int operation, const char* channelId, const uint8_t* payload, size_t payloadLen, char** response, size_t* responseLen, void* userData);

	void* sds_create(const uint8_t* req, size_t reqLen, FFICallback callback, void* userData);
	int sds_wrap_outgoing_message(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
	int sds_unwrap_received_message(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
	int sds_mark_dependencies_met(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
	int sds_reset(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
	int sds_start_periodic_tasks(void* ctx, FFICallback callback, void* userData, const uint8_t* req, size_t reqLen);
	int sds_destroy(void* ctx);
	uint64_t sds_add_event_listener(void* ctx, const char* eventName, FFICallback callback, void* userData);
	int sds_set_retrieval_hint_provider(void* ctx, SdsRetrievalHintProvider callback, void* userData);
	int sds_set_persistence_callback(void* ctx, SdsPersistenceCallback callback, void* userData);

	extern void sdsGlobalEventCallback(int ret, char* msg, size_t len, uintptr_t ctx);

	extern void sdsGlobalRetrievalHintProvider(char* messageId, char** hint, size_t* hintLen, uintptr_t ctx);
	extern int sdsGlobalPersistenceCallback(int operation, char* channelId, uint8_t* payload, size_t payloadLen, char** response, size_t* responseLen, uintptr_t ctx);

	typedef struct {
		int ret;
		char* msg;
		size_t len;
	} SdsResp;

	static void* allocResp(void) {
		return calloc(1, sizeof(SdsResp));
	}

	static void freeResp(void* resp) {
		if (resp != NULL) {
			SdsResp* r = (SdsResp*) resp;
			if (r->msg != NULL) {
				free(r->msg);
			}
			free(r);
		}
	}

	// libsds frees the buffer it hands the callback as soon as the callback
	// returns, so the Go side must copy it before the waiting goroutine reads
	// it. cGoMemDup makes that copy onto the C heap (freed by freeResp).
	static char* cGoMemDup(const void* src, size_t len) {
		if (src == NULL || len == 0) {
			return NULL;
		}
		char* dst = (char*) malloc(len);
		if (dst != NULL) {
			memcpy(dst, src, len);
		}
		return dst;
	}

	static char* getMyCharPtr(void* resp) {
		if (resp == NULL) {
			return NULL;
		}
		SdsResp* m = (SdsResp*) resp;
		return m->msg;
	}

	static size_t getMyCharLen(void* resp) {
		if (resp == NULL) {
			return 0;
		}
		SdsResp* m = (SdsResp*) resp;
		return m->len;
	}

	static int getRet(void* resp) {
		if (resp == NULL) {
			return 0;
		}
		SdsResp* m = (SdsResp*) resp;
		return m->ret;
	}

	// SdsGoCallback is the result callback for the request/response FFI calls.
	void SdsGoCallback(int ret, char* msg, size_t len, void* resp);

	// The context is a token, not an address: it crosses into Go as uintptr_t
	// and is widened to void* only here.
	static uintptr_t cGoSdsCreate(const void* req, size_t reqLen, void* resp) {
		return (uintptr_t) sds_create((const uint8_t*) req, reqLen, (FFICallback) SdsGoCallback, resp);
	}

	static void cGoSdsEventTrampoline(int ret, const char* msg, size_t len, void* userData) {
		sdsGlobalEventCallback(ret, (char*) msg, len, (uintptr_t) userData);
	}

	static void cGoSdsRetrievalHintTrampoline(const char* messageId, char** hint, size_t* hintLen, void* userData) {
		sdsGlobalRetrievalHintProvider((char*) messageId, hint, hintLen, (uintptr_t) userData);
	}

	static void cGoSdsAddEventListener(uintptr_t ctx, const char* eventName) {
		// 'sdsGlobalEventCallback' is shared by all manager instances; we pass the
		// ctx as userData so the dispatcher can route the event to the instance
		// that registered it (cgo can export Go funcs but not methods).
		sds_add_event_listener((void*) ctx, eventName, (FFICallback) cGoSdsEventTrampoline, (void*) ctx);
	}

	static int cGoSdsSetRetrievalHintProvider(uintptr_t ctx) {
		return sds_set_retrieval_hint_provider((void*) ctx, (SdsRetrievalHintProvider) cGoSdsRetrievalHintTrampoline, (void*) ctx);
	}

	static int cGoSdsPersistenceTrampoline(int operation, const char* channelId, const uint8_t* payload, size_t payloadLen, char** response, size_t* responseLen, void* userData) {
		return sdsGlobalPersistenceCallback(operation, (char*) channelId, (uint8_t*) payload, payloadLen, response, responseLen, (uintptr_t) userData);
	}

	static int cGoSdsSetPersistenceCallback(uintptr_t ctx) {
		return sds_set_persistence_callback((void*) ctx, (SdsPersistenceCallback) cGoSdsPersistenceTrampoline, (void*) ctx);
	}

	static int cGoSdsWrapOutgoingMessage(uintptr_t ctx, const void* req, size_t reqLen, void* resp) {
		return sds_wrap_outgoing_message((void*) ctx, (FFICallback) SdsGoCallback, resp, (const uint8_t*) req, reqLen);
	}

	static int cGoSdsUnwrapReceivedMessage(uintptr_t ctx, const void* req, size_t reqLen, void* resp) {
		return sds_unwrap_received_message((void*) ctx, (FFICallback) SdsGoCallback, resp, (const uint8_t*) req, reqLen);
	}

	static int cGoSdsMarkDependenciesMet(uintptr_t ctx, const void* req, size_t reqLen, void* resp) {
		return sds_mark_dependencies_met((void*) ctx, (FFICallback) SdsGoCallback, resp, (const uint8_t*) req, reqLen);
	}

	static int cGoSdsReset(uintptr_t ctx, const void* req, size_t reqLen, void* resp) {
		return sds_reset((void*) ctx, (FFICallback) SdsGoCallback, resp, (const uint8_t*) req, reqLen);
	}

	static int cGoSdsStartPeriodicTasks(uintptr_t ctx, const void* req, size_t reqLen, void* resp) {
		return sds_start_periodic_tasks((void*) ctx, (FFICallback) SdsGoCallback, resp, (const uint8_t*) req, reqLen);
	}

	static int cGoSdsDestroy(uintptr_t ctx) {
		return sds_destroy((void*) ctx);
	}
*/
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/fxamacker/cbor/v2"
	"go.uber.org/zap"
)

var (
	errEmptyReliabilityManager = errors.New("empty reliability manager")
)

// respWaiters maps an in-flight call's resp pointer (a stable C-heap address)
// to the channel its result callback closes. We deliberately do NOT store a Go
// pointer (e.g. &sync.WaitGroup) in the C SdsResp: a goroutine parked in the
// call can have its stack moved by the GC, leaving the C-held Go pointer stale
// — a use-after-free that corrupts memory once the callback fires on the worker
// thread. Keying by the C pointer keeps all Go pointers on the Go side.
var respWaiters sync.Map // uintptr(resp) -> chan struct{}

// awaitResp allocates a resp, registers its waiter, runs `fire` (which
// dispatches the FFI call passing resp as userData), and blocks until the
// result callback closes the channel. The caller owns the returned resp and
// must C.freeResp it.
func awaitResp(fire func(resp unsafe.Pointer)) unsafe.Pointer {
	resp := C.allocResp()
	done := make(chan struct{})
	respWaiters.Store(uintptr(resp), done)
	fire(resp)
	<-done
	respWaiters.Delete(uintptr(resp))
	return resp
}

// Request and reply payloads, CBOR-encoded. Field names must match the {.ffi.}
// objects in nim-sds' library/libsds.nim; each request travels inside the
// envelope named after the Nim proc's parameter.
type sdsConfig struct {
	ParticipantId      string `cbor:"participantId"`
	PersistenceEnabled bool   `cbor:"persistenceEnabled"`
}

type sdsCreateRequest struct {
	Config sdsConfig `cbor:"config"`
}

type sdsWrapRequest struct {
	Message   []byte `cbor:"message"`
	MessageId string `cbor:"messageId"`
	ChannelId string `cbor:"channelId"`
}

type sdsWrapResponse struct {
	Message []byte `cbor:"message"`
}

type sdsUnwrapRequest struct {
	Message []byte `cbor:"message"`
}

type sdsMarkDependenciesRequest struct {
	MessageIds []string `cbor:"messageIds"`
	ChannelId  string   `cbor:"channelId"`
}

type sdsRequest[T any] struct {
	Req T `cbor:"req"`
}

type sdsEmptyRequest struct{}

// eventNames are the events libsds fires; each needs its own listener.
var eventNames = []string{
	"message_ready",
	"message_sent",
	"missing_dependencies",
	"periodic_sync",
	"repair_ready",
}

//export SdsGoCallback
func SdsGoCallback(ret C.int, msg *C.char, length C.size_t, resp unsafe.Pointer) {
	// A stale warning only reports a slow handler; the terminal reply follows.
	if resp == nil || ret == C.RET_STALE_WARN {
		return
	}
	m := (*C.SdsResp)(resp)
	m.ret = ret
	// libsds frees 'msg' as soon as this callback returns, so copy the bytes
	// before unblocking the waiting goroutine that reads them.
	if msg != nil && length > 0 {
		m.msg = C.cGoMemDup(unsafe.Pointer(msg), length)
		m.len = length
	}
	if ch, ok := respWaiters.Load(uintptr(resp)); ok {
		close(ch.(chan struct{}))
	}
}

func respString(resp unsafe.Pointer) string {
	return C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
}

func respBytes(resp unsafe.Pointer) []byte {
	return C.GoBytes(unsafe.Pointer(C.getMyCharPtr(resp)), C.int(C.getMyCharLen(resp)))
}

// request CBOR-encodes req, dispatches one FFI call and blocks until the result
// callback fires. It returns the CBOR-encoded reply.
func (rm *ReliabilityManager) request(
	errPrefix string,
	req any,
	fn func(req unsafe.Pointer, reqLen C.size_t, resp unsafe.Pointer) C.int,
) ([]byte, error) {
	reqCbor, err := cbor.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to encode request: %w", errPrefix, err)
	}
	cReq := C.CBytes(reqCbor)
	defer C.free(cReq)

	resp := awaitResp(func(r unsafe.Pointer) { fn(cReq, C.size_t(len(reqCbor)), r) })
	defer C.freeResp(resp)

	if C.getRet(resp) != C.RET_OK {
		return nil, fmt.Errorf("%s: %s", errPrefix, respString(resp))
	}
	return respBytes(resp), nil
}

func NewReliabilityManager(logger *zap.Logger, persistence ...PersistenceCallbacks) (*ReliabilityManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if len(persistence) > 1 {
		return nil, fmt.Errorf("at most one SDS persistence callback set is allowed")
	}

	var persistenceCallbacks PersistenceCallbacks
	if len(persistence) == 1 {
		persistenceCallbacks = persistence[0]
	}
	if err := persistenceCallbacks.validate(); err != nil {
		return nil, err
	}

	rm := &ReliabilityManager{
		logger:      logger,
		persistence: persistenceCallbacks,
	}

	rm.logger.Info("creating new reliability manager")

	// Status-go intentionally leaves participantId empty: persisted state
	// restores duplicate detection without enabling SDS-R repair.
	reqCbor, err := cbor.Marshal(sdsCreateRequest{Config: sdsConfig{
		ParticipantId:      "",
		PersistenceEnabled: persistenceCallbacks.enabled(),
	}})
	if err != nil {
		return nil, fmt.Errorf("failed to encode config: %w", err)
	}
	cReq := C.CBytes(reqCbor)
	defer C.free(cReq)

	resp := awaitResp(func(r unsafe.Pointer) {
		rm.rmCtx = uintptr(C.cGoSdsCreate(cReq, C.size_t(len(reqCbor)), r))
	})
	defer C.freeResp(resp)

	if rm.rmCtx == 0 || C.getRet(resp) != C.RET_OK {
		return nil, fmt.Errorf("error creating reliability manager: %s", respString(resp))
	}

	// Register before wiring the event listeners, since they route by ctx
	// through the registry.
	registerReliabilityManager(rm)
	for _, name := range eventNames {
		cName := C.CString(name)
		C.cGoSdsAddEventListener(C.uintptr_t(rm.rmCtx), cName)
		C.free(unsafe.Pointer(cName))
	}
	C.cGoSdsSetRetrievalHintProvider(C.uintptr_t(rm.rmCtx))
	if persistenceCallbacks.enabled() {
		if ret := C.cGoSdsSetPersistenceCallback(C.uintptr_t(rm.rmCtx)); ret != C.RET_OK {
			_ = C.cGoSdsDestroy(C.uintptr_t(rm.rmCtx))
			unregisterReliabilityManager(rm)
			return nil, fmt.Errorf("failed to register SDS persistence callback: code %d", int(ret))
		}
	}

	rm.logger.Debug("successfully created reliability manager")
	return rm, nil
}

//export sdsGlobalEventCallback
func sdsGlobalEventCallback(callerRet C.int, msg *C.char, length C.size_t, ctx C.uintptr_t) {
	msgStr := C.GoStringN(msg, C.int(length))
	rm, ok := lookupReliabilityManager(uintptr(ctx))
	if !ok {
		return
	}

	if callerRet == C.RET_OK {
		rm.OnEvent(msgStr)
	} else {
		rm.OnCallbackError(int(callerRet), msgStr)
	}
}

//export sdsGlobalRetrievalHintProvider
func sdsGlobalRetrievalHintProvider(messageId *C.char, hint **C.char, hintLen *C.size_t, ctx C.uintptr_t) {
	msgId := C.GoString(messageId)
	rm, ok := lookupReliabilityManager(uintptr(ctx))
	if ok {
		if rm.callbacks.RetrievalHintProvider != nil {
			hintBytes := rm.callbacks.RetrievalHintProvider(MessageID(msgId))
			if len(hintBytes) > 0 {
				*hint = (*C.char)(C.CBytes(hintBytes))
				*hintLen = C.size_t(len(hintBytes))
			}
		}
	}
}

func (rm *ReliabilityManager) Cleanup() error {
	if rm == nil {
		return errEmptyReliabilityManager
	}

	rm.logger.Debug("cleaning up reliability manager")

	if ret := C.cGoSdsDestroy(C.uintptr_t(rm.rmCtx)); ret != C.RET_OK {
		return fmt.Errorf("error CleanupReliabilityManager: code %d", int(ret))
	}

	unregisterReliabilityManager(rm)
	rm.logger.Debug("cleaned up reliability manager")
	return nil
}

func (rm *ReliabilityManager) Reset() error {
	if rm == nil {
		return errEmptyReliabilityManager
	}

	rm.logger.Debug("resetting reliability manager")

	_, err := rm.request("error ResetReliabilityManager", sdsEmptyRequest{},
		func(req unsafe.Pointer, reqLen C.size_t, resp unsafe.Pointer) C.int {
			return C.cGoSdsReset(C.uintptr_t(rm.rmCtx), req, reqLen, resp)
		})
	if err != nil {
		return err
	}

	rm.logger.Debug("successfully resetted reliability manager")
	return nil
}

func (rm *ReliabilityManager) WrapOutgoingMessage(message []byte, messageId MessageID, channelId string) ([]byte, error) {
	if rm == nil {
		return nil, errEmptyReliabilityManager
	}

	logger := rm.logger.With(zap.String("messageId", string(messageId)))
	logger.Debug("wrapping outgoing message")

	req := sdsRequest[sdsWrapRequest]{Req: sdsWrapRequest{
		Message:   message,
		MessageId: string(messageId),
		ChannelId: channelId,
	}}
	reply, err := rm.request("error WrapOutgoingMessage", req,
		func(req unsafe.Pointer, reqLen C.size_t, resp unsafe.Pointer) C.int {
			return C.cGoSdsWrapOutgoingMessage(C.uintptr_t(rm.rmCtx), req, reqLen, resp)
		})
	if err != nil {
		return nil, err
	}

	var wrapResp sdsWrapResponse
	if err := cbor.Unmarshal(reply, &wrapResp); err != nil {
		return nil, fmt.Errorf("failed to decode wrap response: %w", err)
	}

	logger.Debug("successfully wrapped message")
	return wrapResp.Message, nil
}

func (rm *ReliabilityManager) UnwrapReceivedMessage(message []byte) (*UnwrappedMessage, error) {
	if rm == nil {
		return nil, errEmptyReliabilityManager
	}

	reply, err := rm.request("error UnwrapReceivedMessage", sdsRequest[sdsUnwrapRequest]{Req: sdsUnwrapRequest{Message: message}},
		func(req unsafe.Pointer, reqLen C.size_t, resp unsafe.Pointer) C.int {
			return C.cGoSdsUnwrapReceivedMessage(C.uintptr_t(rm.rmCtx), req, reqLen, resp)
		})
	if err != nil {
		return nil, err
	}

	// libsds builds the unwrap result as JSON and returns it as a CBOR string.
	var respJson string
	if err := cbor.Unmarshal(reply, &respJson); err != nil {
		return nil, fmt.Errorf("failed to decode unwrap response: %w", err)
	}

	var unwrappedMessage UnwrappedMessage
	if err := json.Unmarshal([]byte(respJson), &unwrappedMessage); err != nil {
		return nil, fmt.Errorf("failed to decode unwrap response: %w", err)
	}

	rm.logger.Debug("successfully unwrapped message")
	return &unwrappedMessage, nil
}

// MarkDependenciesMet informs the library that dependencies are met
func (rm *ReliabilityManager) MarkDependenciesMet(messageIDs []MessageID, channelId string) error {
	if rm == nil {
		return errEmptyReliabilityManager
	}

	if len(messageIDs) == 0 {
		return nil // Nothing to do
	}

	ids := make([]string, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = string(id)
	}
	req := sdsRequest[sdsMarkDependenciesRequest]{Req: sdsMarkDependenciesRequest{MessageIds: ids, ChannelId: channelId}}
	_, err := rm.request("error MarkDependenciesMet", req,
		func(req unsafe.Pointer, reqLen C.size_t, resp unsafe.Pointer) C.int {
			return C.cGoSdsMarkDependenciesMet(C.uintptr_t(rm.rmCtx), req, reqLen, resp)
		})
	if err != nil {
		return err
	}

	rm.logger.Debug("successfully marked dependencies as met")
	return nil
}

func (rm *ReliabilityManager) StartPeriodicTasks() error {
	if rm == nil {
		return errEmptyReliabilityManager
	}

	rm.logger.Debug("starting periodic tasks")

	_, err := rm.request("error StartPeriodicTasks", sdsEmptyRequest{},
		func(req unsafe.Pointer, reqLen C.size_t, resp unsafe.Pointer) C.int {
			return C.cGoSdsStartPeriodicTasks(C.uintptr_t(rm.rmCtx), req, reqLen, resp)
		})
	if err != nil {
		return err
	}

	rm.logger.Debug("successfully started periodic tasks")
	return nil
}
