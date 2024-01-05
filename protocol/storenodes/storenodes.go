package storenodes

import (
	"errors"
	"sync"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/mailservers"
)

var (
	ErrNotFound = errors.New("not found")
)

// CommunityStorenodes has methods to handle the storenodes for a community
type CommunityStorenodes struct {
	storenodesByCommunityIDMutex *sync.RWMutex
	storenodesByCommunityID      map[string]storenodesData

	storenodesDB *Database
}

func NewCommunityStorenodes(storenodesDB *Database) *CommunityStorenodes {
	return &CommunityStorenodes{
		storenodesByCommunityIDMutex: &sync.RWMutex{},
		storenodesByCommunityID:      make(map[string]storenodesData),
		storenodesDB:                 storenodesDB,
	}
}

type storenodesData struct {
	storenodes []Storenode
}

// GetStorenodeByCommunnityID returns the active storenode for a community
func (m *CommunityStorenodes) GetStorenodeByCommunnityID(communityID string) (mailservers.Mailserver, error) {
	m.storenodesByCommunityIDMutex.RLock()
	defer m.storenodesByCommunityIDMutex.RUnlock()

	msData, ok := m.storenodesByCommunityID[communityID]
	if !ok || len(msData.storenodes) == 0 {
		return mailservers.Mailserver{}, ErrNotFound
	}
	return toMailserver(msData.storenodes[0]), nil
}

func (m *CommunityStorenodes) HasStorenodeSetup(communityID string) bool {
	m.storenodesByCommunityIDMutex.RLock()
	defer m.storenodesByCommunityIDMutex.RUnlock()

	msData, ok := m.storenodesByCommunityID[communityID]
	return ok && len(msData.storenodes) > 0
}

// ReloadFromDB loads or reloads the mailservers from the database (on adding/deleting mailservers)
func (m *CommunityStorenodes) ReloadFromDB() error {
	if m.storenodesDB == nil {
		return nil
	}
	m.storenodesByCommunityIDMutex.Lock()
	defer m.storenodesByCommunityIDMutex.Unlock()
	dbNodes, err := m.storenodesDB.getAll()
	if err != nil {
		return err
	}
	// overwrite the in-memory storenodes
	m.storenodesByCommunityID = make(map[string]storenodesData)
	for _, node := range dbNodes {
		communityID := node.CommunityID.String()
		if _, ok := m.storenodesByCommunityID[communityID]; !ok {
			m.storenodesByCommunityID[communityID] = storenodesData{}
		}
		data := m.storenodesByCommunityID[communityID]
		data.storenodes = append(data.storenodes, node)
		m.storenodesByCommunityID[communityID] = data
	}
	return nil
}

func (m *CommunityStorenodes) UpdateStorenodesInDB(communityID types.HexBytes, snodes []Storenode, clock uint64) error {
	if err := m.storenodesDB.syncSave(communityID, snodes, clock); err != nil {
		return err
	}
	if err := m.ReloadFromDB(); err != nil {
		return err
	}
	return nil
}

func (m *CommunityStorenodes) GetStorenodesFromDB(communityID types.HexBytes) ([]Storenode, error) {
	snodes, err := m.storenodesDB.getByCommunityID(communityID)
	if err != nil {
		return nil, err
	}
	return snodes, nil
}
