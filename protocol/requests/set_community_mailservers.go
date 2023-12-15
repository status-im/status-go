package requests

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/mailservers"
)

var (
	ErrSetCommunityMailserversEmpty = errors.New("set-community-mailservers: empty payload")
)

type SetCommunityMailServers struct {
	CommunityID types.HexBytes           `json:"communityId"`
	MailServers []mailservers.Mailserver `json:"mailServers"`
}

func (s *SetCommunityMailServers) Validate() error {
	if s == nil || len(s.MailServers) == 0 {
		return ErrSetCommunityMailserversEmpty
	}
	return nil
}
