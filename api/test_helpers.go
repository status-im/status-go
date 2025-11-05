package api

import (
	"encoding/json"
	"path"
	"testing"

	"github.com/status-im/status-go/crypto/types"
	accscommon "github.com/status-im/status-go/internal/accounts-management/common"
	generator2 "github.com/status-im/status-go/internal/accounts-management/generator"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/messaging"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"

	"github.com/stretchr/testify/require"
)

const (
	testPassword = "test-password"
)

var (
	networks = json.RawMessage("{}")
)

type setupContext struct {
	backend        *GethStatusBackend
	mnemonic       string
	settings       settings.Settings
	config         *params.NodeConfig
	multiAcc       *multiaccounts.Account
	profileKeypair *accsmanagementtypes.Keypair
	chatPrivateKey string
}

func setupTestContext(t *testing.T, password string, storeProfile bool, storeMultiAcc bool, useDefaultSettings bool) (data *setupContext) {
	tmpdir := t.TempDir()

	data = &setupContext{}

	var err error
	data.mnemonic, err = accscommon.CreateRandomMnemonicWithDefaultLength()
	require.NoError(t, err)

	genMasterAcc, err := generator2.CreateAccountFromMnemonic(data.mnemonic, "")
	require.NoError(t, err)

	accountsPaths := []string{accscommon.PathWalletRoot, accscommon.PathEIP1581Chat, accscommon.PathDefaultWalletAccount}
	derivedAccs, err := generator2.DeriveChildrenFromAccount(genMasterAcc, append([]string{accscommon.PathWalletRoot}, accountsPaths...))
	require.NoError(t, err)

	data.profileKeypair = &accsmanagementtypes.Keypair{
		KeyUID:      genMasterAcc.KeyUID(),
		Name:        "Test Keypair",
		Type:        accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom: genMasterAcc.Address().Hex(),
	}

	data.chatPrivateKey = derivedAccs[accscommon.PathEIP1581Chat].PrivateKeyHex()

	for path, acc := range derivedAccs {
		if path != accscommon.PathEIP1581Chat && path != accscommon.PathDefaultWalletAccount {
			continue
		}

		data.profileKeypair.Accounts = append(data.profileKeypair.Accounts, &accsmanagementtypes.Account{
			Address:   acc.Address(),
			KeyUID:    genMasterAcc.KeyUID(),
			Path:      path,
			PublicKey: types.Hex2Bytes(acc.PublicKeyHex()),
			Wallet:    path == accscommon.PathDefaultWalletAccount,
			Chat:      path == accscommon.PathEIP1581Chat,
		})
	}

	data.multiAcc = &multiaccounts.Account{
		Name:           "foo",
		Timestamp:      1,
		KeycardPairing: "pairing",
		KeyUID:         genMasterAcc.KeyUID(),
		KDFIterations:  1,
	}

	if useDefaultSettings {
		derivedAddresses := make(map[string]generator2.AccountInfo)
		for path, acc := range derivedAccs {
			derivedAddresses[path] = generator2.AccountInfo{
				Address:   acc.Address().Hex(),
				PublicKey: acc.PublicKeyHex(),
			}
		}

		defSettings, err := defaultSettings(genMasterAcc.KeyUID(), genMasterAcc.Address().Hex(), derivedAddresses)
		require.NoError(t, err)

		data.settings = *defSettings

		data.config, err = DefaultNodeConfig(data.settings.InstallationID, genMasterAcc.KeyUID(), &requests.CreateAccount{
			LogLevel: data.settings.LogLevel,
		})
		require.NoError(t, err)
		data.config.RootDataDir = tmpdir
	} else {
		json := `{
		"NetworkId": 3,
		"RootDataDir": "` + tmpdir + `",
		"KeycardPairingDataFile": "` + path.Join(tmpdir, "keycard/pairings.json") + `",
		"NoDiscovery": true,
		"TorrentConfig": {
			"Port": 9025,
			"Enabled": false,
			"DataDir": "` + tmpdir + `/archivedata",
			"TorrentDir": "` + tmpdir + `/torrents"
		},
		"RuntimeLogLevel": "INFO",
		"LogLevel": "DEBUG"
	}`

		data.config, err = params.NewConfigFromJSON(json)
		require.NoError(t, err)

		data.config, err = params.NewNodeConfig(tmpdir, 178733)
		require.NoError(t, err)

		data.settings = settings.Settings{
			KeyUID:            genMasterAcc.KeyUID(),
			Address:           genMasterAcc.Address(),
			DisplayName:       "UserDisplayName",
			CurrentNetwork:    "mainnet_rpc",
			DappsAddress:      derivedAccs[accscommon.PathDefaultWalletAccount].Address(),
			InstallationID:    messaging.GenerateInstallationID(),
			LatestDerivedPath: 0,
			Name:              "Jittery Cornflowerblue Kingbird",
			Networks:          &networks,
			PhotoPath:         "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
			PreviewPrivacy:    false,
			PublicKey:         genMasterAcc.PublicKeyHex(),
			SigningPhrase:     "yurt joey vibe",
			WalletRootAddress: derivedAccs[accscommon.PathWalletRoot].Address(),
		}
	}

	data.backend = NewGethStatusBackend(tt.MustCreateTestLogger())
	data.backend.UpdateRootDataDir(tmpdir)

	err = data.backend.OpenAccounts(true)
	require.NoError(t, err)

	if storeMultiAcc {
		err = data.backend.SaveAccount(*data.multiAcc)
		require.NoError(t, err)
	}

	if storeProfile {
		err = data.backend.ensureDBsOpened(*data.multiAcc, password)
		require.NoError(t, err)

		keypair, err := data.backend.AccountsManager().CreateKeypairFromMnemonicAndStore(data.mnemonic, password, "Test Keypair",
			&accsmanagementtypes.AccountCreationDetails{
				Path: accscommon.PathDefaultWalletAccount,
			}, true, 0)
		require.NoError(t, err)

		data.profileKeypair = keypair
	}

	return
}
