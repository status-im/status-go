package protocol

// TODO LOTS OF TESTS
// TODO Perhaps a smarter approach that segregates db calls from memory access
//  this could allow for smarter locks on specific read/write of Messenger fields
//  much depends on what we are trying to protect.

import (
	"sync"

	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
)

const (
	DatabaseContacts           = "db_contacts"
	MessengerInstallations     = "m_installations"
	MessengerPushNotifications = "m_push_notifications"
)

type MessengerLocker struct {
	accessLock *sync.Mutex
	accessWait *sync.WaitGroup

	globalLock *sync.Mutex
	globalWait *sync.WaitGroup

	granularLocks map[string]*sync.Mutex
}

func NewMessengerLocker() *MessengerLocker {
	return &MessengerLocker{
		accessLock:    new(sync.Mutex),
		accessWait:    new(sync.WaitGroup),
		globalLock:    new(sync.Mutex),
		globalWait:    new(sync.WaitGroup),
		granularLocks: map[string]*sync.Mutex{},
	}
}

func (ml *MessengerLocker) Get(id string) *sync.Mutex {
	ml.globalWait.Wait() // Blocks until globalLock is released

	ml.accessLock.Lock() // Only allows one granularLock to be issued at a time, also prevents accessWait delta > 1
	defer ml.accessLock.Unlock()
	ml.accessWait.Add(1) // Blocks a globalLock until a granular lock has been issued
	defer ml.accessWait.Done()

	if cl, ok := ml.granularLocks[id]; ok {
		return cl
	}

	ml.granularLocks[id] = new(sync.Mutex)
	return ml.granularLocks[id]
}

func (ml *MessengerLocker) Lock() {
	ml.accessWait.Wait() // Blocks globalLock if a granular lock is currently being issued
	ml.globalLock.Lock() // Protects globalWait from new global.Lock()s
	ml.globalWait.Add(1) // Blocks granularLocks from access until global unlock
}

func (ml *MessengerLocker) Unlock() {
	ml.globalLock.Unlock()
	ml.globalWait.Done()
}

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
	return chat.(*Chat), ok
}

func (cm *chatMap) Store(chatID string, chat *Chat) {
	cm.sm.Store(chatID, chat)
}

func (cm *chatMap) Range(f func(chatID string, chat *Chat) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool){
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
	sm sync.Map
}

func (cm *contactMap) Load(contactID string) (*Contact, bool) {
	contact, ok := cm.sm.Load(contactID)
	return contact.(*Contact), ok
}

func (cm *contactMap) Store(contactID string, contact *Contact) {
	cm.sm.Store(contactID, contact)
}

func (cm *contactMap) Range(f func(contactID string, contact *Contact) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool){
		return f(key.(string), value.(*Contact))
	}
	cm.sm.Range(nf)
}

func (cm *contactMap) Delete(contactID string) {
	cm.sm.Delete(contactID)
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
	return message.(string), ok
}

func (smtm *systemMessageTranslationsMap) Store(eventType protobuf.MembershipUpdateEvent_EventType, message string) {
	smtm.sm.Store(eventType, message)
}

func (smtm *systemMessageTranslationsMap) Range(f func(eventType protobuf.MembershipUpdateEvent_EventType, message string) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool){
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
	return installation.(*multidevice.Installation), ok
}

func (im *installationMap) Store(installationID string, installation *multidevice.Installation) {
	im.sm.Store(installationID, installation)
}

func (im *installationMap) Range(f func(installationID string, installation *multidevice.Installation) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool){
		return f(key.(string), value.(*multidevice.Installation))
	}
	im.sm.Range(nf)
}

func (im *installationMap) Delete(installationID string) {
	im.sm.Delete(installationID)
}

func (im *installationMap) Empty() bool {
	count := 0
	im.Range(func(installationID string, installation *multidevice.Installation) (shouldContinue bool){
		count++
		return false
	})

	return count == 0
}

func (im *installationMap) Len() int {
	count := 0
	im.Range(func(installationID string, installation *multidevice.Installation) (shouldContinue bool){
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
	return state.(bool), ok
}

func (sbm *stringBoolMap) Store(key string, value bool) {
	sbm.sm.Store(key, value)
}

func (sbm *stringBoolMap) Range(f func(key string, value bool) (shouldContinue bool)) {
	nf := func(key, value interface{}) (shouldContinue bool){
		return f(key.(string), value.(bool))
	}
	sbm.sm.Range(nf)
}

func (sbm *stringBoolMap) Delete(key string) {
	sbm.sm.Delete(key)
}
