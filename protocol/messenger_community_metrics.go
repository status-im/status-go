package protocol

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/requests"
)

type CommunityMetricsResponse struct {
	Type        requests.CommunityMetricsRequestType `json:"type"`
	CommunityID types.HexBytes                       `json:"communityId"`
	Entries     map[uint64]int32                     `json:"entries"`
}

func initRangesMap(start uint64, end uint64, step uint64) map[uint64]int32 {
	result := map[uint64]int32{}
	for timestamp := start; timestamp <= end; timestamp += step {
		result[timestamp] = 0
	}
	return result
}

func floorToRange(value uint64, start uint64, end uint64, step uint64) uint64 {
	for timestamp := start; timestamp <= end; timestamp += step {
		if value <= timestamp {
			return timestamp
		}
	}
	return 0
}

func (m *Messenger) collectCommunityMessagesMetrics(request *requests.CommunityMetricsRequest) (*CommunityMetricsResponse, error) {
	community, err := m.GetCommunityByID(request.CommunityID)
	if err != nil {
		return nil, err
	}

	if community == nil {
		return nil, errors.New("no community found")
	}

	// TODO: timestamp summary should be stored in special table, not calculated here
	timestamps, err := m.persistence.FetchMessageTimestampsForChatsByPeriod(community.ChatIDs(), request.StartTimestamp, request.EndTimestamp)
	if err != nil {
		return nil, err
	}

	timestampStep := (request.EndTimestamp - request.StartTimestamp) / uint64(request.MaxCount)
	entries := initRangesMap(request.StartTimestamp, request.EndTimestamp, timestampStep)

	for _, timestamp := range timestamps {
		entries[floorToRange(timestamp, request.StartTimestamp, request.EndTimestamp, timestampStep)] += 1
	}

	response := &CommunityMetricsResponse{
		Type:        request.Type,
		CommunityID: request.CommunityID,
		Entries:     entries,
	}

	return response, nil
}

func (m *Messenger) CollectCommunityMetrics(request *requests.CommunityMetricsRequest) (*CommunityMetricsResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	switch request.Type {
	case requests.CommunityMetricsRequestMessages:
		return m.collectCommunityMessagesMetrics(request)
	default:
		return nil, errors.New("metrics is not implemented yet")
	}
}
