package accounts

import (
	accsmanagementcommon "github.com/status-im/status-go/internal/accounts-management/common"
	generator "github.com/status-im/status-go/internal/accounts-management/generator"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	types "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/common"
)

const (
	testProfileMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"
	testSeedMnemonic    = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	testSeed2Mnemonic   = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about abandon"
)

func GetWatchOnlyAccountsForTest() []*accsmanagementtypes.Account {
	wo1 := &accsmanagementtypes.Account{
		Address: types.Address{0x11},
		Type:    accsmanagementtypes.AccountTypeWatch,
		Name:    "WatchOnlyAcc1",
		ColorID: common.CustomizationColorPrimary,
		Emoji:   "emoji-1",
	}
	wo2 := &accsmanagementtypes.Account{
		Address: types.Address{0x12},
		Type:    accsmanagementtypes.AccountTypeWatch,
		Name:    "WatchOnlyAcc2",
		ColorID: common.CustomizationColorPrimary,
		Emoji:   "emoji-1",
	}
	wo3 := &accsmanagementtypes.Account{
		Address: types.Address{0x13},
		Type:    accsmanagementtypes.AccountTypeWatch,
		Name:    "WatchOnlyAcc3",
		ColorID: common.CustomizationColorPrimary,
		Emoji:   "emoji-1",
	}

	return []*accsmanagementtypes.Account{wo1, wo2, wo3}
}

func GetProfileKeypairForTest(includeChatAccount bool, includeDefaultWalletAccount bool, includeAdditionalAccounts bool) (
	keypair *accsmanagementtypes.Keypair, acc *generator.Account, derivedAccs map[string]*generator.Account, err error) {
	acc, derivedAccs, err = generator.CreateAndDeriveAccountsFromMnemonic(testProfileMnemonic, []string{
		accsmanagementcommon.PathEIP1581Chat,
		accsmanagementcommon.PathDefaultWalletAccount,
		accsmanagementcommon.CustomWalletPath1,
		accsmanagementcommon.CustomWalletPath2,
	}, "")
	if err != nil {
		return
	}

	keypair = &accsmanagementtypes.Keypair{
		KeyUID:      acc.KeyUID(),
		Name:        "Profile Name",
		Type:        accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom: acc.Address().Hex(),
	}

	if includeChatAccount {
		profileAccount := &accsmanagementtypes.Account{
			Address:               derivedAccs[accsmanagementcommon.PathEIP1581Chat].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                false,
			Chat:                  true,
			Type:                  accsmanagementtypes.AccountTypeGenerated,
			Path:                  accsmanagementcommon.PathEIP1581Chat,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathEIP1581Chat].PublicKeyHex()),
			Name:                  "Profile Name",
			Operable:              accsmanagementtypes.AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, profileAccount)
	}

	if includeDefaultWalletAccount {
		defaultWalletAccount := &accsmanagementtypes.Account{
			Address:               derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                true,
			Chat:                  false,
			Type:                  accsmanagementtypes.AccountTypeGenerated,
			Path:                  accsmanagementcommon.PathDefaultWalletAccount,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].PublicKeyHex()),
			Name:                  "Generated Acc 1",
			Emoji:                 "emoji-1",
			ColorID:               common.CustomizationColorPrimary,
			Hidden:                false,
			Clock:                 0,
			Removed:               false,
			Operable:              accsmanagementtypes.AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, defaultWalletAccount)
		keypair.LastUsedDerivationIndex = 0
	}

	if includeAdditionalAccounts {
		generatedWalletAccount1 := &accsmanagementtypes.Account{
			Address:               derivedAccs[accsmanagementcommon.CustomWalletPath1].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                false,
			Chat:                  false,
			Type:                  accsmanagementtypes.AccountTypeGenerated,
			Path:                  accsmanagementcommon.CustomWalletPath1,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.CustomWalletPath1].PublicKeyHex()),
			Name:                  "Generated Acc 2",
			Emoji:                 "emoji-2",
			ColorID:               common.CustomizationColorPrimary,
			Hidden:                false,
			Clock:                 0,
			Removed:               false,
			Operable:              accsmanagementtypes.AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, generatedWalletAccount1)
		keypair.LastUsedDerivationIndex = 1

		generatedWalletAccount2 := &accsmanagementtypes.Account{
			Address:               derivedAccs[accsmanagementcommon.CustomWalletPath2].Address(),
			KeyUID:                keypair.KeyUID,
			Wallet:                false,
			Chat:                  false,
			Type:                  accsmanagementtypes.AccountTypeGenerated,
			Path:                  accsmanagementcommon.CustomWalletPath2,
			PublicKey:             types.Hex2Bytes(derivedAccs[accsmanagementcommon.CustomWalletPath2].PublicKeyHex()),
			Name:                  "Generated Acc 3",
			Emoji:                 "emoji-3",
			ColorID:               common.CustomizationColorPrimary,
			Hidden:                false,
			Clock:                 0,
			Removed:               false,
			Operable:              accsmanagementtypes.AccountFullyOperable,
			ProdPreferredChainIDs: "1",
			TestPreferredChainIDs: "5",
		}
		keypair.Accounts = append(keypair.Accounts, generatedWalletAccount2)
		keypair.LastUsedDerivationIndex = 2
	}

	return
}

