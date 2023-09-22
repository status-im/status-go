package infura

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const baseURL = "https://nft.api.infura.io"

type Client struct {
	thirdparty.CollectibleContractOwnershipProvider
	client           *http.Client
	apiKey           string
	apiKeySecret     string
	connectionStatus *connection.Status
}

func NewClient(apiKey string, apiKeySecret string) *Client {
	return &Client{
		client:           &http.Client{Timeout: time.Minute},
		apiKey:           apiKey,
		connectionStatus: connection.NewStatus(),
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
	return InfuraID
}

func (o *Client) IsChainSupported(chainID walletCommon.ChainID) bool {
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet, walletCommon.ArbitrumMainnet, walletCommon.EthereumGoerli, walletCommon.EthereumSepolia:
		return true
	}
	return false
}

func (o *Client) IsConnected() bool {
	return o.connectionStatus.IsConnected()
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
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

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

func (o *Client) FetchAllAssetsByOwner(chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	return o.fetchOwnedAssets(chainID, owner, queryParams, limit)
}

func (o *Client) FetchAllAssetsByOwnerAndContractAddress(chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{}

	if len(cursor) > 0 {
		queryParams["cursor"] = []string{cursor}
	}

	for _, contractAddress := range contractAddresses {
		queryParams.Add("tokenAddress", contractAddress.String())
	}

	return o.fetchOwnedAssets(chainID, owner, queryParams, limit)
}

func (o *Client) fetchOwnedAssets(chainID walletCommon.ChainID, owner common.Address, queryParams url.Values, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	assets := new(thirdparty.FullCollectibleDataContainer)

	if len(queryParams["cursor"]) > 0 {
		assets.PreviousCursor = queryParams["cursor"][0]
	}

	for {
		url := fmt.Sprintf("%s/networks/%d/accounts/%s/assets/nfts?%s", baseURL, chainID, owner.String(), queryParams.Encode())

		resp, err := o.doQuery(url)
		if err != nil {
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

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

		assets.Items = append(assets.Items, container.toCommon()...)
		assets.NextCursor = container.Cursor

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

func (o *Client) FetchAssetsByCollectibleUniqueID(uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	ret := make([]thirdparty.FullCollectibleData, 0, len(uniqueIDs))

	for _, id := range uniqueIDs {
		url := fmt.Sprintf("%s/networks/%d/nfts/%s/tokens/%s", baseURL, id.ContractID.ChainID, id.ContractID.Address.String(), id.TokenID.String())

		resp, err := o.doQuery(url)
		if err != nil {
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		asset := Asset{}
		err = json.Unmarshal(body, &asset)
		if err != nil {
			return nil, err
		}

		item := asset.toCommon(id)

		ret = append(ret, item)
	}

	return ret, nil
}

func (o *Client) FetchCollectionsDataByContractID(contractIDs []thirdparty.ContractID) ([]thirdparty.CollectionData, error) {
	ret := make([]thirdparty.CollectionData, 0, len(contractIDs))

	for _, id := range contractIDs {
		url := fmt.Sprintf("%s/networks/%d/nfts/%s", baseURL, id.ChainID, id.Address.String())

		resp, err := o.doQuery(url)
		if err != nil {
			o.connectionStatus.SetIsConnected(false)
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// if Json is not returned there must be an error
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid json: %s", string(body))
		}

		contract := ContractMetadata{}
		err = json.Unmarshal(body, &contract)
		if err != nil {
			return nil, err
		}

		item := contract.toCommon(id)

		ret = append(ret, item)
	}

	return ret, nil
}
