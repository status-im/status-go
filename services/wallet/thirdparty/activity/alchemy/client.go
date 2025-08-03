package alchemy

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	geth_rpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	wc "github.com/status-im/status-go/services/wallet/common"
)

const AlchemyID = "alchemy"

type Client struct {
	ethClientGetter  rpc.EthClientGetter
	connectionStatus *connection.Status
	persistence      *Persistence
}

func (c *Client) ID() string {
	return AlchemyID
}

func (c *Client) IsConnected() bool {
	return c.connectionStatus.IsConnected()
}

func (c *Client) IsChainSupported(chainID wc.ChainID) bool {
	client, err := c.ethClientGetter.GetEthClient(uint64(chainID))
	return err == nil && client != nil
}

func NewClient(ethClientGetter rpc.EthClientGetter, persistence *Persistence) *Client {
	return &Client{
		ethClientGetter:  ethClientGetter,
		connectionStatus: connection.NewStatus(),
		persistence:      persistence,
	}
}

func (c *Client) FetchActivity(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, cursor string, limit int) (thirdparty.ActivityEntryContainer, error) {
	response := thirdparty.ActivityEntryContainer{
		Provider:       c.ID(),
		PreviousCursor: cursor,
		NextCursor:     cursor,
	}

	maxCount := MaxAssetTransfersCount
	if limit > thirdparty.FetchNoLimit && limit < MaxAssetTransfersCount {
		maxCount = limit
	}

	// We need to make one request for outgoing transfers and a separate one for incoming ones
	order := TransferOrderNewToOld
	if parameters.Order == thirdparty.OldToNew {
		order = TransferOrderOldToNew
	}

	params := GetAssetTransfersParams{
		ToAddress: &parameters.Address,
		Category: []TransferCategory{
			TransferCategoryExternal,
			TransferCategoryErc20,
			TransferCategoryErc721,
			TransferCategoryErc1155,
			TransferCategorySpecialNft,
		},
		Order:            order,
		WithMetadata:     true,
		ExcludeZeroValue: false,
		MaxCount:         (*hexutil.Big)(big.NewInt((int64)(maxCount))),
	}

	// Defaults to 0x0 (doesn't support "earliest")
	if parameters.FromBlock != nil && *parameters.FromBlock != geth_rpc.EarliestBlockNumber {
		params.FromBlock = parameters.FromBlock
	}
	// Defaults to "latest"
	params.ToBlock = parameters.ToBlock

	responseTransfers := make([]Transfer, 0, 2*maxCount)
	for {
		outgoingCursor, outgoingDone, incomingCursor, incomingDone, err := decodeCursor(response.NextCursor)
		if err != nil {
			return response, err
		}

		if !parameters.Direction.IncludesOutgoing() {
			outgoingDone = true
		}

		if !parameters.Direction.IncludesIncoming() {
			incomingDone = true
		}

		if !outgoingDone {
			params.FromAddress = &parameters.Address
			params.ToAddress = nil
			params.PageKey = outgoingCursor
			tmpResponse, err := c.fetchActivity(ctx, chainID, params)
			if err != nil {
				return response, err
			}
			responseTransfers = append(responseTransfers, tmpResponse.Transfers...)
			if tmpResponse.PageKey == "" {
				outgoingCursor = ""
				outgoingDone = true
			} else {
				outgoingCursor = tmpResponse.PageKey
				outgoingDone = false
			}
		}

		if !incomingDone {
			params.FromAddress = nil
			params.ToAddress = &parameters.Address
			params.PageKey = incomingCursor
			tmpResponse, err := c.fetchActivity(ctx, chainID, params)
			if err != nil {
				return response, err
			}
			responseTransfers = append(responseTransfers, tmpResponse.Transfers...)
			if tmpResponse.PageKey == "" {
				incomingCursor = ""
				incomingDone = true
			} else {
				incomingCursor = tmpResponse.PageKey
				incomingDone = false
			}
		}

		response.NextCursor = encodeCursor(outgoingCursor, outgoingDone, incomingCursor, incomingDone)
		if response.NextCursor == thirdparty.FetchFromStartCursor {
			break
		}
		if limit > thirdparty.FetchNoLimit && len(response.Items) >= limit {
			break
		}
	}

	c.persistence.SaveTransfers(responseTransfers, chainID, parameters.Address)
	response.Items = TransfersToThirdpartyActivityEntries(responseTransfers, chainID, parameters.Address)

	return response, nil
}

func (c *Client) fetchActivity(ctx context.Context, chainID uint64, parameters GetAssetTransfersParams) (*GetAssetTranfersResponse, error) {
	client, err := c.ethClientGetter.GetEthClient(chainID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, thirdparty.ErrChainIDNotSupported
	}

	var response GetAssetTranfersResponse
	err = client.CallContext(ctx, &response, getAssetTransfersMethod, parameters)
	if err != nil {
		if ctx.Err() == nil {
			c.connectionStatus.SetIsConnected(false)
		}
	}
	c.connectionStatus.SetIsConnected(true)

	return &response, err
}

const cursorFormat = "%s///%t///%s///%t"

func decodeCursor(cursor string) (outgoingCursor string, outgoingDone bool, incomingCursor string, incomingDone bool, err error) {
	if cursor == thirdparty.FetchFromStartCursor {
		return "", false, "", false, nil
	}

	_, err = fmt.Scanf(cursorFormat, &outgoingCursor, &outgoingDone, &incomingCursor, &incomingDone)
	return
}

func encodeCursor(outgoingCursor string, outgoingDone bool, incomingCursor string, incomingDone bool) (cursor string) {
	if outgoingDone && incomingDone {
		return thirdparty.FetchFromStartCursor
	}
	return fmt.Sprintf(cursorFormat, outgoingCursor, outgoingDone, incomingCursor, incomingDone)
}

// Local Variables:
// default-directory: "/Users/vkjr/work/projects/status/status-desktop/vendor/status-go/"
// compile-command: "go test -v -run ^TestFetchHistoryBoth$ ./tests-unit-network/alchemy-activity"
// End:

// (setq default-directory "/Users/vkjr/work/projects/status/status-desktop/vendor/status-go/")
