package protocol

import (
	"github.com/status-im/status-go/protocol/communities"
)

func WithCommunityManagerOptions(options []communities.ManagerOption) Option {
	return func(c *config) error {
		c.communityManagerOptions = options
		return nil
	}
}

func WithStubOnlineChecker() Option {
	return func(c *config) error {
		c.onlineChecker = func() bool {
			return true
		}
		return nil
	}
}
