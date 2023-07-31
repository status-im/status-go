package infura

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const baseURL = "https://nft.api.infura.io"

type CollectibleOwner struct {
	ContractAddress common.Address `json:"tokenAddress"`
	TokenID         *bigint.BigInt `json:"tokenId"`
	Amount          *bigint.BigInt `json:"amount"`
	OwnerAddress    common.Address `json:"ownerOf"`
}

type CollectibleContractOwnership struct {
	Owners  []CollectibleOwner `json:"owners"`
	Network string             `json:"network"`
	Cursor  string             `json:"cursor"`
}

type Client struct {
	thirdparty.CollectibleContractOwnershipProvider
	client          *http.Client
	apiKey          string
	apiKeySecret    string
	IsConnected     bool
	IsConnectedLock sync.RWMutex
}

func NewClient(apiKey string, apiKeySecret string) *Client {
	return &Client{
		client: &http.Client{Timeout: time.Minute},
		apiKey: apiKey,
	}
}

func (o *Client) doQuery(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(o.apiKey, o.apiKeySecret)

	resp, err := o.client.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (o *Client) ID() string {
	return "infura"
}

func (o *Client) IsChainSupported(chainID walletCommon.ChainID) bool {
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet, walletCommon.EthereumGoerli, walletCommon.EthereumSepolia, walletCommon.ArbitrumMainnet:
		return true
	}
	return false
}

func infuraOwnershipToCommon(contractAddress common.Address, ownersMap map[common.Address][]CollectibleOwner) (*thirdparty.CollectibleContractOwnership, error) {
	owners := make([]thirdparty.CollectibleOwner, 0, len(ownersMap))

	for ownerAddress, ownerTokens := range ownersMap {
		tokenBalances := make([]thirdparty.TokenBalance, 0, len(ownerTokens))

		for _, token := range ownerTokens {
			tokenBalances = append(tokenBalances, thirdparty.TokenBalance{
				TokenID: token.TokenID,
				Balance: token.Amount,
			})
		}

		owners = append(owners, thirdparty.CollectibleOwner{
			OwnerAddress:  ownerAddress,
			TokenBalances: tokenBalances,
		})
	}

	ownership := thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          owners,
	}

	return &ownership, nil
}

func (o *Client) FetchCollectibleOwnersByContractAddress(chainID walletCommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	cursor := ""
	ownersMap := make(map[common.Address][]CollectibleOwner)

	for {
		url := fmt.Sprintf("%s/networks/%d/nfts/%s/owners", baseURL, chainID, contractAddress.String())

		if cursor != "" {
			url = url + "?cursor=" + cursor
		}

		resp, err := o.doQuery(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var infuraOwnership CollectibleContractOwnership
		err = json.Unmarshal(body, &infuraOwnership)
		if err != nil {
			return nil, err
		}

		for _, infuraOwner := range infuraOwnership.Owners {
			ownersMap[infuraOwner.OwnerAddress] = append(ownersMap[infuraOwner.OwnerAddress], infuraOwner)
		}

		cursor = infuraOwnership.Cursor

		if cursor == "" {
			break
		}
	}

	return infuraOwnershipToCommon(contractAddress, ownersMap)
}
