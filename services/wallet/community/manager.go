package community

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type Manager struct {
	db                    *DataDB
	communityInfoProvider thirdparty.CommunityInfoProvider
	mediaServer           *server.MediaServer
}

func NewManager(db *sql.DB, mediaServer *server.MediaServer) *Manager {
	return &Manager{
		db:          NewDataDB(db),
		mediaServer: mediaServer,
	}
}

// Used to break circular dependency, call once as soon as possible after initialization
func (cm *Manager) SetCommunityInfoProvider(communityInfoProvider thirdparty.CommunityInfoProvider) {
	cm.communityInfoProvider = communityInfoProvider
}

func (cm *Manager) GetCommunityInfo(id string) (*thirdparty.CommunityInfo, *InfoState, error) {
	communityInfo, state, err := cm.db.GetCommunityInfo(id)
	if err != nil {
		return nil, nil, err
	}
	if cm.mediaServer != nil && communityInfo != nil && len(communityInfo.CommunityImagePayload) > 0 {
		communityInfo.CommunityImage = cm.mediaServer.MakeWalletCommunityImagesURL(id)
	}
	return communityInfo, state, err
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

func (cm *Manager) FetchCommunityInfo(communityID string) (*thirdparty.CommunityInfo, error) {
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
