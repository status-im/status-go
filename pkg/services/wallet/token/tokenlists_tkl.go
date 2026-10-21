package token

import (
	"context"
	"sort"
	"time"

	"github.com/status-im/nim-token-lists/go/tkl"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
)

// newTKLReadManager constructs the C-backed catalogue and its SQL bootstrap.
func newTKLReadManager(mng *Manager, chains []uint64, lastSuccess time.Time, options ...tklmanager.RefreshOptions) (*tklmanager.Manager, error) {
	config := tkl.Config{Chains: chains, MainListID: walletcommon.StatusTokenListID,
		RegistryID: remoteListOfTokenListsID, RegistryURL: remoteListOfTokenLists,
		Policy: tkl.Policy{SkippedKeys: walletcommon.SkippedTokenKeys()}}
	// The core defaults to ETH; Status supplies the native metadata for BSC.
	// Include both networks so later SetChains calls retain the same policy.
	for _, chain := range []uint64{walletcommon.BSCMainnet, walletcommon.BSCTestnet} {
		config.Policy.NativeTokens = append(config.Policy.NativeTokens, tkl.Token{
			ChainID: chain, Address: "0x0000000000000000000000000000000000000000",
			CrossChainID: "bsc-native", Name: walletcommon.BNBName, Symbol: walletcommon.BNBSymbol, Decimals: 18,
			LogoURI: "https://assets.coingecko.com/coins/images/825/thumb/bnb-icon2_2x.png?1696501970",
		})
	}
	ids := initialListIDsFromEmbedded()
	sort.Strings(ids)
	mainListID, registryID := config.MainListID, config.RegistryID
	formatOf := func(id string) string {
		switch id {
		case mainListID:
			return tkl.StatusFormat
		case registryID:
			return tkl.RegistryFormat
		}
		return tkl.StandardFormat
	}
	// Bundled bodies are the embedded lists themselves; the core only borrows them.
	bundled := make([]tkl.ListBody, 0, len(ids))
	for _, id := range ids {
		data, err := initialListProviderFromEmbedded(id)
		if err != nil {
			return nil, err
		}
		config.InitialLists = append(config.InitialLists, tkl.ListContent{ID: id, Format: formatOf(id), Source: types.LocalSourceURL, FetchedTimestamp: (time.Time{}).Format(time.RFC3339)})
		bundled = append(bundled, tkl.ListBody{ID: id, Origin: tkl.Bundled, Data: data})
	}
	for chain, addresses := range walletcommon.AdditionalNativeTokenAddresses() {
		for _, address := range addresses {
			config.Policy.NativeAliases = append(config.Policy.NativeAliases, tkl.Identity{ChainID: chain, Address: address.Hex()})
		}
	}
	return tklmanager.New(config, func(ctx context.Context) (tkl.Bootstrap, []tkl.ListBody, error) {
		bootstrap := tkl.Bootstrap{}
		if !lastSuccess.IsZero() {
			bootstrap.State = tkl.RefreshState{LastSuccess: lastSuccess.Unix(), HasSuccess: true}
		}
		contents, err := (&contentStore{walletDb: mng.walletDB}).getAll(ctx)
		if err != nil {
			return bootstrap, nil, err
		}
		bodies := append(make([]tkl.ListBody, 0, len(bundled)+len(contents)), bundled...)
		for id, content := range contents {
			row := tkl.ListContent{ID: id, Source: content.SourceURL, ETag: content.Etag, Format: formatOf(id)}
			if !content.Fetched.IsZero() {
				row.FetchedAt = content.Fetched.Unix()
				row.FetchedTimestamp = content.Fetched.UTC().Format(time.RFC3339)
			}
			bootstrap.Stored = append(bootstrap.Stored, row)
			bodies = append(bodies, tkl.ListBody{ID: id, Origin: tkl.Stored, Data: content.Data})
		}
		sort.Slice(bootstrap.Stored, func(i, j int) bool { return bootstrap.Stored[i].ID < bootstrap.Stored[j].ID })
		// Load only persisted ordinary metadata; community enrichment belongs to
		// the Go query layer and must not introduce uncancellable SQL here.
		rows, err := mng.walletDB.QueryContext(ctx, "SELECT address,name,symbol,decimals,network_id FROM tokens WHERE community_id IS NULL OR community_id = ''")
		if err != nil {
			return bootstrap, nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var token types.Token
			if err := rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.ChainID); err != nil {
				return bootstrap, nil, err
			}
			// Invalid customs are skipped by the core. Values outside
			// the ABI's uint8 field must be skipped before narrowing as well.
			if token.Decimals > 255 {
				continue
			}
			bootstrap.Customs = append(bootstrap.Customs, tkl.Token{ChainID: token.ChainID, Address: token.Address.Hex(), Decimals: uint8(token.Decimals), Name: token.Name, Symbol: token.Symbol, LogoURI: token.LogoURI, CrossChainID: token.CrossChainID, Custom: true})
		}
		if err := rows.Err(); err != nil {
			return bootstrap, nil, err
		}
		return bootstrap, bodies, ctx.Err()
	}, options...)
}
