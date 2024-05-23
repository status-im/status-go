package thirdparty

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	netUrl "net/url"
	"time"
)

const requestTimeout = 5 * time.Second
const maxNumOfRequestRetries = 5

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (c *HTTPClient) DoGetRequest(ctx context.Context, url string, params netUrl.Values) ([]byte, error) {
	if len(params) > 0 {
		url = url + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	for i := 0; i < maxNumOfRequestRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil || i == maxNumOfRequestRetries-1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *HTTPClient) DoPostRequest(ctx context.Context, url string, params map[string]interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:96.0) Gecko/20100101 Firefox/96.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
