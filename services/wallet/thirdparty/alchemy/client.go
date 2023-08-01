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

	return fmt.Sprintf("%s/nft/v3/%s", baseURL, getAPIKeySubpath(apiKey)), nil
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
	ownership := thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          make([]thirdparty.CollectibleOwner, 0),
	}

	queryParams := url.Values{
		"contractAddress":   {contractAddress.String()},
		"withTokenBalances": {"true"},
	}

	baseURL, err := getNFTBaseURL(chainID, o.apiKeys[uint64(chainID)])

	if err != nil {
		return nil, err
	}

	for {
		url := fmt.Sprintf("%s/getOwnersForContract?%s", baseURL, queryParams.Encode())

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

		ownership.Owners = append(ownership.Owners, alchemyCollectibleOwnersToCommon(alchemyOwnership.Owners)...)

		if alchemyOwnership.PageKey == "" {
			break
		}

		queryParams["pageKey"] = []string{alchemyOwnership.PageKey}
	}

	return &ownership, nil
}

func (o *Client) FetchAllAssetsByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{}

	return o.fetchOwnedAssets(chainID, owner, queryParams, cursor, limit)
}

func (o *Client) FetchAllAssetsByOwnerAndContractAddress(chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{}

	for _, contractAddress := range contractAddresses {
		queryParams.Add("contractAddresses", contractAddress.String())
	}

	return o.fetchOwnedAssets(chainID, owner, queryParams, cursor, limit)
}

func (o *Client) fetchOwnedAssets(chainID walletCommon.ChainID, owner common.Address, queryParams url.Values, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	assets := new(thirdparty.FullCollectibleDataContainer)

	queryParams["owner"] = []string{owner.String()}
	queryParams["withMetadata"] = []string{"true"}

	if len(cursor) > 0 {
		queryParams["pageKey"] = []string{cursor}
		assets.PreviousCursor = cursor
	}

	baseURL, err := getNFTBaseURL(chainID, o.apiKeys[uint64(chainID)])

	if err != nil {
		return nil, err
	}

	for {
		url := fmt.Sprintf("%s/getNFTsForOwner?%s", baseURL, queryParams.Encode())

		resp, err := o.doQuery(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		container := NFTList{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			return nil, err
		}

		assets.Items = append(assets.Items, container.toCommon(chainID)...)
		assets.NextCursor = container.PageKey

		if len(assets.NextCursor) == 0 {
			break
		}

		queryParams["cursor"] = []string{assets.NextCursor}

		if limit != thirdparty.FetchNoLimit && len(assets.Items) >= limit {
			break
		}
	}

	return assets, nil
}