func GetSeedImportedKeypair1ForTest() (keypair *accsmanagementtypes.Keypair, acc *generator.Account, derivedAccs map[string]*generator.Account, err error) {
	acc, derivedAccs, err = generator.CreateAndDeriveAccountsFromMnemonic(testSeedMnemonic, []string{
		accsmanagementcommon.PathDefaultWalletAccount,
		accsmanagementcommon.CustomWalletPath1,
	}, "")
	if err != nil {
		return
	}

	keypair = &accsmanagementtypes.Keypair{
		KeyUID:      acc.KeyUID(),
		Name:        "Seed Imported 1",
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: acc.Address().Hex(),
	}

	seedGeneratedWalletAccount1 := &accsmanagementtypes.Account{
		Address:   derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      accsmanagementtypes.AccountTypeSeed,
		Path:      accsmanagementcommon.PathDefaultWalletAccount,
		PublicKey: types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].PublicKeyHex()),
		Name:      "Seed Impo 1 Acc 1",
		Emoji:     "emoji-1",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  accsmanagementtypes.AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount1)
	keypair.LastUsedDerivationIndex = 0

	seedGeneratedWalletAccount2 := &accsmanagementtypes.Account{
		Address:   derivedAccs[accsmanagementcommon.CustomWalletPath1].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      accsmanagementtypes.AccountTypeSeed,
		Path:      accsmanagementcommon.CustomWalletPath1,
		PublicKey: types.Hex2Bytes(derivedAccs[accsmanagementcommon.CustomWalletPath1].PublicKeyHex()),
		Name:      "Seed Impo 1 Acc 2",
		Emoji:     "emoji-2",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  accsmanagementtypes.AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount2)
	keypair.LastUsedDerivationIndex = 1

	return
}

func GetSeedImportedKeypair2ForTest() (keypair *accsmanagementtypes.Keypair, acc *generator.Account, derivedAccs map[string]*generator.Account, err error) {
	acc, derivedAccs, err = generator.CreateAndDeriveAccountsFromMnemonic(testSeed2Mnemonic, []string{
		accsmanagementcommon.PathDefaultWalletAccount,
		accsmanagementcommon.CustomWalletPath1,
	}, "")
	if err != nil {
		return
	}

	keypair = &accsmanagementtypes.Keypair{
		KeyUID:      acc.KeyUID(),
		Name:        "Seed Imported 2",
		Type:        accsmanagementtypes.KeypairTypeSeed,
		DerivedFrom: acc.Address().Hex(),
	}

	seedGeneratedWalletAccount1 := &accsmanagementtypes.Account{
		Address:   derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      accsmanagementtypes.AccountTypeSeed,
		Path:      accsmanagementcommon.PathDefaultWalletAccount,
		PublicKey: types.Hex2Bytes(derivedAccs[accsmanagementcommon.PathDefaultWalletAccount].PublicKeyHex()),
		Name:      "Seed Impo 2 Acc 1",
		Emoji:     "emoji-1",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  accsmanagementtypes.AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount1)
	keypair.LastUsedDerivationIndex = 0

	seedGeneratedWalletAccount2 := &accsmanagementtypes.Account{
		Address:   derivedAccs[accsmanagementcommon.CustomWalletPath1].Address(),
		KeyUID:    keypair.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      accsmanagementtypes.AccountTypeSeed,
		Path:      "m/44'/60'/0'/0/1",
		PublicKey: types.Hex2Bytes("0x000000032"),
		Name:      "Seed Impo 2 Acc 2",
		Emoji:     "emoji-2",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  accsmanagementtypes.AccountFullyOperable,
	}
	keypair.Accounts = append(keypair.Accounts, seedGeneratedWalletAccount2)
	keypair.LastUsedDerivationIndex = 1

	return
}

