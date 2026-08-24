# Review comments for the #7067 networks stack

Do not push. Paste onto the matching PR. Each body is ≤40 words.

Inherit Bugbot agents found no production logic bugs. These are the strongest findings; red tests cover the major ones.

---

## PR #7747 — `refactor/7067-networks-wallet-config`

### 1. `pkg/backend/geth_backend.go:855`

Login rebuilds networks from the request-built WalletConfig, not mergo-merged `b.config`. Empty login secrets will not fall back to stored Infura/Pokt/stage. Confirm that is intended.

### 2. `params/config.go:221`

`PoktAPIKey` is persisted on NodeConfig JSON. Additive, but any client that round-trips WalletConfig must tolerate the new field.

### 3. `test/unit-network/common/utils.go:10`

`GetWalletConfigFromEnv` still only sets proxy fields, not Infura/Pokt keys. Alchemy tests will not exercise direct-provider token auth from env.

---

## PR #7748 — `refactor/7067-networks-service`

### 1. `pkg/services/networks/service.go:16` (major)

`APIs()` omits `Public: true`, which wallet sets for the same methods. `UpdateWithDefaults` treats public APIs as exposable; match wallet here.

### 2. `pkg/services/networks/service.go:26` (major)

`Start`/`Stop` dereference `s.manager` with no nil check. `NewService(nil)` panics on Start. Guard or document that the manager is required.

### 3. `pkg/backend/node/status_node_services.go:77`

`networks` is registered before wallet, so `Service.Stop` closes the publisher while wallet watchers still run. Stop networks after wallet, or with the rpc client as before.

---

## PR #7749 — `refactor/7067-networkdefaults`

### 1. `pkg/services/wallet/thirdparty/collectibles/alchemy/nft_proxy_utils.go:7`

Alchemy now imports the networks service package to resolve proxy names. That is the move; keep `params` free of a cycle back into services.

### 2. `pkg/services/networks/default_networks.go:1`

Defaults dissolved into `networks` rather than a subpackage. `BuildDefaultNetworks` reads well at bootstrap; the spec table is no longer reusable without importing the service.

### 3. `pkg/backend/defaults.go` (`BuildDefaultNetworks` call)

Call sites only retargeted the import. No extra test that create/login still pass `WalletConfig` after the package move.

---

## PR #7750 — `refactor/7067-networkhelper`

### 1. `pkg/services/networks/provider_utils.go:54`

Unexporting `GetUserProviders` and deleting `params/networkhelper` is a Go API break. In-repo callers are updated; confirm desktop/mobile do not import that package.

### 2. `pkg/services/networks/provider_utils.go:76`

`OverrideBasicAuth` and `GetEmbeddedProviders` stay exported for out-of-package tests. Consider testutil helpers so the production API stays small.

### 3. `pkg/services/networks/communities_supported.go:10`

`withCommunitiesSupported` is now unexported and only used from `BuildDefaultNetworks`. Fine; leftover export surface is the two helpers above, not this one.

---

## PR #7751 — `refactor/7067-drop-wallet-network-rpc`

### 1. `pkg/services/wallet/api.go` (deleted network methods) (major)

status-app still has `rpc(getFlatEthereumChains, "wallet")`. Do not merge until status-im/status-app#22061 is in, or login hits method-not-found.

### 2. `pkg/services/networks/api.go:40` (major)

Default `APIModules` is still `eth,net,web3,peer,wallet` with no `networks`. If HTTP/WS filters by that list, the new namespace is invisible once wallet methods disappear.

### 3. `pkg/services/networks/api.go:40`

Deprecated `AddEthereumChain` / `GetEthereumChains` were kept for functional tests. Fine; keep the Deprecated comments so app callers do not treat them as the long-term API.
