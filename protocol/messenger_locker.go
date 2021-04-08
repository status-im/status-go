package protocol

// TODO LOTS OF TESTS
// TODO Perhaps a smarter approach that segregates db calls from memory access
//  this could allow for smarter locks on specific read/write of Messenger fields
//  much depends on what we are trying to protect.

import (
	"sync"
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

type contactMap struct {
	sm sync.Map
}

func (cm *contactMap) Load(contactID string) (*Contact, bool) {
	chat, ok := cm.sm.Load(contactID)
	return chat.(*Contact), ok
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

