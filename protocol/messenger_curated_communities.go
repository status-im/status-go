package protocol

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto/types"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/services/wallet/common"
)

const (
	curatedCommunitiesUpdateInterval = time.Hour
	communitiesUpdateFailureInterval = time.Minute
)

// Regularly gets list of curated communities and signals them to client
func (m *Messenger) startCuratedCommunitiesUpdateLoop() {
	logger := m.logger.Named("curatedCommunitiesUpdateLoop")

	if m.contractMaker == nil {
		logger.Warn("not starting curated communities loop: contract maker not initialized")
		return
	}

	go func() {
		defer gocommon.LogOnPanic()
		// Initialize interval to 0 for immediate execution
		var interval time.Duration = 0

		cache, err := m.communitiesManager.GetCuratedCommunities()
		if err != nil {
			logger.Error("failed to start curated communities loop", zap.Error(err))
			return
		}

		for {
			select {
			case <-time.After(interval):
				if m.shouldPauseCuratedCommunitiesUpdateLoop() {
					interval = curatedCommunitiesUpdateInterval
					continue
				}
				// Immediate execution on first run, then set to regular interval
				interval = curatedCommunitiesUpdateInterval

				logger.Debug("updating curated communities")
				curatedCommunities, err := m.getCuratedCommunitiesFromContract()
				if err != nil {
					interval = communitiesUpdateFailureInterval
					logger.Error("failed to get curated communities from contract", zap.Error(err))
					continue
				}

				if reflect.DeepEqual(cache.ContractCommunities, curatedCommunities.ContractCommunities) &&
					reflect.DeepEqual(cache.ContractFeaturedCommunities, curatedCommunities.ContractFeaturedCommunities) {
					// nothing changed
					continue
				}

				err = m.communitiesManager.SetCuratedCommunities(curatedCommunities)
				if err == nil {
					cache = curatedCommunities
				} else {
					logger.Error("failed to save curated communities", zap.Error(err))
				}

				response, err := m.fetchCuratedCommunities(curatedCommunities)
				if err != nil {
					interval = communitiesUpdateFailureInterval
					logger.Error("failed to fetch curated communities", zap.Error(err))
					continue
				}

				m.config.messengerSignalsHandler.SendCuratedCommunitiesUpdate(response)

			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) shouldPauseCuratedCommunitiesUpdateLoop() bool {
	// TODO when we implement back the setting for the user to select if they want to
	// fetch on expensive networks, use canSyncWithStoreNodes()
	// https://github.com/status-im/status-app/issues/18388
	return m.isPaused() || m.getConnectionState().IsExpensive()
}

func (m *Messenger) getCuratedCommunitiesFromContract() (*communities.CuratedCommunities, error) {
	if m.contractMaker == nil {
		return nil, errors.New("contract maker not initialized")
	}

	testNetworksEnabled, err := m.settings.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	chainID := common.OptimismMainnet
	if testNetworksEnabled {
		chainID = common.OptimismSepolia
	}

	directory, err := m.contractMaker.NewDirectory(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

	contractCommunities, err := directory.GetCommunities(callOpts)
	if err != nil {
		return nil, err
	}
	var contractCommunityIDs []string
	for _, c := range contractCommunities {
		contractCommunityIDs = append(contractCommunityIDs, types.HexBytes(c).String())
	}

	featuredContractCommunities, err := directory.GetFeaturedCommunities(callOpts)
	if err != nil {
		return nil, err
	}
	var contractFeaturedCommunityIDs []string
	for _, c := range featuredContractCommunities {
		contractFeaturedCommunityIDs = append(contractFeaturedCommunityIDs, types.HexBytes(c).String())
	}

	return &communities.CuratedCommunities{
		ContractCommunities:         contractCommunityIDs,
		ContractFeaturedCommunities: contractFeaturedCommunityIDs,
	}, nil
}

func (m *Messenger) fetchCuratedCommunities(curatedCommunities *communities.CuratedCommunities) (*communities.KnownCommunitiesResponse, error) {
	if err := m.subscribeToCuratedCommunityDescriptions(curatedCommunities.ContractCommunities); err != nil {
		return nil, err
	}

	response, err := m.communitiesManager.GetStoredDescriptionForCommunities(curatedCommunities.ContractCommunities)
	if err != nil {
		return nil, err
	}
	response.ContractFeaturedCommunities = curatedCommunities.ContractFeaturedCommunities

	m.shutdownWaitGroup.Add(1)

	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		m.logger.Debug("fetching unknown curated communities")

		for _, communityID := range response.UnknownCommunities {
			options := []StoreNodeRequestOption{
				WithRequireNewerCommunityDescription(false),
			}
			_, _, err := m.storeNodeRequestsManager.FetchCommunity(m.ctx, communityID, options)
			if err != nil {
				m.logger.Error("failed to fetch curated community",
					zap.String("communityID", communityID),
					zap.Error(err),
				)
			}
		}
	}()

	return response, nil
}

func (m *Messenger) subscribeToCuratedCommunityDescriptions(communityIDs []string) error {
	for _, communityID := range communityIDs {
		hadFilter := m.messaging.ChatFilterByChatID(communityID) != nil
		filter, created, err := m.storeNodeRequestsManager.getFilter(storeNodeCommunityRequest, communityID, types2.DefaultShard())
		if err != nil {
			if created && filter != nil {
				m.storeNodeRequestsManager.forgetFilter(filter)
			}
			if !hadFilter {
				if f := m.messaging.ChatFilterByChatID(communityID); f != nil {
					m.storeNodeRequestsManager.forgetFilter(f)
				}
			}
			return fmt.Errorf("failed to subscribe to curated community %s: %w", communityID, err)
		}
	}
	return nil
}

func (m *Messenger) CuratedCommunities() (*communities.KnownCommunitiesResponse, error) {
	curatedCommunities, err := m.communitiesManager.GetCuratedCommunities()
	if err != nil {
		return nil, err
	}
	return m.fetchCuratedCommunities(curatedCommunities)
}
