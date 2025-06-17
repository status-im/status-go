package sds

/*
	#cgo LDFLAGS: -L../third_party/nim-sds/build/ -lsds
	#cgo LDFLAGS: -L../third_party/nim-sds -Wl,-rpath,../third_party/nim-sds/build/

	#include "../third_party/nim-sds/library/libsds.h"
	#include <stdio.h>
	#include <stdlib.h>

	extern void sdsGlobalEventCallback(int ret, char* msg, size_t len, void* userData);

	typedef struct {
		int ret;
		char* msg;
		size_t len;
		void* ffiWg;
	} SdsResp;

	static void* allocResp(void* wg) {
		SdsResp* r = calloc(1, sizeof(SdsResp));
		r->ffiWg = wg;
		return r;
	}

	static void freeResp(void* resp) {
		if (resp != NULL) {
			free(resp);
		}
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

	// resp must be set != NULL in case interest on retrieving data from the callback
	void SdsGoCallback(int ret, char* msg, size_t len, void* resp);

	static void* cGoSdsNewReliabilityManager(const char* channelId, void* resp) {
		// We pass NULL because we are not interested in retrieving data from this callback
		void* ret = SdsNewReliabilityManager(channelId, (SdsCallBack) SdsGoCallback, resp);
		return ret;
	}

	static void cGoSdsSetEventCallback(void* rmCtx) {
		// The 'sdsGlobalEventCallback' Go function is shared amongst all possible Reliability Manager instances.

		// Given that the 'sdsGlobalEventCallback' is shared, we pass again the
		// rmCtx instance but in this case is needed to pick up the correct method
		// that will handle the event.

		// In other words, for every call libsds makes to sdsGlobalEventCallback,
		// the 'userData' parameter will bring the context of the rm that registered
		// that sdsGlobalEventCallback.

		// This technique is needed because cgo only allows to export Go functions and not methods.

		SdsSetEventCallback(rmCtx, (SdsCallBack) sdsGlobalEventCallback, rmCtx);
	}

	static void cGoSdsCleanupReliabilityManager(void* rmCtx, void* resp) {
		SdsCleanupReliabilityManager(rmCtx, (SdsCallBack) SdsGoCallback, resp);
	}

	static void cGoSdsResetReliabilityManager(void* rmCtx, void* resp) {
		SdsResetReliabilityManager(rmCtx, (SdsCallBack) SdsGoCallback, resp);
	}

	static void cGoSdsWrapOutgoingMessage(void* rmCtx,
									void* message,
                    				size_t messageLen,
                    				const char* messageId,
									void* resp) {
		SdsWrapOutgoingMessage(rmCtx,
						message,
						messageLen,
						messageId,
						(SdsCallBack) SdsGoCallback,
						resp);
	}
	static void cGoSdsUnwrapReceivedMessage(void* rmCtx,
									void* message,
                    				size_t messageLen,
									void* resp) {
		SdsUnwrapReceivedMessage(rmCtx,
						message,
						messageLen,
						(SdsCallBack) SdsGoCallback,
						resp);
	}

	static void cGoSdsMarkDependenciesMet(void* rmCtx,
									char** messageIDs,
                    				size_t count,
									void* resp) {
		SdsMarkDependenciesMet(rmCtx,
						messageIDs,
						count,
						(SdsCallBack) SdsGoCallback,
						resp);
	}

	static void cGoSdsStartPeriodicTasks(void* rmCtx, void* resp) {
		SdsStartPeriodicTasks(rmCtx, (SdsCallBack) SdsGoCallback, resp);
	}

*/
import "C"
import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const requestTimeout = 30 * time.Second
const EventChanBufferSize = 1024

//export SdsGoCallback
func SdsGoCallback(ret C.int, msg *C.char, len C.size_t, resp unsafe.Pointer) {
	if resp != nil {
		m := (*C.SdsResp)(resp)
		m.ret = ret
		m.msg = msg
		m.len = len
		wg := (*sync.WaitGroup)(m.ffiWg)
		wg.Done()
	}
}

type EventCallbacks struct {
	OnMessageReady        func(messageId MessageID)
	OnMessageSent         func(messageId MessageID)
	OnMissingDependencies func(messageId MessageID, missingDeps []MessageID)
	OnPeriodicSync        func()
}

// ReliabilityManager represents an instance of a nim-sds ReliabilityManager
type ReliabilityManager struct {
	rmCtx     unsafe.Pointer
	channelId string
	callbacks EventCallbacks
}

func NewReliabilityManager(channelId string) (*ReliabilityManager, error) {
	Debug("Creating new Reliability Manager")
	rm := &ReliabilityManager{
		channelId: channelId,
	}

	wg := sync.WaitGroup{}

	var cChannelId = C.CString(string(channelId))
	var resp = C.allocResp(unsafe.Pointer(&wg))

	defer C.free(unsafe.Pointer(cChannelId))
	defer C.freeResp(resp)

	if C.getRet(resp) != C.RET_OK {
		errMsg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		Error("error NewReliabilityManager: %v", errMsg)
		return nil, errors.New(errMsg)
	}

	wg.Add(1)
	rm.rmCtx = C.cGoSdsNewReliabilityManager(cChannelId, resp)
	wg.Wait()

	C.cGoSdsSetEventCallback(rm.rmCtx)
	registerReliabilityManager(rm)

	Debug("Successfully created Reliability Manager")
	return rm, nil
}

