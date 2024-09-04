package anviltests

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"math/big"

	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	walletRequests "github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/walletdatabase"
)

const testAmount0Point1ETHInWei = 100000000000000000

var anvilHost string

type routerSuggestedRoutesEnvelope struct {
	Type   string                          `json:"type"`
	Routes responses.RouterSuggestedRoutes `json:"event"`
}

func initAnvil() {
	for _, arg := range os.Args {
		if strings.Contains(arg, "anvil-host") {
			anvilHost = strings.Split(arg, "=")[1]
			break
		}
	}
}

func setupSignalHandler(t *testing.T) chan responses.RouterSuggestedRoutes {
	suggestedRoutesCh := make(chan responses.RouterSuggestedRoutes)
	signalHandler := signal.MobileSignalHandler(func(data []byte) {
		var envelope signal.Envelope
		err := json.Unmarshal(data, &envelope)
		assert.NoError(t, err)
		if envelope.Type == string(signal.SuggestedRoutes) {
			var response routerSuggestedRoutesEnvelope
			err := json.Unmarshal(data, &response)
			assert.NoError(t, err)

			suggestedRoutesCh <- response.Routes
		}
	})
	signal.SetMobileSignalHandler(signalHandler)

	t.Cleanup(func() {
		close(suggestedRoutesCh)
		signal.ResetMobileSignalHandler()
	})

	return suggestedRoutesCh
}

func setupTestDB() (*sql.DB, func() error, error) {
	return helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "tests")
}

func setupTestWalletDB() (*sql.DB, func() error, error) {
	return helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "tests")
}

func setupTestMultiDB() (*multiaccounts.Database, func() error, error) {
	tmpfile, err := ioutil.TempFile("", "tests")
	if err != nil {
		return nil, nil, err
	}
	db, err := multiaccounts.InitializeDB(tmpfile.Name())
	if err != nil {
		return nil, nil, err
	}
	return db, func() error {
		err := db.Close()
		if err != nil {
			return err
		}
		return os.Remove(tmpfile.Name())
	}, nil
}

func setupGethStatusBackend(t *testing.T) (*api.GethStatusBackend, error) {
	db, stop1, err := setupTestDB()
	if err != nil {
		return nil, err
	}
	backend := api.NewGethStatusBackend()
	backend.StatusNode().SetAppDB(db)

	ma, stop2, err := setupTestMultiDB()
	if err != nil {
		return nil, err
	}
	backend.StatusNode().SetMultiaccountsDB(ma)

	walletDb, stop3, err := setupTestWalletDB()
	if err != nil {
		return nil, err
	}
	backend.StatusNode().SetWalletDB(walletDb)

	t.Cleanup(func() {
		require.NoError(t, stop1())
		require.NoError(t, stop2())
		require.NoError(t, stop3())
		require.NoError(t, backend.StopNode())
	})
	return backend, err
}

