package wakuv2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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
