package protocol

func WithStubOnlineChecker() Option {
	return func(c *config) error {
		c.onlineChecker = func() bool {
			return true
		}
		return nil
	}
}
