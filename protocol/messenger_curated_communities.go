package protocol

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities"
)

const (
	fetchError       int = 0
	fetchSuccess     int = 1
	fetchHasUnknowns int = 2
)

// Regularly gets list of curated communities and signals them to client
func (m *Messenger) startCuratedCommunitiesUpdateLoop() {
	logger := m.logger.Named("startCuratedCommunitiesUpdateLoop")

	type curatedCommunities struct {
		ContractCommunities         []string
		ContractFeaturedCommunities []string
		UnknownCommunities          []string
	}

	go func() {

		var fetchResultsHistory = make([]int, 0)
		var mu = sync.RWMutex{}
		var c = curatedCommunities{}

		for {
			response, err := m.CuratedCommunities()

			if err != nil {
				fetchResultsHistory = append(fetchResultsHistory, fetchError)
			} else {
				mu.Lock()
				// Check if it's the same values we had
				if !reflect.DeepEqual(c.ContractCommunities, response.ContractCommunities) ||
					!reflect.DeepEqual(c.ContractFeaturedCommunities, response.ContractFeaturedCommunities) ||
					!reflect.DeepEqual(c.UnknownCommunities, response.UnknownCommunities) {
					// One of the communities is different, send the updated response
					m.config.messengerSignalsHandler.SendCuratedCommunitiesUpdate(response)

					// Update the values
					c.ContractCommunities = response.ContractCommunities
					c.ContractFeaturedCommunities = response.ContractFeaturedCommunities
					c.UnknownCommunities = response.UnknownCommunities
				}
				mu.Unlock()

				if len(response.UnknownCommunities) == 0 {
					fetchResultsHistory = append(fetchResultsHistory, fetchSuccess)

				} else {
					fetchResultsHistory = append(fetchResultsHistory, fetchHasUnknowns)
				}
			}

			//keep only 2 last fetch results
			if len(fetchResultsHistory) > 2 {
				fetchResultsHistory = fetchResultsHistory[1:]
			}

			timeTillNextUpdate := calcTimeTillNextUpdate(fetchResultsHistory)
			logger.Debug("Next curated communities update will happen in", zap.Duration("timeTillNextUpdate", timeTillNextUpdate))

			select {
			case <-time.After(timeTillNextUpdate):
			case <-m.quit:
				return
			}
		}
	}()
}

func calcTimeTillNextUpdate(fetchResultsHistory []int) time.Duration {
	// TODO lower this back again once the real curated community contract is up
	// The current contract contains communities that are no longer accessible on waku
	const shortTimeout = 30 * time.Second
	const averageTimeout = 60 * time.Second
	const longTimeout = 300 * time.Second

	twoConsecutiveErrors := (len(fetchResultsHistory) == 2 &&
		fetchResultsHistory[0] == fetchError &&
		fetchResultsHistory[1] == fetchError)

	twoConsecutiveHasUnknowns := (len(fetchResultsHistory) == 2 &&
		fetchResultsHistory[0] == fetchHasUnknowns &&
		fetchResultsHistory[1] == fetchHasUnknowns)

	var timeTillNextUpdate time.Duration

	if twoConsecutiveErrors || twoConsecutiveHasUnknowns {
		timeTillNextUpdate = longTimeout
	} else {
		switch fetchResultsHistory[len(fetchResultsHistory)-1] {
		case fetchError:
			timeTillNextUpdate = shortTimeout
		case fetchSuccess:
			timeTillNextUpdate = longTimeout
		case fetchHasUnknowns:
			timeTillNextUpdate = averageTimeout
		}
	}
	return timeTillNextUpdate
}

func (m *Messenger) CuratedCommunities() (*communities.KnownCommunitiesResponse, error) {
	if m.contractMaker == nil {
		m.logger.Warn("contract maker not initialized")
		return nil, errors.New("contract maker not initialized")
	}

	testNetworksEnabled, err := m.settings.GetTestNetworksEnabled()
	if err != nil {
		return nil, err
	}

	chainID := uint64(10) // Optimism Mainnet
	if testNetworksEnabled {
		chainID = 420 // Optimism Goerli
	}

	directory, err := m.contractMaker.NewDirectory(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

	curatedCommunities, err := directory.GetCommunities(callOpts)
	if err != nil {
		return nil, err
	}
	var communityIDs []types.HexBytes
	for _, c := range curatedCommunities {
		communityIDs = append(communityIDs, c)
	}

	response, err := m.communitiesManager.GetStoredDescriptionForCommunities(communityIDs)
	if err != nil {
		return nil, err
	}

	featuredCommunities, err := directory.GetFeaturedCommunities(callOpts)
	if err != nil {
		return nil, err
	}

	for _, c := range featuredCommunities {
		response.ContractFeaturedCommunities = append(response.ContractFeaturedCommunities, types.HexBytes(c).String())
	}

	// TODO: use mechanism to obtain shard from community ID (https://github.com/status-im/status-desktop/issues/12585)
	var unknownCommunities []communities.CommunityShard
	for _, u := range response.UnknownCommunities {
		unknownCommunities = append(unknownCommunities, communities.CommunityShard{
			CommunityID: u,
		})
	}

	go m.requestCommunitiesFromMailserver(unknownCommunities)

	return response, nil
}
