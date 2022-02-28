package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/go-version"
	"github.com/status-im/status-go/services/ens"
	"github.com/status-im/status-go/signal"
	"go.uber.org/zap"
)

func NewAPI(ensService *ens.Service) *API {
	return &API{
		ensService: ensService,
		httpClient: &http.Client{Timeout: time.Minute},
	}
}

type API struct {
	ensService *ens.Service
	httpClient *http.Client
}

func (api *API) Check(ctx context.Context, chainID uint64, ens string, currentVersion string) {
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		uri, err := api.ensService.API().ResourceURL(ctx, chainID, ens)
		if err != nil || uri.Host == "" {
			log.Error("can't get obtain the updates content hash url", "ens", ens)
			return
		}

		url := uri.Scheme + "://" + uri.Host + uri.Path + "VERSION"
		response, err := api.httpClient.Get(url)
		if err != nil {
			log.Error("can't get content", zap.String("any", url))
			return
		}

		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			log.Error(fmt.Sprintf("version verification response status error: %v", response.StatusCode))
			return
		}

		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error("version verification body err", "err", err)
			return
		}

		c := make(map[string]interface{})
		err = json.Unmarshal(data, &c)
		if err != nil {
			log.Error("invalid json", "err", err)
		}

		current, err := version.NewVersion(currentVersion)
		if err != nil {
			log.Error("invalid current version", "err", err)
			return
		}

		latestStr := ""
		switch c["version"].(type) {
		case string:
			latestStr = c["version"].(string)
		default:
			log.Error("invalid latest version", "val", c["version"])
			return
		}

		latest, err := version.NewVersion(latestStr)
		if err != nil {
			log.Error("invalid latest version", "err", err)
			return
		}

		signal.SendUpdateAvailable(latest.GreaterThan(current), latestStr)
	}()
}
