package protocol

import (
	"fmt"
	"go.uber.org/zap"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
)

/*
|--------------------------------------------------------------------------
| chatMap
|--------------------------------------------------------------------------
|
| A sync.Map wrapper for a specific mapping of map[string]*Chat
|
*/

type chatMap struct {
	sm sync.Map
}

func (cm *chatMap) Load(chatID string) (*Chat, bool) {
	chat, ok := cm.sm.Load(chatID)
	if chat == nil {
		return nil, ok
	}
	return chat.(*Chat), ok
}

func (cm *chatMap) Store(chatID string, chat *Chat) {
	cm.sm.Store(chatID, chat)
}

func (cm *chatMap) Range(f func(chatID string, chat *Chat) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool) {
		return f(key.(string), value.(*Chat))
	}
	cm.sm.Range(nf)
}

func (cm *chatMap) Delete(chatID string) {
	cm.sm.Delete(chatID)
}

/*
|--------------------------------------------------------------------------
| contactMap
|--------------------------------------------------------------------------
|
| A sync.Map wrapper for a specific mapping of map[string]*Contact
|
*/

type contactMap struct {
	sm     sync.Map
	me     *Contact
	logger *zap.Logger
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func printStack(stack []byte) {
	const start = 1 // skip top 2 lines (Stack() and Load()/Store() calls)
	const depth = 3
	stackString := string(stack[:])
	stackArray := strings.Split(stackString, "\n")
	stackArray = stackArray[1 : len(stackArray)-1] // drop "goroutine X [running]:" and last empty line
	var files []string
	var funcs []string
	for i, v := range stackArray {
		l := strings.TrimSpace(v)
		if i%2 == 0 {
			funcs = append(funcs, l)
		} else {
			files = append(files, l)
		}
	}
	end := len(funcs)
	//end := min(start + depth, len(funcs) - 1)
	var maxFuncLen = 0
	for i := start; i < end; i++ {
		if maxFuncLen < len(funcs[i]) {
			maxFuncLen = len(funcs[i])
		}
	}
	for i := start; i < end; i++ {
		fmt.Printf("| %-*s\t%s\n", maxFuncLen, funcs[i], files[i])
	}
}

type StackItem struct {
	File     string `json:"file"`
	Function string `json:"function"`
}

func stackItems(stack []byte) []*StackItem {
	stackString := string(stack[:])
	stackArray := strings.Split(stackString, "\n")
	stackArray = stackArray[1 : len(stackArray)-1] // drop "goroutine X [running]:" and last empty line
	var items []*StackItem
	for i := 0; i < len(stackArray); i += 2 {
		items = append(items, &StackItem{
			Function: strings.TrimSpace(stackArray[i]),
			File:     strings.TrimSpace(stackArray[i+1]),
		})
	}
	return items
}

func (cm *contactMap) Load(contactID string) (*Contact, bool) {
	if contactID == cm.me.ID {
		stack := debug.Stack()
		cm.logger.Info("contacts map: loading own identity", zap.String("contactID", contactID), zap.Any("stack", stackItems(stack)))
		return cm.me, true
	}
	contact, ok := cm.sm.Load(contactID)
	if contact == nil {
		return nil, ok
	}
	return contact.(*Contact), ok
}

func (cm *contactMap) Store(contactID string, contact *Contact) {
	if contactID == cm.me.ID {
		stack := debug.Stack()
		cm.logger.Info("contacts map: storing own identity", zap.String("contactID", contactID), zap.Any("stack", stackItems(stack)))
		return
	}
	cm.sm.Store(contactID, contact)
}

func (cm *contactMap) Range(f func(contactID string, contact *Contact) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool) {
		return f(key.(string), value.(*Contact))
	}
	cm.sm.Range(nf)
}

func (cm *contactMap) Delete(contactID string) {
	if contactID == cm.me.ID {
		cm.logger.Warn("contacts map: deleting own identity", zap.String("contactID", contactID))
		return
	}
	cm.sm.Delete(contactID)
}

