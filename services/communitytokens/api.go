package communitytokens

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
)

func NewAPI(s *Service) *API {
	return &API{
		s: s,
	}
}

type API struct {
	s *Service
}

func (api *API) StoreDeployedCollectibles(ctx context.Context, addressFrom types.Address, addressTo types.Address, chainID uint64,
	txHash common.Hash, deploymentParameters requests.DeploymentParameters) (responses.DeploymentDetails, error) {

	savedCommunityToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), deploymentParameters, addressFrom.Hex(), addressTo.Hex(),
		protobuf.CommunityTokenType_ERC721, token.CommunityLevel, txHash.Hex())
	if err != nil {
		return responses.DeploymentDetails{}, err
	}

	return responses.DeploymentDetails{
		ContractAddress: addressTo.Hex(),
		TransactionHash: txHash.Hex(),
		CommunityToken:  savedCommunityToken}, nil
}

func (api *API) StoreDeployedOwnerToken(ctx context.Context, addressFrom types.Address, chainID uint64, txHash common.Hash,
	ownerTokenParameters requests.DeploymentParameters, masterTokenParameters requests.DeploymentParameters) (responses.DeploymentDetails, error) {

	savedOwnerToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), ownerTokenParameters, addressFrom.Hex(),
		api.s.TemporaryOwnerContractAddress(txHash.Hex()), protobuf.CommunityTokenType_ERC721, token.OwnerLevel, txHash.Hex())
	if err != nil {
		return responses.DeploymentDetails{}, err
	}
	savedMasterToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), masterTokenParameters, addressFrom.Hex(),
		api.s.TemporaryMasterContractAddress(txHash.Hex()), protobuf.CommunityTokenType_ERC721, token.MasterLevel, txHash.Hex())
	if err != nil {
		return responses.DeploymentDetails{}, err
	}

	return responses.DeploymentDetails{
		ContractAddress: "",
		TransactionHash: txHash.Hex(),
		OwnerToken:      savedOwnerToken,
		MasterToken:     savedMasterToken}, nil
}

func (api *API) StoreDeployedAssets(ctx context.Context, addressFrom types.Address, addressTo types.Address, chainID uint64,
	txHash common.Hash, deploymentParameters requests.DeploymentParameters) (responses.DeploymentDetails, error) {

	savedCommunityToken, err := api.s.CreateCommunityTokenAndSave(int(chainID), deploymentParameters, addressFrom.Hex(), addressTo.Hex(),
		protobuf.CommunityTokenType_ERC20, token.CommunityLevel, txHash.Hex())
	if err != nil {
		return responses.DeploymentDetails{}, err
	}

	return responses.DeploymentDetails{
		ContractAddress: addressTo.Hex(),
		TransactionHash: txHash.Hex(),
		CommunityToken:  savedCommunityToken}, nil
}

// This is only ERC721 function
func (api *API) RemoteDestructedAmount(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.s.manager.NewCollectiblesInstance(chainID, common.HexToAddress(contractAddress))
	if err != nil {
		return nil, err
	}

	// total supply = airdropped only (w/o burnt)
	totalSupply, err := contractInst.TotalSupply(callOpts)
	if err != nil {
		return nil, err
	}

	// minted = all created tokens (airdropped and remotely destructed)
	mintedCount, err := contractInst.MintedCount(callOpts)
	if err != nil {
		return nil, err
	}

	var res = new(big.Int)
	res.Sub(mintedCount, totalSupply)

	return &bigint.BigInt{Int: res}, nil
}

func (api *API) RemainingSupply(ctx context.Context, chainID uint64, contractAddress string) (*bigint.BigInt, error) {
	return api.s.remainingSupply(ctx, chainID, contractAddress)
}

// Gets signer public key from smart contract with a given chainId and address
func (api *API) GetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string) (string, error) {
	return api.s.GetSignerPubKey(ctx, chainID, contractAddress)
}

// Gets signer public key directly from deployer contract
func (api *API) SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error) {
	return api.s.SafeGetSignerPubKey(ctx, chainID, communityID)
}

// Gets owner token contract address from deployer contract
func (api *API) SafeGetOwnerTokenAddress(ctx context.Context, chainID uint64, communityID string) (string, error) {
	return api.s.SafeGetOwnerTokenAddress(ctx, chainID, communityID)
}

func (api *API) OwnerTokenOwnerAddress(ctx context.Context, chainID uint64, contractAddress string) (string, error) {
	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contractInst, err := api.s.manager.NewOwnerTokenInstance(chainID, common.HexToAddress(contractAddress))
	if err != nil {
		return "", err
	}
	ownerAddress, err := contractInst.OwnerOf(callOpts, big.NewInt(0))
	if err != nil {
		return "", err
	}
	return ownerAddress.Hex(), nil
}
