package ens

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/wealdtech/go-multicodec"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts/registrar"
	"github.com/status-im/status-go/contracts/resolver"
	"github.com/status-im/status-go/contracts/snt"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/ensresolver"
	"github.com/status-im/status-go/services/utils"
	wcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

func NewAPI(rpcClient *rpc.Client, accountsManager *account.GethManager, pendingTracker *transactions.PendingTxTracker, config *params.NodeConfig, appDb *sql.DB, timeSource func() time.Time, syncUserDetailFunc *syncUsernameDetail) *API {
	return &API{
		ensResolver: ensresolver.NewEnsResolver(rpcClient),

		accountsManager: accountsManager,
		pendingTracker:  pendingTracker,
		config:          config,
		db:              NewEnsDatabase(appDb),

		timeSource:         timeSource,
		syncUserDetailFunc: syncUserDetailFunc,
	}
}

type URI struct {
	Scheme string
	Host   string
	Path   string
}

// use this to avoid using messenger directly to avoid circular dependency (protocol->ens->protocol)
type syncUsernameDetail func(context.Context, *UsernameDetail) error

type API struct {
	ensResolver     *ensresolver.EnsResolver
	accountsManager *account.GethManager
	pendingTracker  *transactions.PendingTxTracker
	config          *params.NodeConfig

	db                 *Database
	syncUserDetailFunc *syncUsernameDetail

	timeSource func() time.Time
}

func (api *API) Stop() {
	api.ensResolver.Stop()
}

func (api *API) EnsResolver() *ensresolver.EnsResolver {
	return api.ensResolver
}

func (api *API) unixTime() uint64 {
	return uint64(api.timeSource().Unix())
}

func (api *API) GetEnsUsernames(ctx context.Context) ([]*UsernameDetail, error) {
	removed := false
	return api.db.GetEnsUsernames(&removed)
}

func (api *API) Add(ctx context.Context, chainID uint64, username string) error {
	ud := &UsernameDetail{Username: username, ChainID: chainID, Clock: api.unixTime()}
	err := api.db.AddEnsUsername(ud)
	if err != nil {
		return err
	}
	return (*api.syncUserDetailFunc)(ctx, ud)
}

func (api *API) Remove(ctx context.Context, chainID uint64, username string) error {
	ud := &UsernameDetail{Username: username, ChainID: chainID, Clock: api.unixTime()}
	affected, err := api.db.RemoveEnsUsername(ud)
	if err != nil {
		return err
	}
	if affected {
		return (*api.syncUserDetailFunc)(ctx, ud)
	}
	return nil
}

func (api *API) GetRegistrarAddress(ctx context.Context, chainID uint64) (common.Address, error) {
	return api.ensResolver.GetRegistrarAddress(ctx, chainID)
}

func (api *API) Resolver(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	return api.ensResolver.Resolver(ctx, chainID, username)
}

func (api *API) GetName(ctx context.Context, chainID uint64, address common.Address) (string, error) {
	return api.ensResolver.GetName(ctx, chainID, address)
}

func (api *API) OwnerOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	return api.ensResolver.OwnerOf(ctx, chainID, username)
}

func (api *API) ContentHash(ctx context.Context, chainID uint64, username string) ([]byte, error) {
	return api.ensResolver.ContentHash(ctx, chainID, username)
}

func (api *API) PublicKeyOf(ctx context.Context, chainID uint64, username string) (string, error) {
	return api.ensResolver.PublicKeyOf(ctx, chainID, username)
}

func (api *API) AddressOf(ctx context.Context, chainID uint64, username string) (*common.Address, error) {
	return api.ensResolver.AddressOf(ctx, chainID, username)
}

func (api *API) ExpireAt(ctx context.Context, chainID uint64, username string) (string, error) {
	return api.ensResolver.ExpireAt(ctx, chainID, username)
}

func (api *API) Price(ctx context.Context, chainID uint64) (string, error) {
	return api.ensResolver.Price(ctx, chainID)
}

// Deprecated: `Release` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) Release(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, password string, username string) (string, error) {
	registryAddr, err := api.ensResolver.GetRegistrarAddress(ctx, chainID)
	if err != nil {
		return "", err
	}

	signFn := utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password)
	tx, err := api.ensResolver.Release(ctx, chainID, registryAddr, txArgs, username, signFn)
	if err != nil {
		return "", err
	}

	err = api.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		registryAddr,
		transactions.ReleaseENS,
		transactions.AutoDelete,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return "", err
	}

	err = api.Remove(ctx, chainID, wcommon.FullDomainName(username))

	if err != nil {
		logutils.ZapLogger().Warn("Releasing ENS username: transaction successful, but removing failed")
	}

	return tx.Hash().String(), nil
}

// Deprecated: `ReleasePrepareTxCallMsg` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) ReleasePrepareTxCallMsg(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string) (ethereum.CallMsg, error) {
	registrarABI, err := abi.JSON(strings.NewReader(registrar.UsernameRegistrarABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	data, err := registrarABI.Pack("release", wcommon.UsernameToLabel(username))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	sntAddress, err := snt.ContractAddress(chainID)
	if err != nil {
		return ethereum.CallMsg{}, err
	}
	return ethereum.CallMsg{
		From:  common.Address(txArgs.From),
		To:    &sntAddress,
		Value: big.NewInt(0),
		Data:  data,
	}, nil
}

func (api *API) ReleasePrepareTx(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string) (interface{}, error) {
	callMsg, err := api.ReleasePrepareTxCallMsg(ctx, chainID, txArgs, username)
	if err != nil {
		return nil, err
	}

	return toCallArg(callMsg), nil
}

// Deprecated: `ReleaseEstimate` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) ReleaseEstimate(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string) (uint64, error) {
	callMsg, err := api.ReleasePrepareTxCallMsg(ctx, chainID, txArgs, username)
	if err != nil {
		return 0, err
	}

	return api.ensResolver.ReleaseEstimate(ctx, chainID, callMsg)
}