func (cm *contactMap) Len() int {
	count := 0
	cm.Range(func(contactID string, contact *Contact) (shouldContinue bool) {
		count++
		return true
	})

	return count
}

/*
|--------------------------------------------------------------------------
| systemMessageTranslationsMap
|--------------------------------------------------------------------------
|
| A sync.Map wrapper for the specific mapping of map[protobuf.MembershipUpdateEvent_EventType]string
|
*/

type systemMessageTranslationsMap struct {
	sm sync.Map
}

func (smtm *systemMessageTranslationsMap) Init(set map[protobuf.MembershipUpdateEvent_EventType]string) {
	for eventType, message := range set {
		smtm.Store(eventType, message)
	}
}

func (smtm *systemMessageTranslationsMap) Load(eventType protobuf.MembershipUpdateEvent_EventType) (string, bool) {
	message, ok := smtm.sm.Load(eventType)
	if message == nil {
		return "", ok
	}
	return message.(string), ok
}

func (smtm *systemMessageTranslationsMap) Store(eventType protobuf.MembershipUpdateEvent_EventType, message string) {
	smtm.sm.Store(eventType, message)
}

func (smtm *systemMessageTranslationsMap) Range(f func(eventType protobuf.MembershipUpdateEvent_EventType, message string) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool) {
		return f(key.(protobuf.MembershipUpdateEvent_EventType), value.(string))
	}
	smtm.sm.Range(nf)
}

func (smtm *systemMessageTranslationsMap) Delete(eventType protobuf.MembershipUpdateEvent_EventType) {
	smtm.sm.Delete(eventType)
}

/*
|--------------------------------------------------------------------------
| installationMap
|--------------------------------------------------------------------------
|
| A sync.Map wrapper for the specific mapping of map[string]*multidevice.Installation
|
*/

type installationMap struct {
	sm sync.Map
}

func (im *installationMap) Load(installationID string) (*multidevice.Installation, bool) {
	installation, ok := im.sm.Load(installationID)
	if installation == nil {
		return nil, ok
	}
	return installation.(*multidevice.Installation), ok
}

func (im *installationMap) Store(installationID string, installation *multidevice.Installation) {
	im.sm.Store(installationID, installation)
}

func (im *installationMap) Range(f func(installationID string, installation *multidevice.Installation) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool) {
		return f(key.(string), value.(*multidevice.Installation))
	}
	im.sm.Range(nf)
}

func (im *installationMap) Delete(installationID string) {
	im.sm.Delete(installationID)
}

func (im *installationMap) Empty() bool {
	count := 0
	im.Range(func(installationID string, installation *multidevice.Installation) (shouldContinue bool) {
		count++
		return false
	})

	return count == 0
}

func (im *installationMap) Len() int {
	count := 0
	im.Range(func(installationID string, installation *multidevice.Installation) (shouldContinue bool) {
		count++
		return true
	})

	return count
}

/*
|--------------------------------------------------------------------------
| stringBoolMap
|--------------------------------------------------------------------------
|
| A sync.Map wrapper for the specific mapping of map[string]bool
|
*/

type stringBoolMap struct {
	sm sync.Map
}

func (sbm *stringBoolMap) Load(key string) (bool, bool) {
	state, ok := sbm.sm.Load(key)
	if state == nil {
		return false, ok
	}
	return state.(bool), ok
}

func (sbm *stringBoolMap) Store(key string, value bool) {
	sbm.sm.Store(key, value)
}

func (sbm *stringBoolMap) Range(f func(key string, value bool) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool) {
		return f(key.(string), value.(bool))
	}
	sbm.sm.Range(nf)
}

func (sbm *stringBoolMap) Delete(key string) {
	sbm.sm.Delete(key)
}

func (sbm *stringBoolMap) Len() int {
	count := 0
	sbm.Range(func(key string, value bool) (shouldContinue bool) {
		count++
		return true
	})

	return count
}