func GetPrivKeyImportedKeypairForTest() *accsmanagementtypes.Keypair {
	kp := &accsmanagementtypes.Keypair{
		KeyUID:      "0000000000000000000000000000000000000000000000000000000000000004",
		Name:        "Priv Key Imported",
		Type:        accsmanagementtypes.KeypairTypeKey,
		DerivedFrom: "", // no derived from for private key imported kp
	}

	privKeyWalletAccount := &accsmanagementtypes.Account{
		Address:   types.Address{0x41},
		KeyUID:    kp.KeyUID,
		Wallet:    false,
		Chat:      false,
		Type:      accsmanagementtypes.AccountTypeKey,
		Path:      "m",
		PublicKey: types.Hex2Bytes("0x000000041"),
		Name:      "Priv Key Impo Acc",
		Emoji:     "emoji-1",
		ColorID:   common.CustomizationColorPrimary,
		Hidden:    false,
		Clock:     0,
		Removed:   false,
		Operable:  accsmanagementtypes.AccountFullyOperable,
	}
	kp.Accounts = append(kp.Accounts, privKeyWalletAccount)

	return kp
}

func Contains[T comparable](container []T, element T, isEqual func(T, T) bool) bool {
	for _, e := range container {
		if isEqual(e, element) {
			return true
		}
	}
	return false
}

func HaveSameElements[T comparable](a []T, b []T, isEqual func(T, T) bool) bool {
	for _, v := range a {
		if !Contains(b, v, isEqual) {
			return false
		}
	}
	return true
}

func SameAccounts(expected, real *accsmanagementtypes.Account) bool {
	return expected.Address == real.Address &&
		expected.KeyUID == real.KeyUID &&
		expected.Wallet == real.Wallet &&
		expected.Chat == real.Chat &&
		expected.Type == real.Type &&
		expected.Path == real.Path &&
		string(expected.PublicKey) == string(real.PublicKey) &&
		expected.Name == real.Name &&
		expected.Emoji == real.Emoji &&
		expected.ColorID == real.ColorID &&
		expected.Hidden == real.Hidden &&
		expected.Clock == real.Clock &&
		expected.Removed == real.Removed &&
		expected.ProdPreferredChainIDs == real.ProdPreferredChainIDs &&
		expected.TestPreferredChainIDs == real.TestPreferredChainIDs
}

func SameAccountsIncludingPosition(expected, real *accsmanagementtypes.Account) bool {
	return SameAccounts(expected, real) && expected.Position == real.Position
}

func SameAccountsWithDifferentOperable(expected, real *accsmanagementtypes.Account, expectedOperableValue accsmanagementtypes.AccountOperable) bool {
	return SameAccounts(expected, real) && real.Operable == expectedOperableValue
}

func SameKeypairs(expected, real *accsmanagementtypes.Keypair) bool {
	same := expected.KeyUID == real.KeyUID &&
		expected.Name == real.Name &&
		expected.Type == real.Type &&
		expected.DerivedFrom == real.DerivedFrom &&
		expected.LastUsedDerivationIndex == real.LastUsedDerivationIndex &&
		expected.Clock == real.Clock &&
		expected.XPub == real.XPub &&
		expected.ColdWallet == real.ColdWallet &&
		len(expected.Accounts) == len(real.Accounts)

	if same {
		for i := range expected.Accounts {
			found := false
			for j := range real.Accounts {
				if SameAccounts(expected.Accounts[i], real.Accounts[j]) {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}
	}

	return same
}

func SameKeypairsWithDifferentSyncedFrom(expected, real *accsmanagementtypes.Keypair, ignoreSyncedFrom bool, expectedSyncedFromValue string,
	expectedOperableValue accsmanagementtypes.AccountOperable) bool {
	same := expected.KeyUID == real.KeyUID &&
		expected.Name == real.Name &&
		expected.Type == real.Type &&
		expected.DerivedFrom == real.DerivedFrom &&
		expected.LastUsedDerivationIndex == real.LastUsedDerivationIndex &&
		expected.Clock == real.Clock &&
		expected.XPub == real.XPub &&
		expected.ColdWallet == real.ColdWallet &&
		len(expected.Accounts) == len(real.Accounts)

	if same && !ignoreSyncedFrom {
		same = same && real.SyncedFrom == expectedSyncedFromValue
	}

	if same {
		for i := range expected.Accounts {
			found := false
			for j := range real.Accounts {
				if SameAccountsWithDifferentOperable(expected.Accounts[i], real.Accounts[j], expectedOperableValue) {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}
	}

	return same
}
