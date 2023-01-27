package utils

import (
	"time"

	"github.com/beevik/ntp"
)

var NTPServer = "pool.ntp.org"

func GetNTPTime() (time.Time, error) {
	t, err := ntp.Time(NTPServer)
	if err != nil {
		return t, err
	}

	return t, nil
}

func GetNTPMetadata() (*ntp.Response, error) {
	options := ntp.QueryOptions{Timeout: 60 * time.Second, TTL: 10}
	response, err := ntp.QueryWithOptions(NTPServer, options)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func GetTimeOffset() (time.Duration, error) {
	options := ntp.QueryOptions{Timeout: 60 * time.Second, TTL: 10}
	response, err := ntp.QueryWithOptions(NTPServer, options)
	if err != nil {
		return 0, err
	}

	return response.ClockOffset, nil
}
