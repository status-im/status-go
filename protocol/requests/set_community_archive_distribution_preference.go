package requests

import (
	"errors"

	"github.com/status-im/status-go/crypto/types"
)

type SetCommunityArchiveDistributionPreference struct {
	CommunityID types.HexBytes `json:"communityId"`
	Preference  string         `json:"preference"`
}

func (s *SetCommunityArchiveDistributionPreference) Validate() error {
	if s == nil {
		return errors.New("invalid request")
	}

	if len(s.CommunityID) == 0 {
		return errors.New("community ID is required")
	}

	// Validate preference value
	switch s.Preference {
	case "auto", "torrent", "codex":
		return nil
	default:
		return errors.New("invalid preference, must be one of: auto, torrent, codex")
	}
}
