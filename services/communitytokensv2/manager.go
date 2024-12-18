package communitytokensv2

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	communitytokens "github.com/status-im/status-go/contracts/community-tokens"
	"github.com/status-im/status-go/contracts/community-tokens/assets"
	"github.com/status-im/status-go/contracts/community-tokens/collectibles"
	communitytokendeployer "github.com/status-im/status-go/contracts/community-tokens/deployer"
	"github.com/status-im/status-go/contracts/community-tokens/ownertoken"
	communityownertokenregistry "github.com/status-im/status-go/contracts/community-tokens/registry"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/requests"
)

type Manager struct {
	contractMaker *communitytokens.CommunityTokensContractMaker
}

func NewManager(rpcClient *rpc.Client) *Manager {
	return &Manager{
		contractMaker: &communitytokens.CommunityTokensContractMaker{
			RPCClient: rpcClient,
		},
	}
}

func (m *Manager) NewCollectiblesInstance(chainID uint64, contractAddress common.Address) (*collectibles.Collectibles, error) {
	return m.contractMaker.NewCollectiblesInstance(chainID, contractAddress)
}

func (m *Manager) NewCommunityTokenDeployerInstance(chainID uint64) (*communitytokendeployer.CommunityTokenDeployer, error) {
	deployerAddr, err := communitytokendeployer.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}
	return m.contractMaker.NewCommunityTokenDeployerInstance(chainID, deployerAddr)
}

func (m *Manager) NewAssetsInstance(chainID uint64, contractAddress common.Address) (*assets.Assets, error) {
	return m.contractMaker.NewAssetsInstance(chainID, contractAddress)
}

func (m *Manager) NewCommunityOwnerTokenRegistryInstance(chainID uint64, contractAddress common.Address) (*communityownertokenregistry.CommunityOwnerTokenRegistry, error) {
	return m.contractMaker.NewCommunityOwnerTokenRegistryInstance(chainID, contractAddress)
}

func (m *Manager) NewOwnerTokenInstance(chainID uint64, contractAddress common.Address) (*ownertoken.OwnerToken, error) {
	return m.contractMaker.NewOwnerTokenInstance(chainID, contractAddress)
}

func (m *Manager) GetCollectibleContractData(chainID uint64, contractAddress string) (*communities.CollectibleContractData, error) {
	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

	contract, err := m.NewCollectiblesInstance(chainID, common.HexToAddress(contractAddress))
	if err != nil {
		return nil, err
	}
	totalSupply, err := contract.MaxSupply(callOpts)
	if err != nil {
		return nil, err
	}
	transferable, err := contract.Transferable(callOpts)
	if err != nil {
		return nil, err
	}
	remoteBurnable, err := contract.RemoteBurnable(callOpts)
	if err != nil {
		return nil, err
	}

	return &communities.CollectibleContractData{
		TotalSupply:    &bigint.BigInt{Int: totalSupply},
		Transferable:   transferable,
		RemoteBurnable: remoteBurnable,
		InfiniteSupply: requests.GetInfiniteSupply().Cmp(totalSupply) == 0,
	}, nil
}

func (m *Manager) GetAssetContractData(chainID uint64, contractAddress string) (*communities.AssetContractData, error) {
	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}
	contract, err := m.NewAssetsInstance(chainID, common.HexToAddress(contractAddress))
	if err != nil {
		return nil, err
	}
	totalSupply, err := contract.MaxSupply(callOpts)
	if err != nil {
		return nil, err
	}

	return &communities.AssetContractData{
		TotalSupply:    &bigint.BigInt{Int: totalSupply},
		InfiniteSupply: requests.GetInfiniteSupply().Cmp(totalSupply) == 0,
	}, nil
}

func convert33BytesPubKeyToEthAddress(pubKey string) (common.Address, error) {
	decoded, err := types.DecodeHex(pubKey)
	if err != nil {
		return common.Address{}, err
	}
	communityPubKey, err := crypto.DecompressPubkey(decoded)
	if err != nil {
		return common.Address{}, err
	}
	return common.Address(crypto.PubkeyToAddress(*communityPubKey)), nil
}

// Simpler version of hashing typed structured data alternative to typedStructuredDataHash. Keeping this for reference.
func customTypedStructuredDataHash(domainSeparator []byte, signatureTypedHash []byte, signer string, deployer string) types.Hash {
	// every field should be 32 bytes, eth address is 20 bytes so padding should be added
	emptyOffset := [12]byte{}
	hashedEncoded := crypto.Keccak256Hash(signatureTypedHash, emptyOffset[:], common.HexToAddress(signer).Bytes(),
		emptyOffset[:], common.HexToAddress(deployer).Bytes())
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", domainSeparator, hashedEncoded.Bytes()))
	return crypto.Keccak256Hash(rawData)
}

// Returns a typed structured hash according to https://eips.ethereum.org/EIPS/eip-712
// Domain separator from smart contract is used.
func typedStructuredDataHash(domainSeparator []byte, signer string, addressFrom string, deployerContractAddress string, chainID uint64) (types.Hash, error) {
	myTypedData := apitypes.TypedData{
		Types: apitypes.Types{
			"Deploy": []apitypes.Type{
				{Name: "signer", Type: "address"},
				{Name: "deployer", Type: "address"},
			},
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: "Deploy",
		// Domain field should be here to keep correct structure but
		// domainSeparator from smart contract is used.
		Domain: apitypes.TypedDataDomain{
			Name:              "CommunityTokenDeployer", // name from Deployer smart contract
			Version:           "1",                      // version from Deployer smart contract
			ChainId:           math.NewHexOrDecimal256(int64(chainID)),
			VerifyingContract: deployerContractAddress,
		},
		Message: apitypes.TypedDataMessage{
			"signer":   signer,
			"deployer": addressFrom,
		},
	}

	typedDataHash, err := myTypedData.HashStruct(myTypedData.PrimaryType, myTypedData.Message)
	if err != nil {
		return types.Hash{}, err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", domainSeparator, string(typedDataHash)))
	return crypto.Keccak256Hash(rawData), nil
}

// Creates
func (m *Manager) DeploymentSignatureDigest(chainID uint64, addressFrom string, communityID string) ([]byte, error) {
	callOpts := &bind.CallOpts{Pending: false}
	communityEthAddr, err := convert33BytesPubKeyToEthAddress(communityID)
	if err != nil {
		return nil, err
	}

	deployerAddr, err := communitytokendeployer.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}
	deployerContractInst, err := m.NewCommunityTokenDeployerInstance(chainID)
	if err != nil {
		return nil, err
	}

	domainSeparator, err := deployerContractInst.DOMAINSEPARATOR(callOpts)
	if err != nil {
		return nil, err
	}

	structedHash, err := typedStructuredDataHash(domainSeparator[:], communityEthAddr.Hex(), addressFrom, deployerAddr.Hex(), chainID)
	if err != nil {
		return nil, err
	}

	return structedHash.Bytes(), nil
}