func setupWalletWithAnvil(t *testing.T, mnemonic string, password string, rootDataDir string) (*api.GethStatusBackend, error) {
	initAnvil()
	const pathEIP1581Root = "m/43'/60'/1581'"
	const pathEIP1581Chat = pathEIP1581Root + "/0'/0"
	const pathWalletRoot = "m/44'/60'/0'/0"
	const pathDefaultWalletAccount = pathWalletRoot + "/0"
	allGeneratedPaths := []string{pathEIP1581Root, pathEIP1581Chat, pathWalletRoot, pathDefaultWalletAccount}

	var err error

	keystoreContainsFileForAccount := func(keyStoreDir string, hexAddress string) bool {
		addrWithoutPrefix := strings.ToLower(hexAddress[2:])
		found := false
		err = filepath.Walk(keyStoreDir, func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fileInfo.IsDir() && strings.Contains(strings.ToUpper(path), strings.ToUpper(addrWithoutPrefix)) {
				found = true
			}
			return nil
		})
		return found
	}

	keyStoreDir := filepath.Join(rootDataDir, "keystore")

	utils.Init()

	backend, err := setupGethStatusBackend(t)
	require.NoError(t, err)

	backend.UpdateRootDataDir(rootDataDir)
	require.NoError(t, backend.AccountManager().InitKeystore(keyStoreDir))
	err = backend.OpenAccounts()
	require.NoError(t, err)

	genAccInfo, err := backend.AccountManager().AccountsGenerator().ImportMnemonic(mnemonic, "")
	assert.NoError(t, err)

	masterAddress := genAccInfo.Address

	accountInfo, err := backend.AccountManager().AccountsGenerator().StoreAccount(genAccInfo.ID, password)
	assert.NoError(t, err)

	found := keystoreContainsFileForAccount(keyStoreDir, accountInfo.Address)
	require.True(t, found)

	derivedAccounts, err := backend.AccountManager().AccountsGenerator().StoreDerivedAccounts(genAccInfo.ID, password, allGeneratedPaths)
	assert.NoError(t, err)

	chatAddress := derivedAccounts[pathEIP1581Chat].Address
	found = keystoreContainsFileForAccount(keyStoreDir, chatAddress)
	require.True(t, found)

	defaultSettings, err := api.DefaultSettings(genAccInfo.KeyUID, genAccInfo.Address, derivedAccounts)
	require.NoError(t, err)
	nodeConfig, err := api.DefaultNodeConfig(defaultSettings.InstallationID, &requests.CreateAccount{
		LogLevel: defaultSettings.LogLevel,
	})
	require.NoError(t, err)
	nodeConfig.DataDir = rootDataDir
	nodeConfig.KeyStoreDir = keyStoreDir

	nodeConfig.NetworkID = 1
	nodeConfig.HTTPEnabled = true
	nodeConfig.WalletConfig = params.WalletConfig{
		Enabled: true,
	}
	nodeConfig.UpstreamConfig = params.UpstreamRPCConfig{
		URL:     anvilHost,
		Enabled: true,
	}
	nodeConfig.Networks = []params.Network{
		params.Network{
			ChainID:                uint64(31337),
			ChainName:              "Anvil",
			DefaultRPCURL:          anvilHost,
			RPCURL:                 anvilHost,
			ShortName:              "eth",
			NativeCurrencyName:     "Ether",
			NativeCurrencySymbol:   "ETH",
			NativeCurrencyDecimals: 18,
			IsTest:                 false,
			Layer:                  1,
			Enabled:                true,
			// RelatedChainID:         goerliChainID,
		},
	}

	profileKeypair := &accounts.Keypair{
		KeyUID:      genAccInfo.KeyUID,
		Name:        "Profile Name",
		Type:        accounts.KeypairTypeProfile,
		DerivedFrom: masterAddress,
	}

	profileKeypair.Accounts = append(profileKeypair.Accounts, &accounts.Account{
		Address:   types.HexToAddress(chatAddress),
		KeyUID:    profileKeypair.KeyUID,
		Type:      accounts.AccountTypeGenerated,
		PublicKey: types.Hex2Bytes(accountInfo.PublicKey),
		Path:      pathEIP1581Chat,
		Wallet:    false,
		Chat:      true,
		Name:      "GeneratedAccount",
	})

	for p, dAccInfo := range derivedAccounts {
		found = keystoreContainsFileForAccount(keyStoreDir, dAccInfo.Address)
		require.NoError(t, err)
		require.True(t, found)

		if p == pathDefaultWalletAccount {
			wAcc := &accounts.Account{
				Address: types.HexToAddress(dAccInfo.Address),
				KeyUID:  genAccInfo.KeyUID,
				Wallet:  false,
				Chat:    false,
				Type:    accounts.AccountTypeGenerated,
				Path:    p,
				Name:    "derivacc" + p,
				Hidden:  false,
				Removed: false,
			}
			if p == pathDefaultWalletAccount {
				wAcc.Wallet = true
			}
			profileKeypair.Accounts = append(profileKeypair.Accounts, wAcc)
		}
	}

	account := multiaccounts.Account{
		Name:      profileKeypair.Name,
		Timestamp: 1,
		KeyUID:    profileKeypair.KeyUID,
	}

	err = backend.EnsureAppDBOpened(account, password)
	require.NoError(t, err)

	err = backend.StartNodeWithAccountAndInitialConfig(account, password, *defaultSettings, nodeConfig, profileKeypair.Accounts, nil)
	require.NoError(t, err)

	return backend, nil
}

