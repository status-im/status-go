package efp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

func setupTest(t *testing.T, response []byte) (*httptest.Server, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write(response)
		if err != nil {
			return
		}
	}))

	return srv, func() {
		srv.Close()
	}
}

func TestFetchFollowingAddressesPagination(t *testing.T) {
	expected := EFPFollowingResponse{
		Following: []EFPFollowingRecord{
			{
				Version:    1,
				RecordType: "address",
				Data:       "0x983110309620D911731Ac0932219af06091b6744",
				Tags:       []string{"ens", "efp"},
				ENS: &ENSData{
					Name:   "vitalik.eth",
					Avatar: "https://example.com/avatar.png",
					Records: map[string]string{
						"com.twitter": "vitalikbuterin",
					},
				},
			},
			{
				Version:    1,
				RecordType: "address",
				Data:       "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
				Tags:       []string{"friend"},
			},
		},
	}

	response, err := json.Marshal(expected)
	require.NoError(t, err)
	srv, stop := setupTest(t, response)
	defer stop()

	efpClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	addresses, err := efpClient.FetchFollowingAddresses(t.Context(), userAddress, "", 10, 0)

	require.NoError(t, err)
	require.Len(t, addresses, 2)
	require.Equal(t, "vitalik.eth", addresses[0].ENSName)
	require.Equal(t, "https://example.com/avatar.png", addresses[0].Avatar)
	require.Equal(t, "vitalikbuterin", addresses[0].Records["com.twitter"])
	require.Equal(t, common.HexToAddress("0x983110309620D911731Ac0932219af06091b6744"), addresses[0].Address)
}

func TestFetchFollowingAddressesSearch(t *testing.T) {
	expected := EFPFollowingResponse{
		Following: []EFPFollowingRecord{
			{
				Version:    1,
				RecordType: "address",
				Data:       "0x983110309620D911731Ac0932219af06091b6744",
				Tags:       []string{"ens"},
				ENS: &ENSData{
					Name: "vitalik.eth",
				},
			},
		},
	}

	response, err := json.Marshal(expected)
	require.NoError(t, err)
	srv, stop := setupTest(t, response)
	defer stop()

	efpClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	addresses, err := efpClient.FetchFollowingAddresses(t.Context(), userAddress, "vitalik", 0, 0)

	require.NoError(t, err)
	require.Len(t, addresses, 1)
	require.Equal(t, "vitalik.eth", addresses[0].ENSName)
}

func TestFetchFollowingStats(t *testing.T) {
	expected := EFPStatsResponse{
		FollowingCount: 150,
		FollowersCount: 42,
	}

	response, err := json.Marshal(expected)
	require.NoError(t, err)
	srv, stop := setupTest(t, response)
	defer stop()

	efpClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	count, err := efpClient.FetchFollowingStats(t.Context(), userAddress)

	require.NoError(t, err)
	require.Equal(t, 150, count)
}

func TestFetchFollowingAddressesError(t *testing.T) {
	// Test with malformed JSON response
	resp := []byte("{invalid json")
	srv, stop := setupTest(t, resp)
	defer stop()

	efpClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	userAddress := common.HexToAddress("0x742d35cc6cf4c7c7")
	_, err := efpClient.FetchFollowingAddresses(t.Context(), userAddress, "", 10, 0)

	require.Error(t, err)
}

func TestClientID(t *testing.T) {
	httpClient := thirdparty.NewHTTPClient(
		thirdparty.WithDetailedTimeouts(
			5*time.Second,
			5*time.Second,
			5*time.Second,
			20*time.Second,
		),
		thirdparty.WithMaxRetries(5),
	)
	efpClient := NewClient(httpClient)
	require.Equal(t, "efp", efpClient.ID())
}

func TestClientIsConnected(t *testing.T) {
	httpClient := thirdparty.NewHTTPClient(
		thirdparty.WithDetailedTimeouts(
			5*time.Second,
			5*time.Second,
			5*time.Second,
			20*time.Second,
		),
		thirdparty.WithMaxRetries(5),
	)
	efpClient := NewClient(httpClient)
	require.True(t, efpClient.IsConnected())
}
