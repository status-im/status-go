package onramp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type DataSourceType int

const (
	DataSourceHTTP DataSourceType = iota + 1
	DataSourceStatic
)

type CryptoOnRamp struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Fees              string            `json:"fees"`
	LogoURL           string            `json:"logoUrl"`
	SiteURL           string            `json:"siteUrl"`
	RecurrentSiteURL  string            `json:"recurrentSiteUrl"`
	Hostname          string            `json:"hostname"`
	Params            map[string]string `json:"params"` // TODO implement params in JSON and parsing status-mobile
	SupportedChainIDs []uint64          `json:"supportedChainIds"`
}

type Options struct {
	DataSource     string
	DataSourceType DataSourceType
}

type Manager struct {
	options    *Options
	ramps      []CryptoOnRamp
	lastCalled time.Time
}

func NewManager(options *Options) *Manager {
	return &Manager{
		options: options,
	}
}

func (c *Manager) Get() ([]CryptoOnRamp, error) {
	var ramps []CryptoOnRamp
	var err error

	switch c.options.DataSourceType {
	case DataSourceHTTP:
		if !c.hasCacheExpired(time.Now()) {
			return c.ramps, nil
		}
		ramps, err = c.getFromHTTPDataSource()
		c.lastCalled = time.Now()
	case DataSourceStatic:
		ramps, err = c.getFromStaticDataSource()
	default:
		return nil, fmt.Errorf("unsupported Manager.DataSourceType '%d'", c.options.DataSourceType)
	}
	if err != nil {
		return nil, err
	}

	c.ramps = ramps

	return c.ramps, nil
}

func (c *Manager) hasCacheExpired(t time.Time) bool {
	// If lastCalled + 1 hour is before the given time, then 1 hour hasn't passed yet
	return c.lastCalled.Add(time.Hour).Before(t)
}

func (c *Manager) getFromHTTPDataSource() ([]CryptoOnRamp, error) {
	if c.options.DataSource == "" {
		return nil, errors.New("data source is not set for Manager")
	}

	sgc := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, c.options.DataSource, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "status-go")

	res, err := sgc.Do(req)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(body))

	var ramps []CryptoOnRamp

	err = json.Unmarshal(body, &ramps)
	if err != nil {
		return nil, err
	}

	return ramps, nil
}

func (c *Manager) getFromStaticDataSource() ([]CryptoOnRamp, error) {
	return getOnRampProviders(), nil
}
