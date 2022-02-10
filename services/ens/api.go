package ens

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/wealdtech/go-multicodec"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/erc20"
	"github.com/status-im/status-go/services/ens/registrar"
	"github.com/status-im/status-go/services/ens/resolver"
	"github.com/status-im/status-go/services/rpcfilters"
	"github.com/status-im/status-go/transactions"
)

func NewAPI(rpcClient *rpc.Client, accountsManager *account.GethManager, rpcFiltersSrvc *rpcfilters.Service, config *params.NodeConfig) *API {
	return &API{
		contractMaker: &contractMaker{
			RPCClient: rpcClient,
		},
		accountsManager: accountsManager,
		rpcFiltersSrvc:  rpcFiltersSrvc,
		config:          config,
	}
}

type uri struct {
	Scheme string
	Host   string
	Path   string
}

type API struct {
	contractMaker   *contractMaker
	accountsManager *account.GethManager
	rpcFiltersSrvc  *rpcfilters.Service
	config          *params.NodeConfig
}

func (api *API) Resolver(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	err := validateENSUsername(username)
	if err != nil {
		return nil, err
	}

	registry, err := api.contractMaker.newRegistry(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	resolver, err := registry.Resolver(callOpts, nameHash(username))
	if err != nil {
		return nil, err
	}

	return &resolver, nil
}

func (api *API) OwnerOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	err := validateENSUsername(username)
	if err != nil {
		return nil, err
	}

	registry, err := api.contractMaker.newRegistry(chainID)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	owner, err := registry.Owner(callOpts, nameHash(username))
	if err != nil {
		return nil, nil
	}

	return &owner, nil
}

func (api *API) ContentHash(ctx context.Context, chainID uint64, username string) ([]byte, error) {
	err := validateENSUsername(username)
	if err != nil {
		return nil, err
	}

	resolverAddress, err := api.Resolver(ctx, chainID, username)
	if err != nil {
		return nil, err
	}

	resolver, err := api.contractMaker.newPublicResolver(chainID, resolverAddress)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	contentHash, err := resolver.Contenthash(callOpts, nameHash(username))
	if err != nil {
		return nil, nil
	}

	return contentHash, nil
}

func (api *API) PublicKeyOf(ctx context.Context, chainID uint64, username string) (string, error) {
	err := validateENSUsername(username)
	if err != nil {
		return "", err
	}

	resolverAddress, err := api.Resolver(ctx, chainID, username)
	if err != nil {
		return "", err
	}

	resolver, err := api.contractMaker.newPublicResolver(chainID, resolverAddress)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	pubKey, err := resolver.Pubkey(callOpts, nameHash(username))
	if err != nil {
		return "", err
	}
	return "0x04" + hex.EncodeToString(pubKey.X[:]) + hex.EncodeToString(pubKey.Y[:]), nil
}

func (api *API) AddressOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	err := validateENSUsername(username)
	if err != nil {
		return nil, err
	}

	resolverAddress, err := api.Resolver(ctx, chainID, username)
	if err != nil {
		return nil, err
	}

	resolver, err := api.contractMaker.newPublicResolver(chainID, resolverAddress)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	addr, err := resolver.Addr(callOpts, nameHash(username))
	if err != nil {
		return nil, err
	}

	return &addr, nil
}

func (api *API) ExpireAt(ctx context.Context, chainID uint64, username string) (string, error) {
	registrar, err := api.contractMaker.newUsernameRegistrar(chainID)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	expTime, err := registrar.GetExpirationTime(callOpts, usernameToLabel(username))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", expTime), nil
}

func (api *API) Price(ctx context.Context, chainID uint64) (string, error) {
	registrar, err := api.contractMaker.newUsernameRegistrar(chainID)
	if err != nil {
		return "", err
	}

	callOpts := &bind.CallOpts{Context: ctx, Pending: false}
	price, err := registrar.GetPrice(callOpts)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", price), nil
}

func (api *API) getSigner(chainID uint64, from types.Address, password string) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		selectedAccount, err := api.accountsManager.VerifyAccountPassword(api.config.KeyStoreDir, from.Hex(), password)
		if err != nil {
			return nil, err
		}
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, selectedAccount.PrivateKey)
	}
}

func (api *API) Release(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, password string, username string) (string, error) {
	registrar, err := api.contractMaker.newUsernameRegistrar(chainID)
	if err != nil {
		return "", err
	}

	txOpts := txArgs.ToTransactOpts(api.getSigner(chainID, txArgs.From, password))
	tx, err := registrar.Release(txOpts, usernameToLabel(username))
	if err != nil {
		return "", err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(types.Hash(tx.Hash()))
	return tx.Hash().String(), nil
}

func (api *API) ReleaseEstimate(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, username string) (uint64, error) {
	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return 0, err
	}

	data, err := registrarABI.Pack("release", nameHash(username))
	if err != nil {
		return 0, err
	}

	ethClient, err := api.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	registrarAddress := usernameRegistrarsByChainID[chainID]
	return ethClient.EstimateGas(ctx, ethereum.CallMsg{
		From:  common.Address(txArgs.From),
		To:    &registrarAddress,
		Value: big.NewInt(0),
		Data:  data,
	})
}

func (api *API) Register(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, password string, username string, pubkey string) (string, error) {
	snt, err := api.contractMaker.newSNT(chainID)
	if err != nil {
		return "", err
	}

	priceHex, err := api.Price(ctx, chainID)
	if err != nil {
		return "", err
	}
	price := new(big.Int)
	price.SetString(priceHex, 16)

	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return "", err
	}

	x, y := extractCoordinates(pubkey)
	extraData, err := registrarABI.Pack("register", usernameToLabel(username), common.Address(txArgs.From), x, y)
	if err != nil {
		return "", err
	}

	txOpts := txArgs.ToTransactOpts(api.getSigner(chainID, txArgs.From, password))
	tx, err := snt.ApproveAndCall(
		txOpts,
		usernameRegistrarsByChainID[chainID],
		price,
		extraData,
	)

	if err != nil {
		return "", err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(types.Hash(tx.Hash()))
	return tx.Hash().String(), nil
}