// The event callback sends back the rm ctx to know to which
// rm is the event being emited for. Since we only have a global
// callback in the go side, We register all the rm's that we create
// so we can later obtain which instance of `ReliabilityManager` it should
// be invoked depending on the ctx received

var rmRegistry map[unsafe.Pointer]*ReliabilityManager

func init() {
	rmRegistry = make(map[unsafe.Pointer]*ReliabilityManager)
}

func registerReliabilityManager(rm *ReliabilityManager) {
	_, ok := rmRegistry[rm.rmCtx]
	if !ok {
		rmRegistry[rm.rmCtx] = rm
	}
}

func unregisterReliabilityManager(rm *ReliabilityManager) {
	delete(rmRegistry, rm.rmCtx)
}

//export sdsGlobalEventCallback
func sdsGlobalEventCallback(callerRet C.int, msg *C.char, len C.size_t, userData unsafe.Pointer) {
	if callerRet == C.RET_OK {
		eventStr := C.GoStringN(msg, C.int(len))
		rm, ok := rmRegistry[userData] // userData contains rm's ctx
		if ok {
			rm.OnEvent(eventStr)
		}
	} else {
		if len != 0 {
			errMsg := C.GoStringN(msg, C.int(len))
			Error("sdsGlobalEventCallback retCode not ok, retCode: %v: %v", callerRet, errMsg)
		} else {
			Error("sdsGlobalEventCallback retCode not ok, retCode: %v", callerRet)
		}
	}
}

type jsonEvent struct {
	EventType string `json:"eventType"`
}

type msgEvent struct {
	MessageId MessageID `json:"messageId"`
}

type missingDepsEvent struct {
	MessageId   MessageID   `json:"messageId"`
	MissingDeps []MessageID `json:"missingDeps"`
}

func (rm *ReliabilityManager) RegisterCallbacks(callbacks EventCallbacks) {
	rm.callbacks = callbacks
}

func (rm *ReliabilityManager) OnEvent(eventStr string) {

	jsonEvent := jsonEvent{}
	err := json.Unmarshal([]byte(eventStr), &jsonEvent)
	if err != nil {
		Error("could not unmarshal sds event string: %v", err)

		return
	}

	switch jsonEvent.EventType {
	case "message_ready":
		rm.parseMessageReadyEvent(eventStr)
	case "message_sent":
		rm.parseMessageSentEvent(eventStr)
	case "missing_dependencies":
		rm.parseMissingDepsEvent(eventStr)
	case "periodic_sync":
		if rm.callbacks.OnPeriodicSync != nil {
			rm.callbacks.OnPeriodicSync()
		}
	}

}

func (rm *ReliabilityManager) parseMessageReadyEvent(eventStr string) {

	msgEvent := msgEvent{}
	err := json.Unmarshal([]byte(eventStr), &msgEvent)
	if err != nil {
		Error("could not parse message ready event %v", err)
	}

	if rm.callbacks.OnMessageReady != nil {
		rm.callbacks.OnMessageReady(msgEvent.MessageId)
	}
}

func (rm *ReliabilityManager) parseMessageSentEvent(eventStr string) {

	msgEvent := msgEvent{}
	err := json.Unmarshal([]byte(eventStr), &msgEvent)
	if err != nil {
		Error("could not parse message sent event %v", err)
	}

	if rm.callbacks.OnMessageSent != nil {
		rm.callbacks.OnMessageSent(msgEvent.MessageId)
	}
}

func (rm *ReliabilityManager) parseMissingDepsEvent(eventStr string) {

	missingDepsEvent := missingDepsEvent{}
	err := json.Unmarshal([]byte(eventStr), &missingDepsEvent)
	if err != nil {
		Error("could not parse missing dependencies event %v", err)
	}

	if rm.callbacks.OnMissingDependencies != nil {
		rm.callbacks.OnMissingDependencies(missingDepsEvent.MessageId, missingDepsEvent.MissingDeps)
	}
}

func (rm *ReliabilityManager) Cleanup() error {
	if rm == nil {
		err := errors.New("reliability manager is nil in Cleanup")
		Error("Failed to cleanup %v", err)
		return err
	}

	Debug("Cleaning up reliability manager")

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoSdsCleanupReliabilityManager(rm.rmCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		unregisterReliabilityManager(rm)
		Debug("Successfully cleaned up reliability manager")
		return nil
	}

	errMsg := "error CleanupReliabilityManager: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to cleanup reliability manager: %v", errMsg)

	return errors.New(errMsg)
}

