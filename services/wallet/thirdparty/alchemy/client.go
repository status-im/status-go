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
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func getBaseURL(chainID uint64) (string, error) {
	switch chainID {
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

	return "", fmt.Errorf("chainID not supported: %d", chainID)
}

func getAPIKeySubpath(apiKey string) string {
	if apiKey == "" {
		return "demo"
	}
	return apiKey
}

func getNFTBaseURL(chainID uint64, apiKey string) (string, error) {
	baseURL, err := getBaseURL(chainID)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/nft/v2/%s", baseURL, getAPIKeySubpath(apiKey)), nil
}

type TokenBalance struct {
	TokenID *bigint.HexBigInt `json:"tokenId"`
	Balance *bigint.BigInt    `json:"balance"`
}

type NFTOwner struct {
	OwnerAddress  common.Address `json:"ownerAddress"`
	TokenBalances []TokenBalance `json:"tokenBalances"`
}

type NFTContractOwnership struct {
	Owners  []NFTOwner `json:"ownerAddresses"`
	PageKey string     `json:"pageKey"`
}

type Client struct {
	thirdparty.NFTContractOwnershipProvider
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

func (o *Client) IsChainSupported(chainID uint64) bool {
	_, err := getBaseURL(chainID)
	return err == nil
}

func alchemyOwnershipToCommon(contractAddress common.Address, alchemyOwnership NFTContractOwnership) (*thirdparty.NFTContractOwnership, error) {
	owners := make([]thirdparty.NFTOwner, 0, len(alchemyOwnership.Owners))
	for _, alchemyOwner := range alchemyOwnership.Owners {
		balances := make([]thirdparty.TokenBalance, 0, len(alchemyOwner.TokenBalances))

		for _, alchemyBalance := range alchemyOwner.TokenBalances {
			balances = append(balances, thirdparty.TokenBalance{
				TokenID: &bigint.BigInt{Int: alchemyBalance.TokenID.Int},
				Balance: alchemyBalance.Balance,
			})
		}
		owner := thirdparty.NFTOwner{
			OwnerAddress:  alchemyOwner.OwnerAddress,
			TokenBalances: balances,
		}

		owners = append(owners, owner)
	}

	ownership := thirdparty.NFTContractOwnership{
		ContractAddress: contractAddress,
		Owners:          owners,
	}

	return &ownership, nil
}

func (o *Client) FetchNFTOwnersByContractAddress(chainID uint64, contractAddress common.Address) (*thirdparty.NFTContractOwnership, error) {
	queryParams := url.Values{
		"contractAddress":   {contractAddress.String()},
		"withTokenBalances": {"true"},
	}

	url, err := getNFTBaseURL(chainID, o.apiKeys[chainID])

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

	var alchemyOwnership NFTContractOwnership
	err = json.Unmarshal(body, &alchemyOwnership)
	if err != nil {
		return nil, err
	}

	return alchemyOwnershipToCommon(contractAddress, alchemyOwnership)
}
