//go:build tkl

package token

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/manager"
	"github.com/status-im/nim-token-lists/go/tkl"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
)

// newTKLReadManager constructs the optional catalogue without selecting it as
// the application's manager. Refresh and durable custom writes are integrated
// before the runtime selection is enabled.
func newTKLReadManager(mng *Manager, chains []uint64, lastSuccess time.Time, options ...tklmanager.RefreshOptions) (*tklmanager.Manager, error) {
	config := tkl.Config{Chains: chains, MainListID: walletcommon.StatusTokenListID,
		RegistryID: remoteListOfTokenListsID, RegistryURL: remoteListOfTokenLists,
		Policy: tkl.Policy{SkippedKeys: walletcommon.SkippedTokenKeys()}}
	// The core defaults to ETH; Status supplies the native metadata for BSC.
	// Include both networks so later SetChains calls retain the same policy.
	for _, chain := range []uint64{walletcommon.BSCMainnet, walletcommon.BSCTestnet} {
		config.Policy.NativeTokens = append(config.Policy.NativeTokens, tkl.Token{
			ChainID: chain, Address: "0x0000000000000000000000000000000000000000",
			CrossChainID: "bsc-native", Name: "BNB", Symbol: "BNB", Decimals: 18,
			LogoURI: "https://assets.coingecko.com/coins/images/825/thumb/bnb-icon2_2x.png?1696501970",
		})
	}
	ids := initialListIDsFromEmbedded()
	sort.Strings(ids)
	for _, id := range ids {
		data, err := initialListProviderFromEmbedded(id)
		if err != nil {
			return nil, err
		}
		format := tkl.StandardFormat
		if id == config.MainListID {
			format = tkl.StatusFormat
		}
		config.InitialLists = append(config.InitialLists, tkl.ListContent{ID: id, Body: string(data), Format: format, Source: manager.LocalSourceURL, FetchedTimestamp: (time.Time{}).Format(time.RFC3339)})
	}
	for chain, addresses := range walletcommon.AdditionalNativeTokenAddresses() {
		for _, address := range addresses {
			config.Policy.NativeAliases = append(config.Policy.NativeAliases, tkl.Identity{ChainID: chain, Address: address.Hex()})
		}
	}
	return tklmanager.New(config, func(ctx context.Context) (tkl.Bootstrap, error) {
		bootstrap := tkl.Bootstrap{}
		if !lastSuccess.IsZero() {
			bootstrap.State = tkl.RefreshState{LastSuccess: lastSuccess.Unix(), HasSuccess: true}
		}
		contents, err := NewContentStore(mng.walletDB).GetAll()
		if err != nil {
			return bootstrap, err
		}
		for id, content := range contents {
			format := tkl.StandardFormat
			if id == config.MainListID {
				format = tkl.StatusFormat
			}
			if id == config.RegistryID {
				format = tkl.RegistryFormat
			}
			row := tkl.ListContent{ID: id, Body: string(content.Data), Source: content.SourceURL, ETag: content.Etag, Format: format}
			if !content.Fetched.IsZero() {
				row.FetchedAt = content.Fetched.Unix()
				row.FetchedTimestamp = content.Fetched.UTC().Format(time.RFC3339)
			}
			bootstrap.Contents = append(bootstrap.Contents, row)
		}
		sort.Slice(bootstrap.Contents, func(i, j int) bool { return bootstrap.Contents[i].ID < bootstrap.Contents[j].ID })
		customs, err := NewCustomTokenStore(mng).GetAll()
		if err != nil {
			return bootstrap, err
		}
		for _, token := range customs {
			if token == nil {
				return bootstrap, fmt.Errorf("nil stored custom token")
			}
			// Invalid customs are skipped by the SDK and core. Values outside
			// the ABI's uint8 field must be skipped before narrowing as well.
			if token.Decimals > 255 {
				continue
			}
			bootstrap.Customs = append(bootstrap.Customs, tkl.Token{ChainID: token.ChainID, Address: token.Address.Hex(), Decimals: uint8(token.Decimals), Name: token.Name, Symbol: token.Symbol, LogoURI: token.LogoURI, CrossChainID: token.CrossChainID, Custom: true})
		}
		return bootstrap, ctx.Err()
	}, options...)
}