// Deprecated: `Register` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) Register(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, password string, username string, pubkey string) (string, error) {
	registryAddr, err := api.ensResolver.GetRegistrarAddress(ctx, chainID)
	if err != nil {
		return "", err
	}

	signFn := utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password)

	tx, err := api.ensResolver.Register(ctx, chainID, registryAddr, txArgs, username, pubkey, signFn)
	if err != nil {
		return "", err
	}

	err = api.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		registryAddr,
		transactions.RegisterENS,
		transactions.AutoDelete,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return "", err
	}

	err = api.Add(ctx, chainID, wcommon.FullDomainName(username))
	if err != nil {
		logutils.ZapLogger().Warn("Registering ENS username: transaction successful, but adding failed")
	}

	return tx.Hash().String(), nil
}

// Deprecated: `RegisterPrepareTxCallMsg` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) RegisterPrepareTxCallMsg(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string, pubkey string) (ethereum.CallMsg, error) {
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

	x, y := wcommon.ExtractCoordinates(pubkey)
	extraData, err := registrarABI.Pack("register", wcommon.UsernameToLabel(username), common.Address(txArgs.From), x, y)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	sntABI, err := abi.JSON(strings.NewReader(snt.SNTABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	registryAddr, err := api.ensResolver.GetRegistrarAddress(ctx, chainID)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	data, err := sntABI.Pack("approveAndCall", registryAddr, price, extraData)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	sntAddress, err := snt.ContractAddress(chainID)
	if err != nil {
		return ethereum.CallMsg{}, err
	}
	return ethereum.CallMsg{
		From:  common.Address(txArgs.From),
		To:    &sntAddress,
		Value: big.NewInt(0),
		Data:  data,
	}, nil
}

// Deprecated: `RegisterPrepareTx` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) RegisterPrepareTx(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string, pubkey string) (interface{}, error) {
	callMsg, err := api.RegisterPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return nil, err
	}

	return toCallArg(callMsg), nil
}

// Deprecated: `RegisterEstimate` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) RegisterEstimate(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string, pubkey string) (uint64, error) {
	callMsg, err := api.RegisterPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return 0, err
	}

	return api.ensResolver.RegisterEstimate(ctx, chainID, callMsg)
}

// Deprecated: `SetPubKey` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) SetPubKey(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, password string, username string, pubkey string) (string, error) {
	resolverAddress, err := api.Resolver(ctx, chainID, username)
	if err != nil {
		return "", err
	}

	signFn := utils.GetSigner(chainID, api.accountsManager, api.config.KeyStoreDir, txArgs.From, password)
	tx, err := api.ensResolver.SetPubKey(ctx, chainID, resolverAddress, txArgs, username, pubkey, signFn)
	if err != nil {
		return "", err
	}

	err = api.pendingTracker.TrackPendingTransaction(
		wcommon.ChainID(chainID),
		tx.Hash(),
		common.Address(txArgs.From),
		*resolverAddress,
		transactions.SetPubKey,
		transactions.AutoDelete,
		"",
	)
	if err != nil {
		logutils.ZapLogger().Error("TrackPendingTransaction error", zap.Error(err))
		return "", err
	}

	err = api.Add(ctx, chainID, wcommon.FullDomainName(username))

	if err != nil {
		logutils.ZapLogger().Warn("Registering ENS username: transaction successful, but adding failed")
	}

	return tx.Hash().String(), nil
}

// Deprecated: `SetPubKeyPrepareTxCallMsg` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) SetPubKeyPrepareTxCallMsg(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string, pubkey string) (ethereum.CallMsg, error) {
	err := wcommon.ValidateENSUsername(username)
	if err != nil {
		return ethereum.CallMsg{}, err
	}
	x, y := wcommon.ExtractCoordinates(pubkey)

	resolverABI, err := abi.JSON(strings.NewReader(resolver.PublicResolverABI))
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	data, err := resolverABI.Pack("setPubkey", wcommon.NameHash(username), x, y)
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

// Deprecated: `SetPubKeyPrepareTx` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) SetPubKeyPrepareTx(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string, pubkey string) (interface{}, error) {
	callMsg, err := api.SetPubKeyPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return nil, err
	}

	return toCallArg(callMsg), nil
}

// Deprecated: `SetPubKeyEstimate` was used before introducing a new, uniform, sending flow that uses router.
// Releasing ens username should start from calling `wallet_getSuggestedRoutesAsync`
// TODO: remove once mobile switches to a new sending flow.
func (api *API) SetPubKeyEstimate(ctx context.Context, chainID uint64, txArgs wallettypes.SendTxArgs, username string, pubkey string) (uint64, error) {
	callMsg, err := api.SetPubKeyPrepareTxCallMsg(ctx, chainID, txArgs, username, pubkey)
	if err != nil {
		return 0, err
	}

	return api.ensResolver.SetPubKeyEstimate(ctx, chainID, callMsg)
}

func (api *API) ResourceURL(ctx context.Context, chainID uint64, username string) (*URI, error) {
	scheme := "https"
	contentHash, err := api.ContentHash(ctx, chainID, username)
	if err != nil {
		return nil, err
	}

	if len(contentHash) == 0 {
		return &URI{}, nil
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

		parsedURL, _ := url.Parse(params.IpfsGatewayURL)
		// Remove scheme from the url
		host := parsedURL.Hostname() + parsedURL.Path + str
		return &URI{scheme, host, ""}, nil
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

		return &URI{scheme, string(decodedMHash.Digest), ""}, nil
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
		return &URI{scheme, "swarm-gateways.net", path}, nil
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
