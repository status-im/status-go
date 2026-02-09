package alchemy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/security"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/puzzleauth"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const nftMetadataBatchLimit = 100
const contractMetadataBatchLimit = 100

type Params struct {
	IsProxy          bool
	ProxyCustomURL   string
	ProxyStageName   string
	APIKey           security.SensitiveString
	Creds            *thirdparty.BasicCreds
	PuzzleAuthClient *puzzleauth.Client
}

func getBaseURL(chainID walletCommon.ChainID) (string, error) {
	switch uint64(chainID) {
	case walletCommon.EthereumMainnet:
		return "https://eth-mainnet.g.alchemy.com", nil
	case walletCommon.EthereumSepolia:
		return "https://eth-sepolia.g.alchemy.com", nil
	case walletCommon.OptimismMainnet:
		return "https://opt-mainnet.g.alchemy.com", nil
	case walletCommon.OptimismSepolia:
		return "https://opt-sepolia.g.alchemy.com", nil
	case walletCommon.ArbitrumMainnet:
		return "https://arb-mainnet.g.alchemy.com", nil
	case walletCommon.ArbitrumSepolia:
		return "https://arb-sepolia.g.alchemy.com", nil
	case walletCommon.BaseMainnet:
		return "https://base-mainnet.g.alchemy.com", nil
	case walletCommon.BaseSepolia:
		return "https://base-sepolia.g.alchemy.com", nil
	case walletCommon.LineaMainnet:
		return "https://linea-mainnet.g.alchemy.com", nil
	case walletCommon.LineaSepolia:
		return "https://linea-sepolia.g.alchemy.com", nil
	}

	return "", thirdparty.ErrChainIDNotSupported
}

func (o *Client) ID() string {
	return o.id
}

func (o *Client) IsChainSupported(chainID walletCommon.ChainID) bool {
	// Check if using proxy with puzzle auth or basic auth
	if o.isProxy {
		_, err := GetNftProxyBaseURL(o.proxyCustomURL, o.proxyStageName, chainID)
		return err == nil
	}
	// Check if using direct Alchemy API
	_, err := getBaseURL(chainID)
	return err == nil
}

func (o *Client) IsConnected() bool {
	return o.connectionStatus.IsConnected()
}

func getAPIKeySubpath(apiKey security.SensitiveString) security.SensitiveString {
	if apiKey.Empty() {
		return security.NewSensitiveString("demo")
	}
	return apiKey
}

func getNFTBaseURL(chainID walletCommon.ChainID, apiKey security.SensitiveString) (string, error) {
	baseURL, err := getBaseURL(chainID)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/nft/v3/%s", baseURL, getAPIKeySubpath(apiKey).Reveal()), nil
}

func (o *Client) getNFTBaseURL(chainID walletCommon.ChainID) (string, error) {
	// When using proxy (with puzzle auth or basic auth), use proxy URL
	if o.isProxy {
		return GetNftProxyBaseURL(o.proxyCustomURL, o.proxyStageName, chainID)
	}

	// When using direct Alchemy API, construct URL with API key
	return getNFTBaseURL(chainID, o.apiKey)
}

type Client struct {
	thirdparty.CollectibleContractOwnershipProvider
	id               string
	isProxy          bool
	client           *http.Client
	apiKey           security.SensitiveString
	proxyCustomURL   string
	proxyStageName   string
	creds            *thirdparty.BasicCreds
	puzzleAuthClient *puzzleauth.Client
	connectionStatus *connection.Status
}

func NewClient(apiKey security.SensitiveString) *Client {
	if apiKey.Empty() {
		logutils.ZapLogger().Warn("Alchemy API key not available")
	}

	return &Client{
		id:               AlchemyID,
		isProxy:          false,
		client:           &http.Client{Timeout: time.Minute},
		apiKey:           apiKey,
		proxyCustomURL:   "",
		proxyStageName:   "",
		creds:            nil,
		connectionStatus: connection.NewStatus(),
	}
}

