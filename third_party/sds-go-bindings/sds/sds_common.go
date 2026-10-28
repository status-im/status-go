package sds

import (
	"encoding/json"
	"errors"
	"sync"

	"go.uber.org/zap"
)

const EventChanBufferSize = 1024

type EventCallbacks struct {
	OnMessageReady        func(messageId MessageID, channelId string)
	OnMessageSent         func(messageId MessageID, channelId string)
	OnMissingDependencies func(messageId MessageID, missingDeps []HistoryEntry, channelId string)
	OnPeriodicSync        func()
	OnRepairReady         func(message []byte, channelId string)
	RetrievalHintProvider func(messageId MessageID) []byte
}

// ReliabilityManager represents an instance of a nim-sds ReliabilityManager
type ReliabilityManager struct {
	logger      *zap.Logger
	rmCtx       uintptr
	callbacks   EventCallbacks
	persistence PersistenceCallbacks
}

func (p PersistenceCallbacks) enabled() bool {
	return p.SaveChannelMeta != nil ||
		p.UpdateHistory != nil ||
		p.LoadChannel != nil ||
		p.DropChannel != nil
}

func (p PersistenceCallbacks) validate() error {
	if !p.enabled() {
		return nil
	}
	if p.SaveChannelMeta == nil || p.UpdateHistory == nil || p.LoadChannel == nil || p.DropChannel == nil {
		return errors.New("all SDS persistence callbacks must be provided")
	}
	return nil
}

// The event callback sends back the rm ctx to know to which
// rm is the event being emited for. Since we only have a global
// callback in the go side, We register all the rm's that we create
// so we can later obtain which instance of `ReliabilityManager` it should
// be invoked depending on the ctx received
var rmRegistry map[uintptr]*ReliabilityManager

// rmRegistryMu guards rmRegistry. libsds invokes the global callbacks on its
// own worker/event threads, so reads from those callbacks race the register/
// unregister writes done on the goroutines that create and clean up managers.
var rmRegistryMu sync.RWMutex

func init() {
	rmRegistry = make(map[uintptr]*ReliabilityManager)
}

// lookupReliabilityManager resolves the manager for a ctx under the read lock.
// Used by the global callbacks, which run on libsds threads.
func lookupReliabilityManager(ctx uintptr) (*ReliabilityManager, bool) {
	rmRegistryMu.RLock()
	defer rmRegistryMu.RUnlock()
	rm, ok := rmRegistry[ctx]
	return rm, ok
}

func registerReliabilityManager(rm *ReliabilityManager) {
	rmRegistryMu.Lock()
	defer rmRegistryMu.Unlock()
	_, ok := rmRegistry[rm.rmCtx]
	if !ok {
		rmRegistry[rm.rmCtx] = rm
	}
}

func unregisterReliabilityManager(rm *ReliabilityManager) {
	rmRegistryMu.Lock()
	defer rmRegistryMu.Unlock()
	delete(rmRegistry, rm.rmCtx)
}

type jsonEvent struct {
	EventType string `json:"eventType"`
}

type msgEvent struct {
	MessageId MessageID `json:"messageId"`
	ChannelId string    `json:"channelId"`
}

type missingDepsEvent struct {
	MessageId   MessageID      `json:"messageId"`
	MissingDeps []HistoryEntry `json:"missingDeps"`
	ChannelId   string         `json:"channelId"`
}

// repairReadyEvent mirrors libsds' repair_ready JSON payload. nim-sds base64-
// encodes the message bytes, which encoding/json decodes back into []byte.
type repairReadyEvent struct {
	Message   []byte `json:"message"`
	ChannelId string `json:"channelId"`
}

func (rm *ReliabilityManager) RegisterCallbacks(callbacks EventCallbacks) {
	rm.callbacks = callbacks
}

func (rm *ReliabilityManager) OnEvent(eventStr string) {
	jsonEvent := jsonEvent{}
	err := json.Unmarshal([]byte(eventStr), &jsonEvent)
	if err != nil {
		rm.logger.Error("failed to unmarshal sds event string", zap.Error(err))
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
	case "repair_ready":
		rm.parseRepairReadyEvent(eventStr)
	}
}

func (rm *ReliabilityManager) OnCallbackError(callerRet int, err string) {
	rm.logger.Error("sds callback error",
		zap.Int("retCode", callerRet),
		zap.String("errMsg", err))
}

func (rm *ReliabilityManager) parseMessageReadyEvent(eventStr string) {
	msgEvent := msgEvent{}
	err := json.Unmarshal([]byte(eventStr), &msgEvent)
	if err != nil {
		rm.logger.Error("failed to parse message ready event", zap.Error(err))
	}

	if rm.callbacks.OnMessageReady != nil {
		rm.callbacks.OnMessageReady(msgEvent.MessageId, msgEvent.ChannelId)
	}
}

func (rm *ReliabilityManager) parseMessageSentEvent(eventStr string) {
	msgEvent := msgEvent{}
	err := json.Unmarshal([]byte(eventStr), &msgEvent)
	if err != nil {
		rm.logger.Error("failed to parse message sent event", zap.Error(err))
		return
	}

	if rm.callbacks.OnMessageSent != nil {
		rm.callbacks.OnMessageSent(msgEvent.MessageId, msgEvent.ChannelId)
	}
}

func (rm *ReliabilityManager) parseMissingDepsEvent(eventStr string) {
	missingDepsEvent := missingDepsEvent{}
	err := json.Unmarshal([]byte(eventStr), &missingDepsEvent)
	if err != nil {
		rm.logger.Error("failed to parse missing dependencies event", zap.Error(err))
		return
	}

	if rm.callbacks.OnMissingDependencies != nil {
		rm.callbacks.OnMissingDependencies(missingDepsEvent.MessageId, missingDepsEvent.MissingDeps, missingDepsEvent.ChannelId)
	}
}

func (rm *ReliabilityManager) parseRepairReadyEvent(eventStr string) {
	repairReadyEvent := repairReadyEvent{}
	err := json.Unmarshal([]byte(eventStr), &repairReadyEvent)
	if err != nil {
		rm.logger.Error("failed to parse repair ready event", zap.Error(err))
		return
	}

	if rm.callbacks.OnRepairReady != nil {
		rm.callbacks.OnRepairReady(repairReadyEvent.Message, repairReadyEvent.ChannelId)
	}
}