func (rm *ReliabilityManager) Reset() error {
	if rm == nil {
		err := errors.New("reliability manager is nil in Reset")
		Error("Failed to reset %v", err)
		return err
	}

	Debug("Resetting reliability manager")

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoSdsResetReliabilityManager(rm.rmCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		Debug("Successfully resetted reliability manager")
		return nil
	}

	errMsg := "error ResetReliabilityManager: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to reset reliability manager: %v", errMsg)

	return errors.New(errMsg)
}

func (rm *ReliabilityManager) WrapOutgoingMessage(message []byte, messageId MessageID) ([]byte, error) {
	if rm == nil {
		err := errors.New("reliability manager is nil in WrapOutgoingMessage")
		Error("Failed to wrap outgoing message %v", err)
		return nil, err
	}

	Debug("Wrapping outgoing message %v", messageId)

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	cMessageId := C.CString(string(messageId))
	defer C.free(unsafe.Pointer(cMessageId))

	var cMessagePtr unsafe.Pointer
	if len(message) > 0 {
		cMessagePtr = C.CBytes(message) // C.CBytes allocates memory that needs to be freed
		defer C.free(cMessagePtr)
	} else {
		cMessagePtr = nil
	}
	cMessageLen := C.size_t(len(message))

	wg.Add(1)
	C.cGoSdsWrapOutgoingMessage(rm.rmCtx, cMessagePtr, cMessageLen, cMessageId, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		resStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		if resStr == "" {
			Debug("Received empty res string for messageId: %v", messageId)
			return nil, nil
		}
		Debug("Successfully wrapped message %s", messageId)

		parts := strings.Split(resStr, ",")
		bytes := make([]byte, len(parts))

		for i, part := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				panic(err)
			}
			bytes[i] = byte(n)
		}

		return bytes, nil
	}

	errMsg := "error WrapOutgoingMessage: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to wrap message %v: %v", messageId, errMsg)

	return nil, errors.New(errMsg)
}

func (rm *ReliabilityManager) UnwrapReceivedMessage(message []byte) (*UnwrappedMessage, error) {
	if rm == nil {
		err := errors.New("reliability manager is nil in UnwrapReceivedMessage")
		Error("Failed to unwrap received message %v", err)
		return nil, err
	}

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	var cMessagePtr unsafe.Pointer
	if len(message) > 0 {
		cMessagePtr = C.CBytes(message) // C.CBytes allocates memory that needs to be freed
		defer C.free(cMessagePtr)
	} else {
		cMessagePtr = nil
	}
	cMessageLen := C.size_t(len(message))

	wg.Add(1)
	C.cGoSdsUnwrapReceivedMessage(rm.rmCtx, cMessagePtr, cMessageLen, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		resStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		if resStr == "" {
			Debug("Received empty res string")
			return nil, nil
		}
		Debug("Successfully unwrapped message")

		unwrappedMessage := UnwrappedMessage{}
		err := json.Unmarshal([]byte(resStr), &unwrappedMessage)
		if err != nil {
			Error("Failed to unmarshal unwrapped message")
			return nil, err
		}

		return &unwrappedMessage, nil
	}

	errMsg := "error UnwrapReceivedMessage: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to unwrap message: %v", errMsg)

	return nil, errors.New(errMsg)
}

// MarkDependenciesMet informs the library that dependencies are met
func (rm *ReliabilityManager) MarkDependenciesMet(messageIDs []MessageID) error {
	if rm == nil {
		err := errors.New("reliability manager is nil in MarkDependenciesMet")
		Error("Failed to mark dependencies met %v", err)
		return err
	}

	if len(messageIDs) == 0 {
		return nil // Nothing to do
	}

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	// Convert Go string slice to C array of C strings (char**)
	cMessageIDs := make([]*C.char, len(messageIDs))
	for i, id := range messageIDs {
		cMessageIDs[i] = C.CString(string(id))
		defer C.free(unsafe.Pointer(cMessageIDs[i])) // Ensure each CString is freed
	}

	// Create a pointer (**C.char) to the first element of the slice
	var cMessageIDsPtr **C.char
	if len(cMessageIDs) > 0 {
		cMessageIDsPtr = &cMessageIDs[0]
	} else {
		cMessageIDsPtr = nil // Handle empty slice case
	}

	wg.Add(1)
	// Pass the pointer variable (cMessageIDsPtr) directly, which is of type **C.char
	C.cGoSdsMarkDependenciesMet(rm.rmCtx, cMessageIDsPtr, C.size_t(len(messageIDs)), resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		Debug("Successfully marked dependencies as met")
		return nil
	}

	errMsg := "error MarkDependenciesMet: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to mark dependencies as met: %v", errMsg)

	return errors.New(errMsg)
}

func (rm *ReliabilityManager) StartPeriodicTasks() error {
	if rm == nil {
		err := errors.New("reliability manager is nil in StartPeriodicTasks")
		Error("Failed to start periodic tasks %v", err)
		return err
	}

	Debug("Starting periodic tasks")

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoSdsStartPeriodicTasks(rm.rmCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		Debug("Successfully started periodic tasks")
		return nil
	}

	errMsg := "error StartPeriodicTasks: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to start periodic tasks: %v", errMsg)

	return errors.New(errMsg)
}
