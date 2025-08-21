package server

type Config struct {
	profilingEnabled bool
}

func defaultConfig() *Config {
	return &Config{
		profilingEnabled: false,
	}
}

type Option func(*Config)

func WithProfiling(enabled bool) func(*Config) {
	return func(c *Config) {
		c.profilingEnabled = enabled
	}
}
