package efp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const baseURL = "https://api.ethfollow.xyz/api/v1"

const (
	requestDelay = 100 * time.Millisecond
)

// ENSData represents ENS information from the EFP API
type ENSData struct {
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Avatar    string            `json:"avatar"`
	Records   map[string]string `json:"records"`
	UpdatedAt string            `json:"updated_at"`
}

// EFPFollowingRecord represents a single following record from the EFP API
type EFPFollowingRecord struct {
	Version    int      `json:"version"`
	RecordType string   `json:"record_type"`
	Data       string   `json:"data"` // Ethereum address
	Tags       []string `json:"tags"`
	ENS        *ENSData `json:"ens"` // Nullable ENS data
}

// EFPFollowingResponse represents the response from the EFP following endpoint
type EFPFollowingResponse struct {
	Following []EFPFollowingRecord `json:"following"`
}

// FollowingAddress represents a processed following address for internal use
type FollowingAddress struct {
	Address common.Address    `json:"address"`
	Tags    []string          `json:"tags"`
	ENSName string            `json:"ensName"` // ENS name from API
	Avatar  string            `json:"avatar"`  // Avatar URL from API
	Records map[string]string `json:"records"` // Social links and other ENS records
}

type Client struct {
	httpClient *thirdparty.HTTPClient
	baseURL    string
}

func NewClient() *Client {
	httpClient := thirdparty.NewHTTPClient(
		thirdparty.WithDetailedTimeouts(
			5*time.Second,  // dialTimeout
			5*time.Second,  // tlsHandshakeTimeout
			5*time.Second,  // responseHeaderTimeout
			20*time.Second, // requestTimeout
		),
		thirdparty.WithMaxRetries(5),
	)

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

func (c *Client) ID() string {
	return "efp"
}

func (c *Client) IsConnected() bool {
	// For now, always return true since we don't have connection status tracking
	// This can be enhanced later with proper connection status management
	return true
}

// FetchFollowingAddresses fetches the list of addresses that the given user is following
func (c *Client) FetchFollowingAddresses(ctx context.Context, userAddress common.Address, search string, limit, offset int) ([]FollowingAddress, error) {
	// Apply sensible defaults and limits
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Use different endpoint for search vs regular listing
	var urlStr string
	if search != "" {
		urlStr = fmt.Sprintf("%s/users/%s/searchFollowing?include=ens&limit=%d&offset=%d&sort=followers&term=%s",
			c.baseURL, userAddress.Hex(), limit, offset, url.QueryEscape(search))
	} else {
		urlStr = fmt.Sprintf("%s/users/%s/following?include=ens&limit=%d&offset=%d&sort=followers",
			c.baseURL, userAddress.Hex(), limit, offset)
	}

	response, err := c.httpClient.DoGetRequest(ctx, urlStr, nil)
	if err != nil {
		return nil, err
	}

	return handleFollowingResponse(response)
}

func handleFollowingResponse(response []byte) ([]FollowingAddress, error) {
	var efpResponse EFPFollowingResponse
	err := json.Unmarshal(response, &efpResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal EFP response: %w - %s", err, string(response))
	}

	result := make([]FollowingAddress, 0, len(efpResponse.Following))
	for _, record := range efpResponse.Following {
		// Only process address records
		if record.RecordType != "address" {
			continue
		}

		// Parse the address
		if !common.IsHexAddress(record.Data) {
			continue // Skip invalid addresses
		}

		followingAddr := FollowingAddress{
			Address: common.HexToAddress(record.Data),
			Tags:    record.Tags,
		}

		// Include ENS data if available
		if record.ENS != nil {
			followingAddr.ENSName = record.ENS.Name
			followingAddr.Avatar = record.ENS.Avatar
			followingAddr.Records = record.ENS.Records
		}

		result = append(result, followingAddr)
	}

	return result, nil
}