func TestRouterFeesUpdate(t *testing.T) {
	const mnemonic = "test test test test test test test test test test test junk"
	const password = "111111"
	rootDataDir := t.TempDir()

	backend, err := setupWalletWithAnvil(t, mnemonic, password, rootDataDir)
	require.NoError(t, err)

	statusNode := backend.StatusNode()
	require.NotNil(t, statusNode)

	walletService := statusNode.WalletService()
	require.NotNil(t, walletService)

	rpcClient := walletService.GetRPCClient()
	transactor := walletService.GetTransactor()
	tokenManager := walletService.GetTokenManager()
	ensService := walletService.GetEnsService()
	stickersService := walletService.GetStickersService()

	walletRouter := router.NewRouter(rpcClient, transactor, tokenManager, walletService.GetMarketManager(), walletService.GetCollectiblesService(),
		walletService.GetCollectiblesManager(), ensService, stickersService)
	require.NotNil(t, walletRouter)
	defer walletRouter.StopSuggestedRoutesAsyncCalculation()

	transfer := pathprocessor.NewTransferProcessor(rpcClient, transactor)
	walletRouter.AddPathProcessor(transfer)

	erc721Transfer := pathprocessor.NewERC721Processor(rpcClient, transactor)
	walletRouter.AddPathProcessor(erc721Transfer)

	erc1155Transfer := pathprocessor.NewERC1155Processor(rpcClient, transactor)
	walletRouter.AddPathProcessor(erc1155Transfer)

	hop := pathprocessor.NewHopBridgeProcessor(rpcClient, transactor, tokenManager, rpcClient.NetworkManager)
	walletRouter.AddPathProcessor(hop)

	paraswap := pathprocessor.NewSwapParaswapProcessor(rpcClient, transactor, tokenManager)
	walletRouter.AddPathProcessor(paraswap)

	ensRegister := pathprocessor.NewENSReleaseProcessor(rpcClient, transactor, ensService)
	walletRouter.AddPathProcessor(ensRegister)

	ensRelease := pathprocessor.NewENSReleaseProcessor(rpcClient, transactor, ensService)
	walletRouter.AddPathProcessor(ensRelease)

	ensPublicKey := pathprocessor.NewENSPublicKeyProcessor(rpcClient, transactor, ensService)
	walletRouter.AddPathProcessor(ensPublicKey)

	buyStickers := pathprocessor.NewStickersBuyProcessor(rpcClient, transactor, stickersService)
	walletRouter.AddPathProcessor(buyStickers)

	suggestedRoutesCh := setupSignalHandler(t)

	input := &walletRequests.RouteInputParams{
		TestnetMode:          false,
		Uuid:                 uuid.NewString(),
		SendType:             sendtype.Transfer,
		AddrFrom:             common.HexToAddress("0xa0Ee7A142d267C1f36714E4a8F75612F20a79720"),
		AddrTo:               common.HexToAddress("0x1"),
		AmountIn:             (*hexutil.Big)(big.NewInt(testAmount0Point1ETHInWei)),
		TokenID:              pathprocessor.EthSymbol,
		DisabledFromChainIDs: []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},
		DisabledToChainIDs:   []uint64{walletCommon.OptimismMainnet, walletCommon.ArbitrumMainnet},

		TestsMode: false,
	}

	// run the suggested routes calculation and cancel it after initial results are received
	receivedResultsCounter := 0
	quitTimer := time.NewTimer(5 * time.Second)
	walletRouter.SuggestedRoutesAsync(input)

loop1:
	for {
		select {
		case asyncRoutes := <-suggestedRoutesCh:
			receivedResultsCounter++
			walletRouter.StopSuggestedRoutesAsyncCalculation()
			t.Log("canceled, async route uuid", asyncRoutes.Uuid)
		case <-quitTimer.C:
			break loop1
		}
	}

	require.False(t, quitTimer.Stop())
	require.Equal(t, 1, receivedResultsCounter)

	// run the suggested routes calculation and let it finish and check if updates are received
	receivedResultsCounter = 0
	quitTimer = time.NewTimer(5 * time.Second)
	walletRouter.SuggestedRoutesAsync(input)

loop2:
	for {
		select {
		case asyncRoutes := <-suggestedRoutesCh:
			t.Log("received, async route uuid", asyncRoutes.Uuid)
			receivedResultsCounter++
		case <-quitTimer.C:
			break loop2
		}
	}

	require.False(t, quitTimer.Stop())
	require.Greater(t, receivedResultsCounter, 1)
}
