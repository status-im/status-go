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
}

func (c *Client) ID() string {
	return AlchemyID
}

func (c *Client) IsConnected() bool {
	return c.connectionStatus.IsConnected()
}

func (c *Client) IsChainSupported(chainID wc.ChainID) bool {
	client, err := c.ethClientGetter.EthClient(uint64(chainID))
	return err == nil && client != nil
}

func NewClient(ethClientGetter rpc.EthClientGetter) *Client {
	return &Client{
		ethClientGetter:  ethClientGetter,
		connectionStatus: connection.NewStatus(),
	}
}

// FetchTransfers fetches transfer data from Alchemy API with cursor management for pagination.
// It fetches both incoming and outgoing transfers separately and returns them together
func (c *Client) FetchTransfers(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, cursor string, limit int) ([]Transfer, string, error) {
	nextCursor := cursor

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
		outgoingCursor, outgoingDone, incomingCursor, incomingDone, err := decodeCursor(nextCursor)
		if err != nil {
			return nil, nextCursor, err
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
				return nil, nextCursor, err
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
				return nil, nextCursor, err
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

		nextCursor = encodeCursor(outgoingCursor, outgoingDone, incomingCursor, incomingDone)
		if nextCursor == thirdparty.FetchFromStartCursor {
			break
		}
		if limit > thirdparty.FetchNoLimit && len(responseTransfers) >= limit {
			break
		}
	}

	return responseTransfers, nextCursor, nil
}

func (c *Client) fetchActivity(ctx context.Context, chainID uint64, parameters GetAssetTransfersParams) (*GetAssetTranfersResponse, error) {
	client, err := c.ethClientGetter.EthClient(chainID)
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
