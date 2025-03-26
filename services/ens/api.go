package ens

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/wealdtech/go-multicodec"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ens/ensresolver"
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
