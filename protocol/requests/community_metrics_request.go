package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
)

var ErrNoCommunityId = errors.New("community metrics request has no community id")
var ErrInvalidTimestampInterval = errors.New("community metrics request invalid time interval")
var ErrInvalidTimestampStep = errors.New("community metrics request invalid time step")

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
	StepTimestamp  uint64                      `json:"stepTimestamp"`
}

func (r *CommunityMetricsRequest) Validate() error {
	if len(r.CommunityID) == 0 {
		return ErrNoCommunityId
	}

	if r.StartTimestamp == 0 || r.EndTimestamp == 0 || r.StartTimestamp >= r.EndTimestamp {
		return ErrInvalidTimestampInterval
	}

	if r.StepTimestamp < 1 || r.StepTimestamp > (r.EndTimestamp-r.StartTimestamp) {
		return ErrInvalidTimestampStep
	}
	return nil
}
