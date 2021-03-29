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

	globalLock *sync.Mutex
	globalWait *sync.WaitGroup

	granularLocks map[string]*sync.Mutex
}

func NewMessengerLocker() *MessengerLocker {
	return &MessengerLocker{
		accessLock:    new(sync.Mutex),
		globalLock:    new(sync.Mutex),
		globalWait:    new(sync.WaitGroup),
		granularLocks: map[string]*sync.Mutex{},
	}
}

func (ml *MessengerLocker) Get(id string) *sync.Mutex {
	ml.globalWait.Wait() // Blocks until globalLock is released

	ml.accessLock.Lock()
	defer ml.accessLock.Unlock()

	if cl, ok := ml.granularLocks[id]; ok {
		return cl
	}

	ml.granularLocks[id] = new(sync.Mutex)
	return ml.granularLocks[id]
}

func (ml *MessengerLocker) Lock() {
	ml.globalLock.Lock() // Protects globalWait from new global.Lock()s
	ml.globalWait.Add(1)
}

func (ml *MessengerLocker) Unlock() {
	ml.globalLock.Unlock()
	ml.globalWait.Done()
}