func (api *API) RegisterPrepareTxCallMsg(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, username string, pubkey string) (ethereum.CallMsg, error) {
	priceHex, err := api.Price(ctx, chainID)
	if err != nil {
		return ethereum.CallMsg{}, err
	}
	price := new(big.Int)
	price.SetString(priceHex, 16)

	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	x, y := extractCoordinates(pubkey)
	extraData, err := registrarABI.Pack("register", usernameToLabel(username), common.Address(txArgs.From), x, y)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	sntABI, err := abi.JSON(strings.NewReader(erc20.SNTABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	data, err := sntABI.Pack("approveAndCall", usernameRegistrarsByChainID[chainID], price, extraData)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	contractAddress := sntByChainID[chainID]

	return ethereum.CallMsg{
		From:  common.Address(txArgs.From),
		To:    &contractAddress,
		Value: big.NewInt(0),
		Data:  data,
	}, nil
}

func (api *API) RegisterPrepareTx(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, username string, pubkey string) (interface{}, error) {
	callMsg, err := api.RegisterPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return nil, err
	}

	return toCallArg(callMsg), nil
}

func (api *API) RegisterEstimate(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, username string, pubkey string) (uint64, error) {
	ethClient, err := api.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	callMsg, err := api.RegisterPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return 0, err
	}

	return ethClient.EstimateGas(ctx, callMsg)
}

func (api *API) SetPubKey(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, password string, username string, pubkey string) (string, error) {
	err := validateENSUsername(username)
	if err != nil {
		return "", err
	}

	resolverAddress, err := api.Resolver(ctx, chainID, username)
	if err != nil {
		return "", err
	}

	resolver, err := api.contractMaker.newPublicResolver(chainID, resolverAddress)
	if err != nil {
		return "", err
	}

	x, y := extractCoordinates(pubkey)
	txOpts := txArgs.ToTransactOpts(api.getSigner(chainID, txArgs.From, password))
	tx, err := resolver.SetPubkey(txOpts, nameHash(username), x, y)
	if err != nil {
		return "", err
	}

	go api.rpcFiltersSrvc.TriggerTransactionSentToUpstreamEvent(types.Hash(tx.Hash()))
	return tx.Hash().String(), nil
}

func (api *API) SetPubKeyPrepareTxCallMsg(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, username string, pubkey string) (ethereum.CallMsg, error) {
	err := validateENSUsername(username)
	if err != nil {
		return ethereum.CallMsg{}, err
	}
	x, y := extractCoordinates(pubkey)

	resolverABI, err := abi.JSON(strings.NewReader(resolver.PublicResolverABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	data, err := resolverABI.Pack("setPubkey", nameHash(username), x, y)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	resolverAddress, err := api.Resolver(ctx, chainID, username)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	return ethereum.CallMsg{
		From:  common.Address(txArgs.From),
		To:    resolverAddress,
		Value: big.NewInt(0),
		Data:  data,
	}, nil
}

func (api *API) SetPubKeyPrepareTx(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, username string, pubkey string) (interface{}, error) {
	callMsg, err := api.SetPubKeyPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return nil, err
	}

	return toCallArg(callMsg), nil
}

func (api *API) SetPubKeyEstimate(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, username string, pubkey string) (uint64, error) {
	ethClient, err := api.contractMaker.RPCClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	callMsg, err := api.SetPubKeyPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return 0, err
	}

	return ethClient.EstimateGas(ctx, callMsg)
}

func (api *API) ResourceURL(ctx context.Context, chainID uint64, username string) (*uri, error) {
	scheme := "https"
	contentHash, err := api.ContentHash(ctx, chainID, username)
	if err != nil {
		return nil, err
	}

	if len(contentHash) == 0 {
		return &uri{}, nil
	}

	data, codec, err := multicodec.RemoveCodec(contentHash)
	if err != nil {
		return nil, err
	}
	codecName, err := multicodec.Name(codec)
	if err != nil {
		return nil, err
	}

	switch codecName {
	case "ipfs-ns":
		thisCID, err := cid.Parse(data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse CID")
		}
		str, err := thisCID.StringOfBase(multibase.Base32)
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain base36 representation")
		}
		host := str + ".ipfs.cf-ipfs.com"
		return &uri{scheme, host, ""}, nil
	case "ipns-ns":
		id, offset := binary.Uvarint(data)
		if id == 0 {
			return nil, fmt.Errorf("unknown CID")
		}

		data, _, err := multicodec.RemoveCodec(data[offset:])
		if err != nil {
			return nil, err
		}
		decodedMHash, err := multihash.Decode(data)
		if err != nil {
			return nil, err
		}

		return &uri{scheme, string(decodedMHash.Digest), ""}, nil
	case "swarm-ns":
		id, offset := binary.Uvarint(data)
		if id == 0 {
			return nil, fmt.Errorf("unknown CID")
		}
		data, _, err := multicodec.RemoveCodec(data[offset:])
		if err != nil {
			return nil, err
		}
		decodedMHash, err := multihash.Decode(data)
		if err != nil {
			return nil, err
		}
		path := "/bzz:/" + hex.EncodeToString(decodedMHash.Digest) + "/"
		return &uri{scheme, "swarm-gateways.net", path}, nil
	default:
		return nil, fmt.Errorf("unknown codec name %s", codecName)
	}
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
