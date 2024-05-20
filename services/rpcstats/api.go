package rpcstats

import (
	"context"
)

// PublicAPI represents a set of APIs from the namespace.
type PublicAPI struct {
	s *Service
}

// NewAPI creates an instance of the API.
func NewAPI(s *Service) *PublicAPI {
	return &PublicAPI{s: s}
}

// Reset resets RPC usage stats
func (api *PublicAPI) Reset(context context.Context) {
	resetStats()
}

type RPCStats struct {
	Total            uint            `json:"total"`
	CounterPerMethod map[string]uint `json:"methods"`
}

// GetStats returns RPC usage stats
func (api *PublicAPI) GetStats(context context.Context) (RPCStats, error) {
	total, perMethod := getStats()

	counterPerMethod := make(map[string]uint)
	perMethod.Range(func(key, value interface{}) bool {
		counterPerMethod[key.(string)] = value.(uint)
		return true
	})

	return RPCStats{
		Total:            total,
		CounterPerMethod: counterPerMethod,
	}, nil
}
