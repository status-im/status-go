package community

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const failedCommunityFetchRetryDelay = 1 * time.Hour

type Manager struct {
	db                    *DataDB
	communityInfoProvider thirdparty.CommunityInfoProvider
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db: NewDataDB(db),
	}
}

// Used to break circular dependency, call once as soon as possible after initialization
func (cm *Manager) SetCommunityInfoProvider(communityInfoProvider thirdparty.CommunityInfoProvider) {
	cm.communityInfoProvider = communityInfoProvider
}

func (cm *Manager) GetCommunityInfo(id string) (*thirdparty.CommunityInfo, *InfoState, error) {
	return cm.db.GetCommunityInfo(id)
}

func (cm *Manager) GetCommunityID(tokenURI string) string {
	return cm.communityInfoProvider.GetCommunityID(tokenURI)
}

func (cm *Manager) FillCollectibleMetadata(c *thirdparty.FullCollectibleData) error {
	return cm.communityInfoProvider.FillCollectibleMetadata(c)
}

func (cm *Manager) setCommunityInfo(id string, c *thirdparty.CommunityInfo) (err error) {
	return cm.db.SetCommunityInfo(id, c)
}

func (cm *Manager) mustFetchCommunityInfo(communityID string) bool {
	// See if we have cached data
	_, state, err := cm.GetCommunityInfo(communityID)
	if err != nil {
		return true
	}

	// If we don't have a state, this community has never been fetched before
	if state == nil {
		return true
	}

	// If the last fetch was successful, we can safely refresh our cache
	if state.LastUpdateSuccesful {
		return true
	}

	// If the last fetch was not successful, we should only retry after a delay
	if time.Unix(int64(state.LastUpdateTimestamp), 0).Add(failedCommunityFetchRetryDelay).Before(time.Now()) {
		return true
	}

	return false
}

func (cm *Manager) FetchCommunityInfo(communityID string) (*thirdparty.CommunityInfo, error) {
	if !cm.mustFetchCommunityInfo(communityID) {
		return nil, fmt.Errorf("backing off fetchCommunityInfo for id: %s", communityID)
	}

	communityInfo, err := cm.communityInfoProvider.FetchCommunityInfo(communityID)
	if err != nil {
		dbErr := cm.setCommunityInfo(communityID, nil)
		if dbErr != nil {
			log.Error("SetCommunityInfo failed", "communityID", communityID, "err", dbErr)
		}
		return nil, err
	}

	err = cm.setCommunityInfo(communityID, communityInfo)
	return communityInfo, err
}