func NewClientWithParams(params Params) *Client {
	if params.APIKey.Empty() && params.Creds == nil && params.PuzzleAuthClient == nil {
		logutils.ZapLogger().Warn("Alchemy API key, credentials, and puzzle auth not available")
	}

	clientID := AlchemyID
	if params.IsProxy {
		clientID = AlchemyProxyID
	}

	return &Client{
		id:               clientID,
		isProxy:          params.IsProxy,
		client:           &http.Client{Timeout: time.Minute},
		apiKey:           params.APIKey,
		proxyCustomURL:   params.ProxyCustomURL,
		proxyStageName:   params.ProxyStageName,
		creds:            params.Creds,
		puzzleAuthClient: params.PuzzleAuthClient,
		connectionStatus: connection.NewStatus(),
	}
}

func (o *Client) doQuery(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if o.creds != nil {
		req.SetBasicAuth(o.creds.User.Reveal(), o.creds.Password.Reveal())
	}

	return o.doWithRetries(req)
}

func (o *Client) doPostWithJSON(ctx context.Context, url string, payload any) (*http.Response, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	payloadString := string(payloadJSON)
	payloadReader := strings.NewReader(payloadString)

	req, err := http.NewRequestWithContext(ctx, "POST", url, payloadReader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	if o.creds != nil {
		req.SetBasicAuth(o.creds.User.Reveal(), o.creds.Password.Reveal())
	}

	return o.doWithRetries(req)
}

func (o *Client) doWithRetries(req *http.Request) (*http.Response, error) {
	// If puzzle auth client is available, use it
	if o.puzzleAuthClient != nil {
		return o.puzzleAuthClient.DoRequest(req)
	}

	// Otherwise use the shared backoff retry logic
	return thirdparty.DoWithExponentialBackoff(o.client, req, o.ID())
}

func (o *Client) FetchCollectibleOwnersByContractAddress(ctx context.Context, chainID walletCommon.ChainID, contractAddress common.Address) (*thirdparty.CollectibleContractOwnership, error) {
	ownership := thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          make([]thirdparty.CollectibleOwner, 0),
	}

	queryParams := url.Values{
		"contractAddress":   {contractAddress.String()},
		"withTokenBalances": {"true"},
	}

	baseURL, err := o.getNFTBaseURL(chainID)

	if err != nil {
		return nil, err
	}

	for {
		url := fmt.Sprintf("%s/getOwnersForContract?%s", baseURL, queryParams.Encode())

		resp, err := o.doQuery(ctx, url)
		if err != nil {
			if ctx.Err() == nil {
				o.connectionStatus.SetIsConnected(false)
			}
			return nil, err
		}
		o.connectionStatus.SetIsConnected(true)

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

func (o *Client) FetchAllAssetsByOwner(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{}

	return o.fetchOwnedAssets(ctx, chainID, owner, queryParams, cursor, limit)
}

func (o *Client) FetchAllAssetsByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, contractAddresses []common.Address, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	queryParams := url.Values{}

	for _, contractAddress := range contractAddresses {
		queryParams.Add("contractAddresses", contractAddress.String())
	}

	return o.fetchOwnedAssets(ctx, chainID, owner, queryParams, cursor, limit)
}

func (o *Client) fetchOwnedAssets(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, queryParams url.Values, cursor string, limit int) (*thirdparty.FullCollectibleDataContainer, error) {
	assets := new(thirdparty.FullCollectibleDataContainer)

	queryParams["owner"] = []string{owner.String()}
	queryParams["withMetadata"] = []string{"true"}
	queryParams["orderBy"] = []string{"transferTime"}

	if len(cursor) > 0 {
		queryParams["pageKey"] = []string{cursor}
		assets.PreviousCursor = cursor
	}
	assets.Provider = o.ID()

	baseURL, err := o.getNFTBaseURL(chainID)

	if err != nil {
		return nil, err
	}

	for {
		url := fmt.Sprintf("%s/getNFTsForOwner?%s", baseURL, queryParams.Encode())

		resp, err := o.doQuery(ctx, url)
		if err != nil {
			if ctx.Err() == nil {
				o.connectionStatus.SetIsConnected(false)
			}
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

		container := OwnedNFTList{}
		err = json.Unmarshal(body, &container)
		if err != nil {
			return nil, err
		}

		assets.Items = append(assets.Items, alchemyToCollectiblesData(chainID, container.OwnedNFTs, &owner)...)
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

func getCollectibleUniqueIDBatches(ids []thirdparty.CollectibleUniqueID) []BatchTokenIDs {
	batches := make([]BatchTokenIDs, 0)

	for startIdx := 0; startIdx < len(ids); startIdx += nftMetadataBatchLimit {
		endIdx := startIdx + nftMetadataBatchLimit
		if endIdx > len(ids) {
			endIdx = len(ids)
		}

		pageIDs := ids[startIdx:endIdx]

		batchIDs := BatchTokenIDs{
			IDs: make([]TokenID, 0, len(pageIDs)),
		}
		for _, id := range pageIDs {
			batchID := TokenID{
				ContractAddress: id.ContractID.Address,
				TokenID:         id.TokenID,
			}
			batchIDs.IDs = append(batchIDs.IDs, batchID)
		}

		batches = append(batches, batchIDs)
	}

	return batches
}

func (o *Client) fetchAssetsByBatchTokenIDs(ctx context.Context, chainID walletCommon.ChainID, batchIDs BatchTokenIDs) ([]thirdparty.FullCollectibleData, error) {
	baseURL, err := o.getNFTBaseURL(chainID)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/getNFTMetadataBatch", baseURL)

	resp, err := o.doPostWithJSON(ctx, url, batchIDs)
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

	assets := NFTList{}
	err = json.Unmarshal(body, &assets)
	if err != nil {
		return nil, err
	}

	ret := alchemyToCollectiblesData(chainID, assets.NFTs, nil)

	return ret, nil
}

func (o *Client) FetchAssetsByCollectibleUniqueID(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID) ([]thirdparty.FullCollectibleData, error) {
	ret := make([]thirdparty.FullCollectibleData, 0, len(uniqueIDs))

	idsPerChainID := thirdparty.GroupCollectibleUIDsByChainID(uniqueIDs)

	for chainID, ids := range idsPerChainID {
		batches := getCollectibleUniqueIDBatches(ids)
		for _, batch := range batches {
			assets, err := o.fetchAssetsByBatchTokenIDs(ctx, chainID, batch)
			if err != nil {
				return nil, err
			}

			ret = append(ret, assets...)
		}
	}

	return ret, nil
}

func (o *Client) FetchCollectionSocials(ctx context.Context, contractID thirdparty.ContractID) (*thirdparty.CollectionSocials, error) {
	resp, err := o.FetchCollectionsDataByContractID(ctx, []thirdparty.ContractID{contractID})
	if err != nil {
		return nil, err
	}
	if len(resp) > 0 {
		return resp[0].Socials, nil
	}
	return nil, nil
}

func getContractAddressBatches(ids []thirdparty.ContractID) []BatchContractAddresses {
	batches := make([]BatchContractAddresses, 0)

	for startIdx := 0; startIdx < len(ids); startIdx += contractMetadataBatchLimit {
		endIdx := startIdx + contractMetadataBatchLimit
		if endIdx > len(ids) {
			endIdx = len(ids)
		}

		pageIDs := ids[startIdx:endIdx]

		batchIDs := BatchContractAddresses{
			Addresses: make([]common.Address, 0, len(pageIDs)),
		}
		for _, id := range pageIDs {
			batchIDs.Addresses = append(batchIDs.Addresses, id.Address)
		}

		batches = append(batches, batchIDs)
	}

	return batches
}

func (o *Client) fetchCollectionsDataByBatchContractAddresses(ctx context.Context, chainID walletCommon.ChainID, batchAddresses BatchContractAddresses) ([]thirdparty.CollectionData, error) {
	baseURL, err := o.getNFTBaseURL(chainID)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/getContractMetadataBatch", baseURL)

	resp, err := o.doPostWithJSON(ctx, url, batchAddresses)
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

	collections := ContractList{}
	err = json.Unmarshal(body, &collections)
	if err != nil {
		return nil, err
	}

	ret := alchemyToCollectionsData(chainID, collections.Contracts)

	return ret, nil
}

func (o *Client) FetchCollectionsDataByContractID(ctx context.Context, contractIDs []thirdparty.ContractID) ([]thirdparty.CollectionData, error) {
	ret := make([]thirdparty.CollectionData, 0, len(contractIDs))

	idsPerChainID := thirdparty.GroupContractIDsByChainID(contractIDs)

	for chainID, ids := range idsPerChainID {
		batches := getContractAddressBatches(ids)
		for _, batch := range batches {
			contractsData, err := o.fetchCollectionsDataByBatchContractAddresses(ctx, chainID, batch)
			if err != nil {
				return nil, err
			}

			ret = append(ret, contractsData...)
		}
	}

	return ret, nil
}
