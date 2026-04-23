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
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const nftMetadataBatchLimit = 100
const contractMetadataBatchLimit = 100
const fetchNoLimitMaxPages = 10
const getNFTsForOwnerPageSize = 500

type Params struct {
	IsProxy        bool
	ProxyCustomURL string
	ProxyStageName string
	APIKey         security.SensitiveString
	Creds          *thirdparty.BasicCreds
	// HttpClient, if set, is used for HTTP to Alchemy (e.g. with Transport wrapping puzzleauth.NewTransport). Otherwise a default 1m client is used.
	HttpClient *http.Client
}

func (o *Client) ID() string {
	return o.id
}

func (o *Client) IsChainSupported(chainID walletCommon.ChainID) bool {
	return o.urlResolver.IsChainSupported(chainID)
}

func (o *Client) IsConnected() bool {
	return o.connectionStatus.IsConnected()
}

type Client struct {
	thirdparty.CollectibleContractOwnershipProvider
	id               string
	urlResolver      URLResolver
	authTransport    *thirdparty.AuthTransport
	connectionStatus *connection.Status
}

func NewClient(apiKey security.SensitiveString) *Client {
	if apiKey.Empty() {
		logutils.ZapLogger().Warn("Alchemy API key not available")
	}

	authParams := thirdparty.AuthParams{
		Type:   thirdparty.AuthTypeAPIKey,
		APIKey: apiKey,
	}

	return &Client{
		id:               AlchemyID,
		urlResolver:      &directURLResolver{apiKey: apiKey},
		authTransport:    thirdparty.NewAuthTransport(&http.Client{Timeout: time.Minute}, authParams, AlchemyID),
		connectionStatus: connection.NewStatus(),
	}
}

func NewClientWithParams(params Params) *Client {
	if params.APIKey.Empty() && params.Creds == nil && params.HttpClient == nil {
		logutils.ZapLogger().Warn("Alchemy API key, credentials, and custom HTTP client not available")
	}

	httpClient := params.HttpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: time.Minute}
	}

	authParams := thirdparty.AuthParams{
		APIKey: params.APIKey,
		Creds:  params.Creds,
	}
	switch {
	case params.HttpClient != nil:
		// e.g. puzzle [http.Transport] on the client; no Basic in applyAuth
		authParams.Type = thirdparty.AuthTypeNone
	case params.Creds != nil:
		authParams.Type = thirdparty.AuthTypeBasic
	case params.IsProxy:
		authParams.Type = thirdparty.AuthTypeNone
	default:
		authParams.Type = thirdparty.AuthTypeAPIKey
	}

	clientID := AlchemyID
	if params.IsProxy {
		clientID = AlchemyProxyID
	}

	var resolver URLResolver
	if params.IsProxy {
		resolver = &proxyURLResolver{customURL: params.ProxyCustomURL, stageName: params.ProxyStageName}
	} else {
		resolver = &directURLResolver{apiKey: params.APIKey}
	}

	return &Client{
		id:               clientID,
		urlResolver:      resolver,
		authTransport:    thirdparty.NewAuthTransport(httpClient, authParams, clientID),
		connectionStatus: connection.NewStatus(),
	}
}

func (o *Client) doQuery(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return o.authTransport.Do(req)
}

func (o *Client) doPostWithJSON(ctx context.Context, url string, payload any) (*http.Response, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadJSON)))
	if err != nil {
		return nil, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	return o.authTransport.Do(req)
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

	baseURL, err := o.urlResolver.GetNFTBaseURL(chainID)
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
	queryParams["pageSize"] = []string{fmt.Sprintf("%d", getNFTsForOwnerPageSize)}
	queryParams["excludeFilters"] = []string{"SPAM"}

	if len(cursor) > 0 {
		queryParams["pageKey"] = []string{cursor}
		assets.PreviousCursor = cursor
	}
	assets.Provider = o.ID()

	baseURL, err := o.urlResolver.GetNFTBaseURL(chainID)
	if err != nil {
		return nil, err
	}

	pageCount := 0

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
		pageCount++
		assets.NextCursor = container.PageKey

		if len(assets.NextCursor) == 0 ||
			(limit != thirdparty.FetchNoLimit && len(assets.Items) >= limit) ||
			(limit == thirdparty.FetchNoLimit && pageCount >= fetchNoLimitMaxPages) {
			break
		}

		queryParams["pageKey"] = []string{assets.NextCursor}
	}

	if limit != thirdparty.FetchNoLimit && len(assets.Items) > limit {
		assets.Items = assets.Items[:limit]
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
	baseURL, err := o.urlResolver.GetNFTBaseURL(chainID)
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
	baseURL, err := o.urlResolver.GetNFTBaseURL(chainID)
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
