package protocol

func WithCuratedCommunitiesUpdateLoop(enabled bool) Option {
	return func(c *config) error {
		c.codeControlFlags.CuratedCommunitiesUpdateLoopEnabled = enabled
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
