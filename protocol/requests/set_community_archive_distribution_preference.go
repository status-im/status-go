package requests

import (
	"errors"
)

type SetArchiveDistributionPreference struct {
	Preference string `json:"preference"`
}

func (s *SetArchiveDistributionPreference) Validate() error {
	if s == nil {
		return errors.New("invalid request")
	}

	// Validate preference value
	switch s.Preference {
	case "torrent", "LogosStorage":
		return nil
	default:
		return errors.New("invalid preference, must be one of: torrent, LogosStorage")
	}
}
