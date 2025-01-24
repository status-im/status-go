package waku

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v3"
)

type NwakuInfo struct {
	ListenAddresses []string `json:"listenAddresses"`
	EnrUri          string   `json:"enrUri"`
}

func GetNwakuInfo(host *string, port *int) (NwakuInfo, error) {
	nwakuRestPort := 8645
	if port != nil {
		nwakuRestPort = *port
	}
	envNwakuRestPort := os.Getenv("NWAKU_REST_PORT")
	if envNwakuRestPort != "" {
		v, err := strconv.Atoi(envNwakuRestPort)
		if err != nil {
			return NwakuInfo{}, err
		}
		nwakuRestPort = v
	}

	nwakuRestHost := "localhost"
	if host != nil {
		nwakuRestHost = *host
	}
	envNwakuRestHost := os.Getenv("NWAKU_REST_HOST")
	if envNwakuRestHost != "" {
		nwakuRestHost = envNwakuRestHost
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:%d/debug/v1/info", nwakuRestHost, nwakuRestPort))
	if err != nil {
		return NwakuInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NwakuInfo{}, err
	}

	var data NwakuInfo
	err = json.Unmarshal(body, &data)
	if err != nil {
		return NwakuInfo{}, err
	}

	return data, nil
}

type BackOffOption func(*backoff.ExponentialBackOff)

func RetryWithBackOff(o func() error, options ...BackOffOption) error {
	b := backoff.ExponentialBackOff{
		InitialInterval:     time.Millisecond * 100,
		RandomizationFactor: 0.1,
		Multiplier:          1,
		MaxInterval:         time.Second,
		MaxElapsedTime:      time.Second * 10,
		Clock:               backoff.SystemClock,
	}
	for _, option := range options {
		option(&b)
	}
	b.Reset()
	return backoff.Retry(o, &b)
}
