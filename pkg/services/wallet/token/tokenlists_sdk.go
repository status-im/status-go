package token

import (
	"database/sql"
	"time"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/autofetcher"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/fetcher"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/manager"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/parsers"
	sdktypes "github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

// Kept only for runtime rollback until the native-only cutover.
func setUpTokenListsManager(mng *Manager, walletDB *sql.DB, enabledChains []uint64, lastUpdate time.Time,
	autoRefreshInterval time.Duration, autoRefreshCheckInterval time.Duration) (types.Catalogue, error) {

	wsdkFetcher := fetcher.New(fetcher.DefaultConfig())

	contentStore := NewContentStore(walletDB)

	customTokenStore := NewCustomTokenStore(mng)

	config := &manager.Config{
		AutoFetcherConfig: &autofetcher.ConfigRemoteListOfTokenLists{
			Config: autofetcher.Config{
				LastUpdate:               lastUpdate,
				AutoRefreshInterval:      autoRefreshInterval,
				AutoRefreshCheckInterval: autoRefreshCheckInterval,
			},
			RemoteListOfTokenListsFetchDetails: sdktypes.ListDetails{
				ID:        remoteListOfTokenListsID,
				SourceURL: remoteListOfTokenLists,
				Schema:    fetcher.ListOfTokenListsSchema,
			},
			RemoteListOfTokenListsParser: &parsers.StatusListOfTokenListsParser{},
		},

		MainListID: walletcommon.StatusTokenListID,

		InitialListIDs:      initialListIDsFromEmbedded(),
		InitialListProvider: initialListProviderFromEmbedded,

		CustomParsers: map[string]parsers.TokenListParser{
			walletcommon.StatusTokenListID: &parsers.StatusTokenListParser{},
		},

		Chains: enabledChains,

		SkippedTokenKeys: walletcommon.SkippedTokenKeys(),

		AdditionalAddressesForNativeToken: walletcommon.AdditionalNativeTokenAddresses(),
	}

	legacy, err := manager.New(config, wsdkFetcher, contentStore, customTokenStore)
	if err != nil {
		return nil, err
	}
	return &sdkCatalogue{Manager: legacy}, nil
}
