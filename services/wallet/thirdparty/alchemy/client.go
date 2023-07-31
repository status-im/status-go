package alchemy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const AlchemyID = "alchemy"

func getBaseURL(chainID walletCommon.ChainID) (string, error) {
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet:
		return "https://eth-mainnet.g.alchemy.com", nil
	case walletCommon.EthereumGoerli:
		return "https://eth-goerli.g.alchemy.com", nil
	case walletCommon.EthereumSepolia:
		return "https://eth-sepolia.g.alchemy.com", nil
	case walletCommon.OptimismMainnet:
		return "https://opt-mainnet.g.alchemy.com", nil
	case walletCommon.OptimismGoerli:
		return "https://opt-goerli.g.alchemy.com", nil
	case walletCommon.ArbitrumMainnet:
		return "https://arb-mainnet.g.alchemy.com", nil
	case walletCommon.ArbitrumGoerli:
		return "https://arb-goerli.g.alchemy.com", nil
	}

	return "", thirdparty.ErrChainIDNotSupported
}

func (o *Client) ID() string {
	return AlchemyID
}

func (o *Client) IsChainSupported(chainID walletCommon.ChainID) bool {
	_, err := getBaseURL(chainID)
	return err == nil
}

func getAPIKeySubpath(apiKey string) string {
	if apiKey == "" {
		return "demo"
	}
	return apiKey
}

func getNFTBaseURL(chainID walletCommon.ChainID, apiKey string) (string, error) {
	baseURL, err := getBaseURL(chainID)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/nft/v2/%s", baseURL, getAPIKeySubpath(apiKey)), nil
}

type Client struct {
	thirdparty.CollectibleContractOwnershipProvider
	client          *http.Client
	apiKeys         map[uint64]string
	IsConnected     bool
	IsConnectedLock sync.RWMutex
}

func NewClient(apiKeys map[uint64]string) *Client {
	return &Client{
		client:  &http.Client{Timeout: time.Minute},
		apiKeys: apiKeys,
	}
}

func (o *Client) doQuery(url string) (*http.Response, error) {
	resp, err := o.client.Get(url)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (o *Client) FetchCollectibleOwnersByContractAddress(chainID walletCommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	queryParams := url.Values{
		"contractAddress":   {contractAddress.String()},
		"withTokenBalances": {"true"},
	}

	url, err := getNFTBaseURL(chainID, o.apiKeys[uint64(chainID)])

	if err != nil {
		return nil, err
	}

	url = url + "/getOwnersForCollection?" + queryParams.Encode()

	resp, err := o.doQuery(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var alchemyOwnership CollectibleContractOwnership
	err = json.Unmarshal(body, &alchemyOwnership)
	if err != nil {
		return nil, err
	}

	return alchemyOwnershipToCommon(contractAddress, alchemyOwnership)
}
