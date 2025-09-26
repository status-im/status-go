package eventfilter

import (
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/eventlog"
)

type TransferType string

const (
	TransferTypeERC20   TransferType = "erc20"
	TransferTypeERC721  TransferType = "erc721"
	TransferTypeERC1155 TransferType = "erc1155"
)

type Direction string

const (
	Send    Direction = "send"
	Receive Direction = "receive"
	Both    Direction = "both"
)

type TransferQueryConfig struct {
	FromBlock         *big.Int
	ToBlock           *big.Int
	ContractAddresses []common.Address
	Accounts          []common.Address
	TransferTypes     []TransferType
	Direction         Direction
}

func (c *TransferQueryConfig) ToFilterQueries() []ethereum.FilterQuery {
	// FilterQuery should match Transfer events of the given types, with
	// any of the given addresses in the from/to fields

	// Transfer types need to be specified
	if len(c.TransferTypes) == 0 {
		return nil
	}

	// If both are specified, FromBlock must not be greater than ToBlock
	if c.FromBlock != nil && c.ToBlock != nil && c.FromBlock.Cmp(c.ToBlock) > 0 {
		return nil
	}

	// Convert addresses to topic format (32-byte padded)
	var addressTopics []common.Hash
	for _, addr := range c.Accounts {
		addressTopics = append(addressTopics, common.BytesToHash(addr.Bytes()))
	}

	// Create optimized queries based on transfer types and direction
	var topicsList []topics
	switch c.Direction {
	case Send:
		topicsList = buildSendTopicsList(addressTopics, c.TransferTypes)
	case Receive:
		topicsList = buildReceiveTopicsList(addressTopics, c.TransferTypes)
	case Both:
		topicsList = buildBothTopicsList(addressTopics, c.TransferTypes)
	}

	queries := make([]ethereum.FilterQuery, 0, len(topicsList))
	for _, topics := range topicsList {
		queries = append(queries, buildFilterQuery(c.FromBlock, c.ToBlock, c.ContractAddresses, topics))
	}

	return queries
}

func buildFilterQuery(fromBlock *big.Int, toBlock *big.Int, contractAddresses []common.Address, topics topics) ethereum.FilterQuery {
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Topics:    topics,
	}
	if len(contractAddresses) > 0 {
		query.Addresses = contractAddresses
	}
	return query
}

func unpackTransferTypes(transferTypes []TransferType) (hasERC20 bool, hasERC721 bool, hasERC1155 bool) {
	for _, transferType := range transferTypes {
		switch transferType {
		case TransferTypeERC20:
			hasERC20 = true
		case TransferTypeERC721:
			hasERC721 = true
		case TransferTypeERC1155:
			hasERC1155 = true
		}
	}
	return
}

type topics [][]common.Hash

func buildSendTopicsList(addressTopics []common.Hash, transferTypes []TransferType) []topics {
	var topicsList []topics

	// Send direction: separate queries for each transfer type
	// - ERC20/ERC721: [eventSignature, address] (2 topics)
	// - ERC1155: [eventSignature, {}, address] (3 topics, omitting empty last topic)

	hasERC20, hasERC721, hasERC1155 := unpackTransferTypes(transferTypes)

	if hasERC20 || hasERC721 {
		topicsList = append(topicsList, topics{
			{eventlog.ERC20TransferID}, // Match Transfer event signature (same for ERC20 and ERC721)
			addressTopics,              // Match any of our addresses in 'from' field
		})
	}

	if hasERC1155 {
		topicsList = append(topicsList, topics{
			{eventlog.ERC1155TransferSingleID, eventlog.ERC1155TransferBatchID}, // Match either TransferSingle OR TransferBatch
			{},            // Any operator
			addressTopics, // Match any of our addresses in 'from' field
		})
	}

	return topicsList
}

func buildReceiveTopicsList(addressTopics []common.Hash, transferTypes []TransferType) []topics {
	var topicsList []topics

	// Receive direction: separate queries for each transfer type
	// - ERC20/ERC721: [eventSignature, {}, address] (3 topics)
	// - ERC1155: [eventSignature, {}, {}, address] (4 topics)

	hasERC20, hasERC721, hasERC1155 := unpackTransferTypes(transferTypes)

	if hasERC20 || hasERC721 {
		topicsList = append(topicsList, topics{
			{eventlog.ERC20TransferID}, // Match Transfer event signature (same for ERC20 and ERC721)
			{},                         // Any 'from' address
			addressTopics,              // Match any of our addresses in 'to' field
		})
	}

	if hasERC1155 {
		topicsList = append(topicsList, topics{
			{eventlog.ERC1155TransferSingleID, eventlog.ERC1155TransferBatchID}, // Match either TransferSingle OR TransferBatch
			{},            // Any operator
			{},            // Any 'from' address
			addressTopics, // Match any of our addresses in 'to' field
		})
	}

	return topicsList
}

func buildBothTopicsList(addressTopics []common.Hash, transferTypes []TransferType) []topics {
	var topicsList []topics

	// Both direction: optimized with merging where possible
	// - ERC20/ERC721 Send: [eventSignature, address] (2 topics)
	// - Merged ERC20/ERC721 Receive + ERC1155 Send: [eventSignature, {}, address] (3 topics)
	// - ERC1155 Receive: [eventSignature, {}, {}, address] (4 topics)

	hasERC20, hasERC721, hasERC1155 := unpackTransferTypes(transferTypes)

	// ERC20/ERC721 Send query
	if hasERC20 || hasERC721 {
		topicsList = append(topicsList, topics{
			{eventlog.ERC20TransferID}, // Match Transfer event signature (same for ERC20 and ERC721)
			addressTopics,              // Match any of our addresses in 'from' field
		})
	}

	// Merged ERC20/ERC721 Receive + ERC1155 Send query (only if we have both)
	{
		var eventSignatures []common.Hash
		if hasERC20 || hasERC721 {
			eventSignatures = append(eventSignatures, eventlog.ERC20TransferID) // Transfer event signature (same for ERC20 and ERC721)
		}
		if hasERC1155 {
			eventSignatures = append(eventSignatures, eventlog.ERC1155TransferSingleID, eventlog.ERC1155TransferBatchID)
		}

		topicsList = append(topicsList, topics{
			eventSignatures, // Match any of the event signatures
			{},              // Any 'from' address (or operator for ERC1155)
			addressTopics,   // Match any of our addresses in 'to' field
		})
	}

	// ERC1155 Receive query
	if hasERC1155 {
		topicsList = append(topicsList, topics{
			{eventlog.ERC1155TransferSingleID, eventlog.ERC1155TransferBatchID}, // Match either TransferSingle OR TransferBatch
			{},            // Any operator
			{},            // Any 'from' address
			addressTopics, // Match any of our addresses in 'to' field
		})
	}

	return topicsList
}
