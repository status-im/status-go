package protocol

import (
	"errors"
	"sync"

	"github.com/status-im/status-go/services/mailservers"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
)

var (
	ErrNotFound = errors.New("not found")
)

// communityStoreNodes has methods to handle the mailservers for a community
// including the active mailservers and the list of mailservers
type communityStoreNodes struct {
	storenodesByCommunityIDMutex *sync.RWMutex
	storenodesByCommunityID      map[string]storenodesData
	storenodesDatabase           *mailserversDB.Database
}

type storenodesData struct {
	// TODO for now we support only one mailserver per community, we will assume it is always active,
	// then we will need to support a way to regularly check connection similar to the `messenger_mailserver_cycle.go`
	storenodes []mailservers.Mailserver
}

// GetStorenodeByCommunnityID returns the active storenode for a community
func (m *communityStoreNodes) GetStorenodeByCommunnityID(communityID string) (mailservers.Mailserver, error) {
	m.storenodesByCommunityIDMutex.RLock()
	defer m.storenodesByCommunityIDMutex.RUnlock()

	msData, ok := m.storenodesByCommunityID[communityID]
	if !ok || len(msData.storenodes) == 0 {
		return mailservers.Mailserver{}, ErrNotFound
	}
	return msData.storenodes[0], nil
}

func (m *communityStoreNodes) HasStorenodeSetup(communityID string) bool {
	m.storenodesByCommunityIDMutex.RLock()
	defer m.storenodesByCommunityIDMutex.RUnlock()

	msData, ok := m.storenodesByCommunityID[communityID]
	return ok && len(msData.storenodes) > 0
}

// ReloadFromDB loads or reloads the mailservers from the database (on adding/deleting mailservers)
func (m *communityStoreNodes) ReloadFromDB() error {
	if m.storenodesDatabase == nil {
		return nil
	}
	m.storenodesByCommunityIDMutex.RLock()
	defer m.storenodesByCommunityIDMutex.RUnlock()
	dbMailservers, err := m.storenodesDatabase.GetMailserversForCommunities()
	if err != nil {
		return err
	}
	// overwrite the in-memory mailservers
	m.storenodesByCommunityID = make(map[string]storenodesData)
	for communityID, mailservers := range dbMailservers {
		m.storenodesByCommunityID[communityID] = storenodesData{
			storenodes: mailservers,
		}
	}
	return nil
}
