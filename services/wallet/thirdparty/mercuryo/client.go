package mercuryo

import (
	"github.com/status-im/status-go/v10/services/wallet/thirdparty"
)

type Client struct {
	httpClient *thirdparty.HTTPClient
}

func NewClient() *Client {
	return &Client{
		httpClient: thirdparty.NewHTTPClient(),
	}
}
