package community

import (
	"database/sql"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// These events are used to notify the UI of state changes
const (
	EventCommmunityDataUpdated walletevent.EventType = "wallet-community-data-updated"
)

type Manager struct {
	db                    *DataDB
	communityInfoProvider thirdparty.CommunityInfoProvider
	mediaServer           *server.MediaServer
	feed                  *event.Feed
}

type Data struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Image string `json:"image,omitempty"`
}

func NewManager(db *sql.DB, mediaServer *server.MediaServer, feed *event.Feed) *Manager {
	return &Manager{
		db:          NewDataDB(db),
		mediaServer: mediaServer,
		feed:        feed,
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
		communityInfo.CommunityImage = cm.GetCommunityImageURL(id)
	}
	return communityInfo, state, err
}

func (cm *Manager) GetCommunityID(tokenURI string) string {
	return cm.communityInfoProvider.GetCommunityID(tokenURI)
}

func (cm *Manager) FillCollectiblesMetadata(communityID string, cs []*thirdparty.FullCollectibleData) (bool, error) {
	return cm.communityInfoProvider.FillCollectiblesMetadata(communityID, cs)
}

func (cm *Manager) setCommunityInfo(id string, c *thirdparty.CommunityInfo) (err error) {
	return cm.db.SetCommunityInfo(id, c)
}

func (cm *Manager) fetchCommunityInfo(communityID string, fetcher func() (*thirdparty.CommunityInfo, error)) (*thirdparty.CommunityInfo, error) {
	communityInfo, err := fetcher()
	if err != nil {
		dbErr := cm.setCommunityInfo(communityID, nil)
		if dbErr != nil {
			logutils.ZapLogger().Error("SetCommunityInfo failed", zap.String("communityID", communityID), zap.Error(dbErr))
		}
		return nil, err
	}
	err = cm.setCommunityInfo(communityID, communityInfo)
	return communityInfo, err
}

func (cm *Manager) FetchCommunityInfo(communityID string) (*thirdparty.CommunityInfo, error) {
	return cm.fetchCommunityInfo(communityID, func() (*thirdparty.CommunityInfo, error) {
		return cm.communityInfoProvider.FetchCommunityInfo(communityID)
	})
}

func (cm *Manager) FetchCommunityMetadataAsync(communityID string) {
	go func() {
		defer gocommon.LogOnPanic()
		communityInfo, err := cm.FetchCommunityMetadata(communityID)
		if err != nil {
			logutils.ZapLogger().Error("FetchCommunityInfo failed", zap.String("communityID", communityID), zap.Error(err))
		}
		cm.signalUpdatedCommunityMetadata(communityID, communityInfo)
	}()
}

func (cm *Manager) FetchCommunityMetadata(communityID string) (*thirdparty.CommunityInfo, error) {
	communityInfo, err := cm.FetchCommunityInfo(communityID)
	if err != nil {
		return nil, err
	}
	_ = cm.setCommunityInfo(communityID, communityInfo)
	return communityInfo, err
}

func (cm *Manager) GetCommunityImageURL(communityID string) string {
	if cm.mediaServer != nil {
		return cm.mediaServer.MakeWalletCommunityImagesURL(communityID)
	}
	return ""
}

func (cm *Manager) signalUpdatedCommunityMetadata(communityID string, communityInfo *thirdparty.CommunityInfo) {
	if communityInfo == nil {
		return
	}
	data := Data{
		ID:    communityID,
		Name:  communityInfo.CommunityName,
		Color: communityInfo.CommunityColor,
		Image: cm.GetCommunityImageURL(communityID),
	}

	payload, err := json.Marshal(data)
	if err != nil {
		logutils.ZapLogger().Error("Error marshaling response", zap.Error(err))
		return
	}

	event := walletevent.Event{
		Type:    EventCommmunityDataUpdated,
		Message: string(payload),
	}

	cm.feed.Send(event)
}
