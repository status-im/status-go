package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrNoCommunityId = errors.New("community metrics request has no community id")
var ErrInvalidTimeInterval = errors.New("community metrics request invalid time interval")
var ErrInvalidMaxCount = errors.New("community metrics request max count should be gratear than zero")

type CommunityMetricsRequestType uint

const (
	CommunityMetricsRequestMessages CommunityMetricsRequestType = iota + 1
	CommunityMetricsRequestMembers
	CommunityMetricsRequestControlNodeUptime
)

type CommunityMetricsRequest struct {
	CommunityID    types.HexBytes              `json:"communityId"`
	Type           CommunityMetricsRequestType `json:"type"`
	StartTimestamp uint64                      `json:"startTimestamp"`
	EndTimestamp   uint64                      `json:"endTimestamp"`
	MaxCount       uint                        `json:"maxCount"`
}

func (r *CommunityMetricsRequest) Validate() error {
	if len(r.CommunityID) == 0 {
		return ErrNoCommunityId
	}

	if r.StartTimestamp == 0 || r.EndTimestamp == 0 || r.StartTimestamp >= r.EndTimestamp {
		return ErrInvalidTimeInterval
	}

	if r.MaxCount < 1 {
		return ErrInvalidMaxCount
	}
	return nil
}
