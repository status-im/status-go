package common

import (
	"encoding/json"
	"fmt"
)

type RateLimitsConfig struct {
	Filter       *RateLimit `json:"-"`
	Lightpush    *RateLimit `json:"-"`
	PeerExchange *RateLimit `json:"-"`
}

type RateLimit struct {
	Volume   int               // Number of allowed messages per period
	Period   int               // Length of each rate-limit period (in TimeUnit)
	TimeUnit RateLimitTimeUnit // Time unit of the period
}

type RateLimitTimeUnit string

const Hour RateLimitTimeUnit = "h"
const Minute RateLimitTimeUnit = "m"
const Second RateLimitTimeUnit = "s"
const Millisecond RateLimitTimeUnit = "ms"

func (rl RateLimit) String() string {
	return fmt.Sprintf("%d/%d%s", rl.Volume, rl.Period, rl.TimeUnit)
}

func (rl RateLimit) MarshalJSON() ([]byte, error) {
	return json.Marshal(rl.String())
}

func (rlc RateLimitsConfig) MarshalJSON() ([]byte, error) {
	output := []string{}
	if rlc.Filter != nil {
		output = append(output, fmt.Sprintf("filter:%s", rlc.Filter.String()))
	}
	if rlc.Lightpush != nil {
		output = append(output, fmt.Sprintf("lightpush:%s", rlc.Lightpush.String()))
	}
	if rlc.PeerExchange != nil {
		output = append(output, fmt.Sprintf("px:%s", rlc.PeerExchange.String()))
	}
	return json.Marshal(output)
}
