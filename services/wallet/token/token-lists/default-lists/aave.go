package defaulttokenlists

import (
	"time"

	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
)

var AaveTokenList = fetcher.FetchedTokenList{
	TokenList: fetcher.TokenList{
		ID:        "aave",
		SourceURL: "https://raw.githubusercontent.com/bgd-labs/aave-address-book/main/tokenlist.json",
	},
	Fetched: time.Unix(1741614301, 0),
	JsonData: `{
  "name": "Aave token list",
  "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg",
  "keywords": ["audited", "verified", "aave"],
  "tags": {
    "underlying": {
      "name": "underlyingAsset",
      "description": "Tokens that are used as underlying assets in the Aave protocol"
    },
    "aaveV2": { "name": "Aave V2", "description": "Tokens related to aave v2" },
    "aaveV3": { "name": "Aave V3", "description": "Tokens related to aave v3" },
    "aTokenV2": {
      "name": "aToken V2",
      "description": "Tokens that earn interest on the Aave Protocol V2"
    },
    "aTokenV3": {
      "name": "aToken V3",
      "description": "Tokens that earn interest on the Aave Protocol V3"
    },
    "stataToken": {
      "name": "stata token",
      "description": "Tokens that are wrapped into a 4626 Vault"
    },
    "staticAT": {
      "name": "static a token",
      "description": "Tokens that are wrapped into a 4626 Vault"
    }
  },
  "tokens": [
    {
      "chainId": 1,
      "address": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 1,
      "address": "0xf9Fb4AD91812b704Ba883B11d2B576E890a6730A",
      "name": "Aave AMM Market WETH",
      "decimals": 18,
      "symbol": "aAmmWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
      }
    },
    {
      "chainId": 1,
      "address": "0x6B175474E89094C44Da98b954EedeAC495271d0F",
      "name": "Dai Stablecoin",
      "decimals": 18,
      "symbol": "DAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 1,
      "address": "0x79bE75FFC64DD58e66787E4Eae470c8a1FD08ba4",
      "name": "Aave AMM Market DAI",
      "decimals": 18,
      "symbol": "aAmmDAI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x6B175474E89094C44Da98b954EedeAC495271d0F"
      }
    },
    {
      "chainId": 1,
      "address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 1,
      "address": "0xd24946147829DEaA935bE2aD85A3291dbf109c80",
      "name": "Aave AMM Market USDC",
      "decimals": 6,
      "symbol": "aAmmUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
      }
    },
    {
      "chainId": 1,
      "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
      "name": "Tether USD",
      "decimals": 6,
      "symbol": "USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 1,
      "address": "0x17a79792Fe6fE5C95dFE95Fe3fCEE3CAf4fE4Cb7",
      "name": "Aave AMM Market USDT",
      "decimals": 6,
      "symbol": "aAmmUSDT",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xdAC17F958D2ee523a2206206994597C13D831ec7"
      }
    },
    {
      "chainId": 1,
      "address": "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
      "name": "Wrapped BTC",
      "decimals": 8,
      "symbol": "WBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbtc.svg"
    },
    {
      "chainId": 1,
      "address": "0x13B2f6928D7204328b0E8E4BCd0379aA06EA21FA",
      "name": "Aave AMM Market WBTC",
      "decimals": 8,
      "symbol": "aAmmWBTC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"
      }
    },
    {
      "chainId": 1,
      "address": "0xA478c2975Ab1Ea89e8196811F51A7B7Ade33eB11",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_DAI_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x9303EabC860a743aABcc3A1629014CaBcc3F8D36",
      "name": "Aave AMM Market UniDAIWETH",
      "decimals": 18,
      "symbol": "aAmmUniDAIWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xA478c2975Ab1Ea89e8196811F51A7B7Ade33eB11"
      }
    },
    {
      "chainId": 1,
      "address": "0xBb2b8038a1640196FbE3e38816F3e67Cba72D940",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_WBTC_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0xc58F53A8adff2fB4eb16ED56635772075E2EE123",
      "name": "Aave AMM Market UniWBTCWETH",
      "decimals": 18,
      "symbol": "aAmmUniWBTCWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xBb2b8038a1640196FbE3e38816F3e67Cba72D940"
      }
    },
    {
      "chainId": 1,
      "address": "0xDFC14d2Af169B0D36C4EFF567Ada9b2E0CAE044f",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_AAVE_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0xe59d2FF6995a926A574390824a657eEd36801E55",
      "name": "Aave AMM Market UniAAVEWETH",
      "decimals": 18,
      "symbol": "aAmmUniAAVEWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xDFC14d2Af169B0D36C4EFF567Ada9b2E0CAE044f"
      }
    },
    {
      "chainId": 1,
      "address": "0xB6909B960DbbE7392D405429eB2b3649752b4838",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_BAT_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0xA1B0edF4460CC4d8bFAA18Ed871bFF15E5b57Eb4",
      "name": "Aave AMM Market UniBATWETH",
      "decimals": 18,
      "symbol": "aAmmUniBATWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xB6909B960DbbE7392D405429eB2b3649752b4838"
      }
    },
    {
      "chainId": 1,
      "address": "0xAE461cA67B15dc8dc81CE7615e0320dA1A9aB8D5",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_DAI_USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0xE340B25fE32B1011616bb8EC495A4d503e322177",
      "name": "Aave AMM Market UniDAIUSDC",
      "decimals": 18,
      "symbol": "aAmmUniDAIUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xAE461cA67B15dc8dc81CE7615e0320dA1A9aB8D5"
      }
    },
    {
      "chainId": 1,
      "address": "0x3dA1313aE46132A397D90d95B1424A9A7e3e0fCE",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_CRV_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x0ea20e7fFB006d4Cfe84df2F72d8c7bD89247DB0",
      "name": "Aave AMM Market UniCRVWETH",
      "decimals": 18,
      "symbol": "aAmmUniCRVWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x3dA1313aE46132A397D90d95B1424A9A7e3e0fCE"
      }
    },
    {
      "chainId": 1,
      "address": "0xa2107FA5B38d9bbd2C461D6EDf11B11A50F6b974",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_LINK_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0xb8db81B84d30E2387de0FF330420A4AAA6688134",
      "name": "Aave AMM Market UniLINKWETH",
      "decimals": 18,
      "symbol": "aAmmUniLINKWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xa2107FA5B38d9bbd2C461D6EDf11B11A50F6b974"
      }
    },
    {
      "chainId": 1,
      "address": "0xC2aDdA861F89bBB333c90c492cB837741916A225",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_MKR_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x370adc71f67f581158Dc56f539dF5F399128Ddf9",
      "name": "Aave AMM Market UniMKRWETH",
      "decimals": 18,
      "symbol": "aAmmUniMKRWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xC2aDdA861F89bBB333c90c492cB837741916A225"
      }
    },
    {
      "chainId": 1,
      "address": "0x8Bd1661Da98EBDd3BD080F0bE4e6d9bE8cE9858c",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_REN_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0xA9e201A4e269d6cd5E9F0FcbcB78520cf815878B",
      "name": "Aave AMM Market UniRENWETH",
      "decimals": 18,
      "symbol": "aAmmUniRENWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x8Bd1661Da98EBDd3BD080F0bE4e6d9bE8cE9858c"
      }
    },
    {
      "chainId": 1,
      "address": "0x43AE24960e5534731Fc831386c07755A2dc33D47",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_SNX_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x38E491A71291CD43E8DE63b7253E482622184894",
      "name": "Aave AMM Market UniSNXWETH",
      "decimals": 18,
      "symbol": "aAmmUniSNXWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x43AE24960e5534731Fc831386c07755A2dc33D47"
      }
    },
    {
      "chainId": 1,
      "address": "0xd3d2E2692501A5c9Ca623199D38826e513033a17",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_UNI_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x3D26dcd840fCC8e4B2193AcE8A092e4a65832F9f",
      "name": "Aave AMM Market UniUNIWETH",
      "decimals": 18,
      "symbol": "aAmmUniUNIWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xd3d2E2692501A5c9Ca623199D38826e513033a17"
      }
    },
    {
      "chainId": 1,
      "address": "0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_USDC_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x391E86e2C002C70dEe155eAceB88F7A3c38f5976",
      "name": "Aave AMM Market UniUSDCWETH",
      "decimals": 18,
      "symbol": "aAmmUniUSDCWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc"
      }
    },
    {
      "chainId": 1,
      "address": "0x004375Dff511095CC5A197A54140a24eFEF3A416",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_WBTC_USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x2365a4890eD8965E564B7E2D27C38Ba67Fec4C6F",
      "name": "Aave AMM Market UniWBTCUSDC",
      "decimals": 18,
      "symbol": "aAmmUniWBTCUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x004375Dff511095CC5A197A54140a24eFEF3A416"
      }
    },
    {
      "chainId": 1,
      "address": "0x2fDbAdf3C4D5A8666Bc06645B8358ab803996E28",
      "name": "Uniswap V2",
      "decimals": 18,
      "symbol": "UNI_YFI_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0x5394794Be8b6eD5572FCd6b27103F46b5F390E8f",
      "name": "Aave AMM Market UniYFIWETH",
      "decimals": 18,
      "symbol": "aAmmUniYFIWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x2fDbAdf3C4D5A8666Bc06645B8358ab803996E28"
      }
    },
    {
      "chainId": 1,
      "address": "0x1efF8aF5D577060BA4ac8A29A13525bb0Ee2A3D5",
      "name": "Balancer Pool Token",
      "decimals": 18,
      "symbol": "BPT_WBTC_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/bpt.svg"
    },
    {
      "chainId": 1,
      "address": "0x358bD0d980E031E23ebA9AA793926857703783BD",
      "name": "Aave AMM Market BptWBTCWETH",
      "decimals": 18,
      "symbol": "aAmmBptWBTCWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abpt.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x1efF8aF5D577060BA4ac8A29A13525bb0Ee2A3D5"
      }
    },
    {
      "chainId": 1,
      "address": "0x59A19D8c652FA0284f44113D0ff9aBa70bd46fB4",
      "name": "Balancer Pool Token",
      "decimals": 18,
      "symbol": "BPT_BAL_WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/bpt.svg"
    },
    {
      "chainId": 1,
      "address": "0xd109b2A304587569c84308c55465cd9fF0317bFB",
      "name": "Aave AMM Market BptBALWETH",
      "decimals": 18,
      "symbol": "aAmmBptBALWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abpt.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x59A19D8c652FA0284f44113D0ff9aBa70bd46fB4"
      }
    },
    {
      "chainId": 1,
      "address": "0x50379f632ca68D36E50cfBC8F78fe16bd1499d1e",
      "name": "Gelato Uniswap DAI/USDC LP",
      "decimals": 18,
      "symbol": "GUNI_DAI_USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/guni.svg"
    },
    {
      "chainId": 1,
      "address": "0xd145c6ae8931ed5Bca9b5f5B7dA5991F5aB63B5c",
      "name": "Aave AMM Market GUniDAIUSDC",
      "decimals": 18,
      "symbol": "aAmmGUniDAIUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aguni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0x50379f632ca68D36E50cfBC8F78fe16bd1499d1e"
      }
    },
    {
      "chainId": 1,
      "address": "0xD2eeC91055F07fE24C9cCB25828ecfEFd4be0c41",
      "name": "Gelato Uniswap USDC/USDT LP",
      "decimals": 18,
      "symbol": "GUNI_USDC_USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/guni.svg"
    },
    {
      "chainId": 1,
      "address": "0xCa5DFDABBfFD58cfD49A9f78Ca52eC8e0591a3C5",
      "name": "Aave AMM Market GUniUSDCUSDT",
      "decimals": 18,
      "symbol": "aAmmGUniUSDCUSDT",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aguni.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xD2eeC91055F07fE24C9cCB25828ecfEFd4be0c41"
      }
    },
    {
      "chainId": 1,
      "address": "0xd35f648C3C7f17cd1Ba92e5eac991E3EfcD4566d",
      "name": "Aave Arc market USDC",
      "decimals": 6,
      "symbol": "aUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x37D7306019a38Af123e4b245Eb6C28AF552e0bB0",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
      }
    },
    {
      "chainId": 1,
      "address": "0xe6d6E7dA65A2C18109Ff56B7CBBdc7B706Fc13F8",
      "name": "Aave Arc market WBTC",
      "decimals": 8,
      "symbol": "aWBTC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x37D7306019a38Af123e4b245Eb6C28AF552e0bB0",
        "underlying": "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"
      }
    },
    {
      "chainId": 1,
      "address": "0x319190E3Bbc595602A9E63B2bCfB61c6634355b1",
      "name": "Aave Arc market WETH",
      "decimals": 18,
      "symbol": "aWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x37D7306019a38Af123e4b245Eb6C28AF552e0bB0",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
      }
    },
    {
      "chainId": 1,
      "address": "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9",
      "name": "Aave Token",
      "decimals": 18,
      "symbol": "AAVE",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg"
    },
    {
      "chainId": 1,
      "address": "0x89eFaC495C65d43619c661df654ec64fc10C0A75",
      "name": "Aave Arc market AAVE",
      "decimals": 18,
      "symbol": "aAAVE",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x37D7306019a38Af123e4b245Eb6C28AF552e0bB0",
        "underlying": "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9"
      }
    },
    {
      "chainId": 1,
      "address": "0x3Ed3B47Dd13EC9a98b44e6204A523E766B225811",
      "name": "Aave interest bearing USDT",
      "decimals": 6,
      "symbol": "aUSDT",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xdAC17F958D2ee523a2206206994597C13D831ec7"
      }
    },
    {
      "chainId": 1,
      "address": "0x9ff58f4fFB29fA2266Ab25e75e2A8b3503311656",
      "name": "Aave interest bearing WBTC",
      "decimals": 8,
      "symbol": "aWBTC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"
      }
    },
    {
      "chainId": 1,
      "address": "0x030bA81f1c18d280636F32af80b9AAd02Cf0854e",
      "name": "Aave interest bearing WETH",
      "decimals": 18,
      "symbol": "aWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
      }
    },
    {
      "chainId": 1,
      "address": "0x0bc529c00C6401aEF6D220BE8C6Ea1667F6Ad93e",
      "name": "yearn.finance",
      "decimals": 18,
      "symbol": "YFI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/yfi.svg"
    },
    {
      "chainId": 1,
      "address": "0x5165d24277cD063F5ac44Efd447B27025e888f37",
      "name": "Aave interest bearing YFI",
      "decimals": 18,
      "symbol": "aYFI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ayfi.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x0bc529c00C6401aEF6D220BE8C6Ea1667F6Ad93e"
      }
    },
    {
      "chainId": 1,
      "address": "0xE41d2489571d322189246DaFA5ebDe1F4699F498",
      "name": "0x Protocol Token",
      "decimals": 18,
      "symbol": "ZRX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/zrx.svg"
    },
    {
      "chainId": 1,
      "address": "0xDf7FF54aAcAcbFf42dfe29DD6144A69b629f8C9e",
      "name": "Aave interest bearing ZRX",
      "decimals": 18,
      "symbol": "aZRX",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/azrx.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xE41d2489571d322189246DaFA5ebDe1F4699F498"
      }
    },
    {
      "chainId": 1,
      "address": "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984",
      "name": "Uniswap",
      "decimals": 18,
      "symbol": "UNI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/uni.svg"
    },
    {
      "chainId": 1,
      "address": "0xB9D7CB55f463405CDfBe4E90a6D2Df01C2B92BF1",
      "name": "Aave interest bearing UNI",
      "decimals": 18,
      "symbol": "aUNI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984"
      }
    },
    {
      "chainId": 1,
      "address": "0xFFC97d72E13E01096502Cb8Eb52dEe56f74DAD7B",
      "name": "Aave interest bearing AAVE",
      "decimals": 18,
      "symbol": "aAAVE",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9"
      }
    },
    {
      "chainId": 1,
      "address": "0x0D8775F648430679A709E98d2b0Cb6250d2887EF",
      "name": "Basic Attention Token",
      "decimals": 18,
      "symbol": "BAT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/bat.svg"
    },
    {
      "chainId": 1,
      "address": "0x05Ec93c0365baAeAbF7AefFb0972ea7ECdD39CF1",
      "name": "Aave interest bearing BAT",
      "decimals": 18,
      "symbol": "aBAT",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abat.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x0D8775F648430679A709E98d2b0Cb6250d2887EF"
      }
    },
    {
      "chainId": 1,
      "address": "0x4Fabb145d64652a948d72533023f6E7A623C7C53",
      "name": "BUSD",
      "decimals": 18,
      "symbol": "BUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/busd.svg"
    },
    {
      "chainId": 1,
      "address": "0xA361718326c15715591c299427c62086F69923D9",
      "name": "Aave interest bearing BUSD",
      "decimals": 18,
      "symbol": "aBUSD",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abusd.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x4Fabb145d64652a948d72533023f6E7A623C7C53"
      }
    },
    {
      "chainId": 1,
      "address": "0x028171bCA77440897B824Ca71D1c56caC55b68A3",
      "name": "Aave interest bearing DAI",
      "decimals": 18,
      "symbol": "aDAI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x6B175474E89094C44Da98b954EedeAC495271d0F"
      }
    },
    {
      "chainId": 1,
      "address": "0xF629cBd94d3791C9250152BD8dfBDF380E2a3B9c",
      "name": "Enjin Coin",
      "decimals": 18,
      "symbol": "ENJ",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/enj.svg"
    },
    {
      "chainId": 1,
      "address": "0xaC6Df26a590F08dcC95D5a4705ae8abbc88509Ef",
      "name": "Aave interest bearing ENJ",
      "decimals": 18,
      "symbol": "aENJ",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aenj.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xF629cBd94d3791C9250152BD8dfBDF380E2a3B9c"
      }
    },
    {
      "chainId": 1,
      "address": "0xdd974D5C2e2928deA5F71b9825b8b646686BD200",
      "name": "Kyber Network Crystal",
      "decimals": 18,
      "symbol": "KNC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/knc.svg"
    },
    {
      "chainId": 1,
      "address": "0x39C6b3e42d6A679d7D776778Fe880BC9487C2EDA",
      "name": "Aave interest bearing KNC",
      "decimals": 18,
      "symbol": "aKNC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aknc.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xdd974D5C2e2928deA5F71b9825b8b646686BD200"
      }
    },
    {
      "chainId": 1,
      "address": "0x514910771AF9Ca656af840dff83E8264EcF986CA",
      "name": "ChainLink Token",
      "decimals": 18,
      "symbol": "LINK",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/link.svg"
    },
    {
      "chainId": 1,
      "address": "0xa06bC25B5805d5F8d82847D191Cb4Af5A3e873E0",
      "name": "Aave interest bearing LINK",
      "decimals": 18,
      "symbol": "aLINK",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x514910771AF9Ca656af840dff83E8264EcF986CA"
      }
    },
    {
      "chainId": 1,
      "address": "0x0F5D2fB29fb7d3CFeE444a200298f468908cC942",
      "name": "Decentraland MANA",
      "decimals": 18,
      "symbol": "MANA",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/mana.svg"
    },
    {
      "chainId": 1,
      "address": "0xa685a61171bb30d4072B338c80Cb7b2c865c873E",
      "name": "Aave interest bearing MANA",
      "decimals": 18,
      "symbol": "aMANA",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amana.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x0F5D2fB29fb7d3CFeE444a200298f468908cC942"
      }
    },
    {
      "chainId": 1,
      "address": "0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2",
      "name": "Maker",
      "decimals": 18,
      "symbol": "MKR",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/mkr.svg"
    },
    {
      "chainId": 1,
      "address": "0xc713e5E149D5D0715DcD1c156a020976e7E56B88",
      "name": "Aave interest bearing MKR",
      "decimals": 18,
      "symbol": "aMKR",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amkr.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2"
      }
    },
    {
      "chainId": 1,
      "address": "0x408e41876cCCDC0F92210600ef50372656052a38",
      "name": "Republic Token",
      "decimals": 18,
      "symbol": "REN",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ren.svg"
    },
    {
      "chainId": 1,
      "address": "0xCC12AbE4ff81c9378D670De1b57F8e0Dd228D77a",
      "name": "Aave interest bearing REN",
      "decimals": 18,
      "symbol": "aREN",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aren.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x408e41876cCCDC0F92210600ef50372656052a38"
      }
    },
    {
      "chainId": 1,
      "address": "0xC011a73ee8576Fb46F5E1c5751cA3B9Fe0af2a6F",
      "name": "Synthetix Network Token",
      "decimals": 18,
      "symbol": "SNX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/snx.svg"
    },
    {
      "chainId": 1,
      "address": "0x35f6B052C598d933D69A4EEC4D04c73A191fE6c2",
      "name": "Aave interest bearing SNX",
      "decimals": 18,
      "symbol": "aSNX",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asnx.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xC011a73ee8576Fb46F5E1c5751cA3B9Fe0af2a6F"
      }
    },
    {
      "chainId": 1,
      "address": "0x57Ab1ec28D129707052df4dF418D58a2D46d5f51",
      "name": "Synth sUSD",
      "decimals": 18,
      "symbol": "sUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/susd.svg"
    },
    {
      "chainId": 1,
      "address": "0x6C5024Cd4F8A59110119C56f8933403A539555EB",
      "name": "Aave interest bearing SUSD",
      "decimals": 18,
      "symbol": "aSUSD",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asusd.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x57Ab1ec28D129707052df4dF418D58a2D46d5f51"
      }
    },
    {
      "chainId": 1,
      "address": "0x0000000000085d4780B73119b644AE5ecd22b376",
      "name": "TrueUSD",
      "decimals": 18,
      "symbol": "TUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/tusd.svg"
    },
    {
      "chainId": 1,
      "address": "0x101cc05f4A51C0319f570d5E146a8C625198e636",
      "name": "Aave interest bearing TUSD",
      "decimals": 18,
      "symbol": "aTUSD",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/atusd.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x0000000000085d4780B73119b644AE5ecd22b376"
      }
    },
    {
      "chainId": 1,
      "address": "0xBcca60bB61934080951369a648Fb03DF4F96263C",
      "name": "Aave interest bearing USDC",
      "decimals": 6,
      "symbol": "aUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
      }
    },
    {
      "chainId": 1,
      "address": "0xD533a949740bb3306d119CC777fa900bA034cd52",
      "name": "Curve DAO Token",
      "decimals": 18,
      "symbol": "CRV",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/crv.svg"
    },
    {
      "chainId": 1,
      "address": "0x8dAE6Cb04688C62d939ed9B68d32Bc62e49970b1",
      "name": "Aave interest bearing CRV",
      "decimals": 18,
      "symbol": "aCRV",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acrv.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xD533a949740bb3306d119CC777fa900bA034cd52"
      }
    },
    {
      "chainId": 1,
      "address": "0x056Fd409E1d7A124BD7017459dFEa2F387b6d5Cd",
      "name": "Gemini dollar",
      "decimals": 2,
      "symbol": "GUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/gusd.svg"
    },
    {
      "chainId": 1,
      "address": "0xD37EE7e4f452C6638c96536e68090De8cBcdb583",
      "name": "Aave interest bearing GUSD",
      "decimals": 2,
      "symbol": "aGUSD",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/agusd.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x056Fd409E1d7A124BD7017459dFEa2F387b6d5Cd"
      }
    },
    {
      "chainId": 1,
      "address": "0xba100000625a3754423978a60c9317c58a424e3D",
      "name": "Balancer",
      "decimals": 18,
      "symbol": "BAL",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/bal.svg"
    },
    {
      "chainId": 1,
      "address": "0x272F97b7a56a387aE942350bBC7Df5700f8a4576",
      "name": "Aave interest bearing BAL",
      "decimals": 18,
      "symbol": "aBAL",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abal.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xba100000625a3754423978a60c9317c58a424e3D"
      }
    },
    {
      "chainId": 1,
      "address": "0x8798249c2E607446EfB7Ad49eC89dD1865Ff4272",
      "name": "SushiBar",
      "decimals": 18,
      "symbol": "xSUSHI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/xsushi.svg"
    },
    {
      "chainId": 1,
      "address": "0xF256CC7847E919FAc9B808cC216cAc87CCF2f47a",
      "name": "Aave interest bearing XSUSHI",
      "decimals": 18,
      "symbol": "aXSUSHI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/axsushi.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x8798249c2E607446EfB7Ad49eC89dD1865Ff4272"
      }
    },
    {
      "chainId": 1,
      "address": "0xD5147bc8e386d91Cc5DBE72099DAC6C9b99276F5",
      "name": "renFIL",
      "decimals": 18,
      "symbol": "renFIL",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/renfil.svg"
    },
    {
      "chainId": 1,
      "address": "0x514cd6756CCBe28772d4Cb81bC3156BA9d1744aa",
      "name": "Aave interest bearing RENFIL",
      "decimals": 18,
      "symbol": "aRENFIL",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/arenfil.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xD5147bc8e386d91Cc5DBE72099DAC6C9b99276F5"
      }
    },
    {
      "chainId": 1,
      "address": "0x03ab458634910AaD20eF5f1C8ee96F1D6ac54919",
      "name": "Rai Reflex Index",
      "decimals": 18,
      "symbol": "RAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/rai.svg"
    },
    {
      "chainId": 1,
      "address": "0xc9BC48c72154ef3e5425641a3c747242112a46AF",
      "name": "Aave interest bearing RAI",
      "decimals": 18,
      "symbol": "aRAI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/arai.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x03ab458634910AaD20eF5f1C8ee96F1D6ac54919"
      }
    },
    {
      "chainId": 1,
      "address": "0xD46bA6D942050d489DBd938a2C909A5d5039A161",
      "name": "Ampleforth",
      "decimals": 9,
      "symbol": "AMPL",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ampl.svg"
    },
    {
      "chainId": 1,
      "address": "0x1E6bb68Acec8fefBD87D192bE09bb274170a0548",
      "name": "Aave interest bearing AMPL",
      "decimals": 9,
      "symbol": "aAMPL",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aampl.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xD46bA6D942050d489DBd938a2C909A5d5039A161"
      }
    },
    {
      "chainId": 1,
      "address": "0x8E870D67F660D95d5be530380D0eC0bd388289E1",
      "name": "Pax Dollar",
      "decimals": 18,
      "symbol": "USDP",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdp.svg"
    },
    {
      "chainId": 1,
      "address": "0x2e8F4bdbE3d47d7d7DE490437AeA9915D930F1A3",
      "name": "Aave interest bearing USDP",
      "decimals": 18,
      "symbol": "aUSDP",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdp.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x8E870D67F660D95d5be530380D0eC0bd388289E1"
      }
    },
    {
      "chainId": 1,
      "address": "0x1494CA1F11D487c2bBe4543E90080AeBa4BA3C2b",
      "name": "DefiPulse Index",
      "decimals": 18,
      "symbol": "DPI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dpi.svg"
    },
    {
      "chainId": 1,
      "address": "0x6F634c6135D2EBD550000ac92F494F9CB8183dAe",
      "name": "Aave interest bearing DPI",
      "decimals": 18,
      "symbol": "aDPI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adpi.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x1494CA1F11D487c2bBe4543E90080AeBa4BA3C2b"
      }
    },
    {
      "chainId": 1,
      "address": "0x853d955aCEf822Db058eb8505911ED77F175b99e",
      "name": "Frax",
      "decimals": 18,
      "symbol": "FRAX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/frax.svg"
    },
    {
      "chainId": 1,
      "address": "0xd4937682df3C8aEF4FE912A96A74121C0829E664",
      "name": "Aave interest bearing FRAX",
      "decimals": 18,
      "symbol": "aFRAX",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afrax.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x853d955aCEf822Db058eb8505911ED77F175b99e"
      }
    },
    {
      "chainId": 1,
      "address": "0x956F47F50A910163D8BF957Cf5846D573E7f87CA",
      "name": "Fei USD",
      "decimals": 18,
      "symbol": "FEI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/fei.svg"
    },
    {
      "chainId": 1,
      "address": "0x683923dB55Fead99A79Fa01A27EeC3cB19679cC3",
      "name": "Aave interest bearing FEI",
      "decimals": 18,
      "symbol": "aFEI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afei.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x956F47F50A910163D8BF957Cf5846D573E7f87CA"
      }
    },
    {
      "chainId": 1,
      "address": "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84",
      "name": "Liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "stETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/steth.svg"
    },
    {
      "chainId": 1,
      "address": "0x1982b2F5814301d4e9a8b0201555376e62F82428",
      "name": "Aave interest bearing STETH",
      "decimals": 18,
      "symbol": "aSTETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asteth.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"
      }
    },
    {
      "chainId": 1,
      "address": "0xC18360217D8F7Ab5e7c516566761Ea12Ce7F9D72",
      "name": "Ethereum Name Service",
      "decimals": 18,
      "symbol": "ENS",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ens.svg"
    },
    {
      "chainId": 1,
      "address": "0x9a14e23A58edf4EFDcB360f68cd1b95ce2081a2F",
      "name": "Aave interest bearing ENS",
      "decimals": 18,
      "symbol": "aENS",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aens.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xC18360217D8F7Ab5e7c516566761Ea12Ce7F9D72"
      }
    },
    {
      "chainId": 1,
      "address": "0xa693B19d2931d498c5B318dF961919BB4aee87a5",
      "name": "UST",
      "decimals": 6,
      "symbol": "UST",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ust.svg"
    },
    {
      "chainId": 1,
      "address": "0xc2e2152647F4C26028482Efaf64b2Aa28779EFC4",
      "name": "Aave interest bearing UST",
      "decimals": 6,
      "symbol": "aUST",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aust.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0xa693B19d2931d498c5B318dF961919BB4aee87a5"
      }
    },
    {
      "chainId": 1,
      "address": "0x4e3FBD56CD56c3e72c1403e103b45Db9da5B9D2B",
      "name": "Convex Token",
      "decimals": 18,
      "symbol": "CVX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/cvx.svg"
    },
    {
      "chainId": 1,
      "address": "0x952749E07d7157bb9644A894dFAF3Bad5eF6D918",
      "name": "Aave interest bearing CVX",
      "decimals": 18,
      "symbol": "aCVX",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acvx.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x4e3FBD56CD56c3e72c1403e103b45Db9da5B9D2B"
      }
    },
    {
      "chainId": 1,
      "address": "0x111111111117dC0aa78b770fA6A738034120C302",
      "name": "1INCH Token",
      "decimals": 18,
      "symbol": "ONE_INCH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/1inch.svg"
    },
    {
      "chainId": 1,
      "address": "0xB29130CBcC3F791f077eAdE0266168E808E5151e",
      "name": "Aave interest bearing 1INCH",
      "decimals": 18,
      "symbol": "aONE_INCH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/a1inch.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x111111111117dC0aa78b770fA6A738034120C302"
      }
    },
    {
      "chainId": 1,
      "address": "0x5f98805A4E8be255a32880FDeC7F6728C6568bA0",
      "name": "LUSD Stablecoin",
      "decimals": 18,
      "symbol": "LUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/lusd.svg"
    },
    {
      "chainId": 1,
      "address": "0xce1871f791548600cb59efbefFC9c38719142079",
      "name": "Aave interest bearing LUSD",
      "decimals": 18,
      "symbol": "aLUSD",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alusd.svg",
      "extensions": {
        "pool": "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9",
        "underlying": "0x5f98805A4E8be255a32880FDeC7F6728C6568bA0"
      }
    },
    {
      "chainId": 137,
      "address": "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063",
      "name": "(PoS) Dai Stablecoin",
      "decimals": 18,
      "symbol": "DAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 137,
      "address": "0x27F8D03b3a2196956ED754baDc28D73be8830A6e",
      "name": "Aave Matic Market DAI",
      "decimals": 18,
      "symbol": "amDAI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063"
      }
    },
    {
      "chainId": 137,
      "address": "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
      "name": "USD Coin (PoS)",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 137,
      "address": "0x1a13F4Ca1d028320A707D99520AbFefca3998b7F",
      "name": "Aave Matic Market USDC",
      "decimals": 6,
      "symbol": "amUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
      }
    },
    {
      "chainId": 137,
      "address": "0xc2132D05D31c914a87C6611C10748AEb04B58e8F",
      "name": "(PoS) Tether USD",
      "decimals": 6,
      "symbol": "USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 137,
      "address": "0x60D55F02A771d515e077c9C2403a1ef324885CeC",
      "name": "Aave Matic Market USDT",
      "decimals": 6,
      "symbol": "amUSDT",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0xc2132D05D31c914a87C6611C10748AEb04B58e8F"
      }
    },
    {
      "chainId": 137,
      "address": "0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6",
      "name": "(PoS) Wrapped BTC",
      "decimals": 8,
      "symbol": "WBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbtc.svg"
    },
    {
      "chainId": 137,
      "address": "0x5c2ed810328349100A66B82b78a1791B101C9D61",
      "name": "Aave Matic Market WBTC",
      "decimals": 8,
      "symbol": "amWBTC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6"
      }
    },
    {
      "chainId": 137,
      "address": "0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 137,
      "address": "0x28424507fefb6f7f8E9D3860F56504E4e5f5f390",
      "name": "Aave Matic Market WETH",
      "decimals": 18,
      "symbol": "amWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619"
      }
    },
    {
      "chainId": 137,
      "address": "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270",
      "name": "Wrapped Matic",
      "decimals": 18,
      "symbol": "WMATIC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wmatic.svg"
    },
    {
      "chainId": 137,
      "address": "0x8dF3aad3a84da6b69A4DA8aeC3eA40d9091B2Ac4",
      "name": "Aave Matic Market WMATIC",
      "decimals": 18,
      "symbol": "amWMATIC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awmatic.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270"
      }
    },
    {
      "chainId": 137,
      "address": "0xD6DF932A45C0f255f85145f286eA0b292B21C90B",
      "name": "Aave (PoS)",
      "decimals": 18,
      "symbol": "AAVE",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg"
    },
    {
      "chainId": 137,
      "address": "0x1d2a0E5EC8E5bBDCA5CB219e649B565d8e5c3360",
      "name": "Aave Matic Market AAVE",
      "decimals": 18,
      "symbol": "amAAVE",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0xD6DF932A45C0f255f85145f286eA0b292B21C90B"
      }
    },
    {
      "chainId": 137,
      "address": "0x385Eeac5cB85A38A9a07A70c73e0a3271CfB54A7",
      "name": "Aavegotchi GHST Token (PoS)",
      "decimals": 18,
      "symbol": "GHST",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ghst.svg"
    },
    {
      "chainId": 137,
      "address": "0x080b5BF8f360F624628E0fb961F4e67c9e3c7CF1",
      "name": "Aave Matic Market GHST",
      "decimals": 18,
      "symbol": "amGHST",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aghst.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x385Eeac5cB85A38A9a07A70c73e0a3271CfB54A7"
      }
    },
    {
      "chainId": 137,
      "address": "0x9a71012B13CA4d3D0Cdc72A177DF3ef03b0E76A3",
      "name": "Balancer (PoS)",
      "decimals": 18,
      "symbol": "BAL",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/bal.svg"
    },
    {
      "chainId": 137,
      "address": "0xc4195D4060DaEac44058Ed668AA5EfEc50D77ff6",
      "name": "Aave Matic Market BAL",
      "decimals": 18,
      "symbol": "amBAL",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abal.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x9a71012B13CA4d3D0Cdc72A177DF3ef03b0E76A3"
      }
    },
    {
      "chainId": 137,
      "address": "0x85955046DF4668e1DD369D2DE9f3AEB98DD2A369",
      "name": "DefiPulse Index (PoS)",
      "decimals": 18,
      "symbol": "DPI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dpi.svg"
    },
    {
      "chainId": 137,
      "address": "0x81fB82aAcB4aBE262fc57F06fD4c1d2De347D7B1",
      "name": "Aave Matic Market DPI",
      "decimals": 18,
      "symbol": "amDPI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adpi.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x85955046DF4668e1DD369D2DE9f3AEB98DD2A369"
      }
    },
    {
      "chainId": 137,
      "address": "0x172370d5Cd63279eFa6d502DAB29171933a610AF",
      "name": "CRV (PoS)",
      "decimals": 18,
      "symbol": "CRV",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/crv.svg"
    },
    {
      "chainId": 137,
      "address": "0x3Df8f92b7E798820ddcCA2EBEA7BAbda2c90c4aD",
      "name": "Aave Matic Market CRV",
      "decimals": 18,
      "symbol": "amCRV",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acrv.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x172370d5Cd63279eFa6d502DAB29171933a610AF"
      }
    },
    {
      "chainId": 137,
      "address": "0x0b3F868E0BE5597D5DB7fEB59E1CADBb0fdDa50a",
      "name": "SushiToken (PoS)",
      "decimals": 18,
      "symbol": "SUSHI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/sushi.svg"
    },
    {
      "chainId": 137,
      "address": "0x21eC9431B5B55c5339Eb1AE7582763087F98FAc2",
      "name": "Aave Matic Market SUSHI",
      "decimals": 18,
      "symbol": "amSUSHI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asushi.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x0b3F868E0BE5597D5DB7fEB59E1CADBb0fdDa50a"
      }
    },
    {
      "chainId": 137,
      "address": "0x53E0bca35eC356BD5ddDFebbD1Fc0fD03FaBad39",
      "name": "ChainLink Token",
      "decimals": 18,
      "symbol": "LINK",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/link.svg"
    },
    {
      "chainId": 137,
      "address": "0x0Ca2e42e8c21954af73Bc9af1213E4e81D6a669A",
      "name": "Aave Matic Market LINK",
      "decimals": 18,
      "symbol": "amLINK",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x8dFf5E27EA6b7AC08EbFdf9eB090F32ee9a30fcf",
        "underlying": "0x53E0bca35eC356BD5ddDFebbD1Fc0fD03FaBad39"
      }
    },
    {
      "chainId": 43114,
      "address": "0x49D5c2BdFfac6CE2BFdB6640F4F80f226bc10bAB",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETHe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 43114,
      "address": "0x53f7c5869a859F0AeC3D334ee8B4Cf01E3492f21",
      "name": "Aave Avalanche Market WETH",
      "decimals": 18,
      "symbol": "avWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x4F01AeD16D97E3aB5ab2B501154DC9bb0F1A5A2C",
        "underlying": "0x49D5c2BdFfac6CE2BFdB6640F4F80f226bc10bAB"
      }
    },
    {
      "chainId": 43114,
      "address": "0xd586E7F844cEa2F87f50152665BCbc2C279D8d70",
      "name": "Dai Stablecoin",
      "decimals": 18,
      "symbol": "DAIe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 43114,
      "address": "0x47AFa96Cdc9fAb46904A55a6ad4bf6660B53c38a",
      "name": "Aave Avalanche Market DAI",
      "decimals": 18,
      "symbol": "avDAI",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x4F01AeD16D97E3aB5ab2B501154DC9bb0F1A5A2C",
        "underlying": "0xd586E7F844cEa2F87f50152665BCbc2C279D8d70"
      }
    },
    {
      "chainId": 43114,
      "address": "0xc7198437980c041c805A1EDcbA50c1Ce5db95118",
      "name": "Tether USD",
      "decimals": 6,
      "symbol": "USDTe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 43114,
      "address": "0x532E6537FEA298397212F09A61e03311686f548e",
      "name": "Aave Avalanche Market USDT",
      "decimals": 6,
      "symbol": "avUSDT",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x4F01AeD16D97E3aB5ab2B501154DC9bb0F1A5A2C",
        "underlying": "0xc7198437980c041c805A1EDcbA50c1Ce5db95118"
      }
    },
    {
      "chainId": 43114,
      "address": "0xA7D7079b0FEaD91F3e65f86E8915Cb59c1a4C664",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDCe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 43114,
      "address": "0x46A51127C3ce23fb7AB1DE06226147F446e4a857",
      "name": "Aave Avalanche Market USDC",
      "decimals": 6,
      "symbol": "avUSDC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x4F01AeD16D97E3aB5ab2B501154DC9bb0F1A5A2C",
        "underlying": "0xA7D7079b0FEaD91F3e65f86E8915Cb59c1a4C664"
      }
    },
    {
      "chainId": 43114,
      "address": "0x63a72806098Bd3D9520cC43356dD78afe5D386D9",
      "name": "Aave Token",
      "decimals": 18,
      "symbol": "AAVEe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg"
    },
    {
      "chainId": 43114,
      "address": "0xD45B7c061016102f9FA220502908f2c0f1add1D7",
      "name": "Aave Avalanche Market AAVE",
      "decimals": 18,
      "symbol": "avAAVE",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x4F01AeD16D97E3aB5ab2B501154DC9bb0F1A5A2C",
        "underlying": "0x63a72806098Bd3D9520cC43356dD78afe5D386D9"
      }
    },
    {
      "chainId": 43114,
      "address": "0x50b7545627a5162F82A992c33b87aDc75187B218",
      "name": "Wrapped BTC",
      "decimals": 8,
      "symbol": "WBTCe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbtc.svg"
    },
    {
      "chainId": 43114,
      "address": "0x686bEF2417b6Dc32C50a3cBfbCC3bb60E1e9a15D",
      "name": "Aave Avalanche Market WBTC",
      "decimals": 8,
      "symbol": "avWBTC",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x4F01AeD16D97E3aB5ab2B501154DC9bb0F1A5A2C",
        "underlying": "0x50b7545627a5162F82A992c33b87aDc75187B218"
      }
    },
    {
      "chainId": 43114,
      "address": "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
      "name": "Wrapped AVAX",
      "decimals": 18,
      "symbol": "WAVAX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wavax.svg"
    },
    {
      "chainId": 43114,
      "address": "0xDFE521292EcE2A4f44242efBcD66Bc594CA9714B",
      "name": "Aave Avalanche Market WAVAX",
      "decimals": 18,
      "symbol": "avWAVAX",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awavax.svg",
      "extensions": {
        "pool": "0x4F01AeD16D97E3aB5ab2B501154DC9bb0F1A5A2C",
        "underlying": "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"
      }
    },
    {
      "chainId": 1,
      "address": "0x4d5F47FA6A74757f35C14fD3a6Ef8E3C9BC514E8",
      "name": "Aave Ethereum WETH",
      "decimals": 18,
      "symbol": "aEthWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
      }
    },
    {
      "chainId": 1,
      "address": "0x252231882FB38481497f3C767469106297c8d93b",
      "name": "Static Aave Ethereum WETH",
      "decimals": 18,
      "symbol": "stataEthWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
        "underlyingAToken": "0x4d5F47FA6A74757f35C14fD3a6Ef8E3C9BC514E8"
      }
    },
    {
      "chainId": 1,
      "address": "0x0bfc9d54Fc184518A81162F8fB99c2eACa081202",
      "name": "Wrapped Aave Ethereum WETH",
      "decimals": 18,
      "symbol": "waEthWETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
        "underlyingAToken": "0x4d5F47FA6A74757f35C14fD3a6Ef8E3C9BC514E8"
      }
    },
    {
      "chainId": 1,
      "address": "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 1,
      "address": "0x0B925eD163218f6662a35e0f0371Ac234f9E9371",
      "name": "Aave Ethereum wstETH",
      "decimals": 18,
      "symbol": "aEthwstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"
      }
    },
    {
      "chainId": 1,
      "address": "0x322AA5F5Be95644d6c36544B6c5061F072D16DF5",
      "name": "Static Aave Ethereum wstETH",
      "decimals": 18,
      "symbol": "stataEthwstETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
        "underlyingAToken": "0x0B925eD163218f6662a35e0f0371Ac234f9E9371"
      }
    },
    {
      "chainId": 1,
      "address": "0x5Ee5bf7ae06D1Be5997A1A72006FE6C607eC6DE8",
      "name": "Aave Ethereum WBTC",
      "decimals": 8,
      "symbol": "aEthWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599"
      }
    },
    {
      "chainId": 1,
      "address": "0xB07E357cc262E92eee03D8B81464D596B258eA7a",
      "name": "Static Aave Ethereum WBTC",
      "decimals": 8,
      "symbol": "stataEthWBTC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawbtc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
        "underlyingAToken": "0x5Ee5bf7ae06D1Be5997A1A72006FE6C607eC6DE8"
      }
    },
    {
      "chainId": 1,
      "address": "0x98C23E9d8f34FEFb1B7BD6a91B7FF122F4e16F5c",
      "name": "Aave Ethereum USDC",
      "decimals": 6,
      "symbol": "aEthUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
      }
    },
    {
      "chainId": 1,
      "address": "0x73edDFa87C71ADdC275c2b9890f5c3a8480bC9E6",
      "name": "Static Aave Ethereum USDC",
      "decimals": 6,
      "symbol": "stataEthUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
        "underlyingAToken": "0x98C23E9d8f34FEFb1B7BD6a91B7FF122F4e16F5c"
      }
    },
    {
      "chainId": 1,
      "address": "0xD4fa2D31b7968E448877f69A96DE69f5de8cD23E",
      "name": "Wrapped Aave Ethereum USDC",
      "decimals": 6,
      "symbol": "waEthUSDC",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
        "underlyingAToken": "0x98C23E9d8f34FEFb1B7BD6a91B7FF122F4e16F5c"
      }
    },
    {
      "chainId": 1,
      "address": "0x018008bfb33d285247A21d44E50697654f754e63",
      "name": "Aave Ethereum DAI",
      "decimals": 18,
      "symbol": "aEthDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x6B175474E89094C44Da98b954EedeAC495271d0F"
      }
    },
    {
      "chainId": 1,
      "address": "0xaf270C38fF895EA3f95Ed488CEACe2386F038249",
      "name": "Static Aave Ethereum DAI",
      "decimals": 18,
      "symbol": "stataEthDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadai.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x6B175474E89094C44Da98b954EedeAC495271d0F",
        "underlyingAToken": "0x018008bfb33d285247A21d44E50697654f754e63"
      }
    },
    {
      "chainId": 1,
      "address": "0x5E8C8A7243651DB1384C0dDfDbE39761E8e7E51a",
      "name": "Aave Ethereum LINK",
      "decimals": 18,
      "symbol": "aEthLINK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x514910771AF9Ca656af840dff83E8264EcF986CA"
      }
    },
    {
      "chainId": 1,
      "address": "0x57bd8C73838d1781b4f6E0d5Cf89eb676488d3df",
      "name": "Static Aave Ethereum LINK",
      "decimals": 18,
      "symbol": "stataEthLINK",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalink.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x514910771AF9Ca656af840dff83E8264EcF986CA",
        "underlyingAToken": "0x5E8C8A7243651DB1384C0dDfDbE39761E8e7E51a"
      }
    },
    {
      "chainId": 1,
      "address": "0xA700b4eB416Be35b2911fd5Dee80678ff64fF6C9",
      "name": "Aave Ethereum AAVE",
      "decimals": 18,
      "symbol": "aEthAAVE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9"
      }
    },
    {
      "chainId": 1,
      "address": "0xFEB859A50f92C6D5ad7C9eF7C2c060D164B3280f",
      "name": "Static Aave Ethereum AAVE",
      "decimals": 18,
      "symbol": "stataEthAAVE",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataaave.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9",
        "underlyingAToken": "0xA700b4eB416Be35b2911fd5Dee80678ff64fF6C9"
      }
    },
    {
      "chainId": 1,
      "address": "0xBe9895146f7AF43049ca1c1AE358B0541Ea49704",
      "name": "Coinbase Wrapped Staked ETH",
      "decimals": 18,
      "symbol": "cbETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/cbeth.svg"
    },
    {
      "chainId": 1,
      "address": "0x977b6fc5dE62598B08C85AC8Cf2b745874E8b78c",
      "name": "Aave Ethereum cbETH",
      "decimals": 18,
      "symbol": "aEthcbETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acbeth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xBe9895146f7AF43049ca1c1AE358B0541Ea49704"
      }
    },
    {
      "chainId": 1,
      "address": "0xe2a6863C8f043457B497667Ef3c43073e2D69089",
      "name": "Static Aave Ethereum cbETH",
      "decimals": 18,
      "symbol": "stataEthcbETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacbeth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xBe9895146f7AF43049ca1c1AE358B0541Ea49704",
        "underlyingAToken": "0x977b6fc5dE62598B08C85AC8Cf2b745874E8b78c"
      }
    },
    {
      "chainId": 1,
      "address": "0x23878914EFE38d27C4D67Ab83ed1b93A74D4086a",
      "name": "Aave Ethereum USDT",
      "decimals": 6,
      "symbol": "aEthUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xdAC17F958D2ee523a2206206994597C13D831ec7"
      }
    },
    {
      "chainId": 1,
      "address": "0x862c57d48becB45583AEbA3f489696D22466Ca1b",
      "name": "Static Aave Ethereum USDT",
      "decimals": 6,
      "symbol": "stataEthUSDT",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdt.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
        "underlyingAToken": "0x23878914EFE38d27C4D67Ab83ed1b93A74D4086a"
      }
    },
    {
      "chainId": 1,
      "address": "0x7Bc3485026Ac48b6cf9BaF0A377477Fff5703Af8",
      "name": "Wrapped Aave Ethereum USDT",
      "decimals": 6,
      "symbol": "waEthUSDT",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdt.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
        "underlyingAToken": "0x23878914EFE38d27C4D67Ab83ed1b93A74D4086a"
      }
    },
    {
      "chainId": 1,
      "address": "0xae78736Cd615f374D3085123A210448E74Fc6393",
      "name": "Rocket Pool ETH",
      "decimals": 18,
      "symbol": "rETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/reth.svg"
    },
    {
      "chainId": 1,
      "address": "0xCc9EE9483f662091a1de4795249E24aC0aC2630f",
      "name": "Aave Ethereum rETH",
      "decimals": 18,
      "symbol": "aEthrETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/areth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xae78736Cd615f374D3085123A210448E74Fc6393"
      }
    },
    {
      "chainId": 1,
      "address": "0x867Cf025B5dA438c4e215c60B59bBB3aFe896Fda",
      "name": "Static Aave Ethereum rETH",
      "decimals": 18,
      "symbol": "stataEthrETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statareth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xae78736Cd615f374D3085123A210448E74Fc6393",
        "underlyingAToken": "0xCc9EE9483f662091a1de4795249E24aC0aC2630f"
      }
    },
    {
      "chainId": 1,
      "address": "0x3Fe6a295459FAe07DF8A0ceCC36F37160FE86AA9",
      "name": "Aave Ethereum LUSD",
      "decimals": 18,
      "symbol": "aEthLUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alusd.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x5f98805A4E8be255a32880FDeC7F6728C6568bA0"
      }
    },
    {
      "chainId": 1,
      "address": "0xDBf5E36569798D1E39eE9d7B1c61A7409a74F23A",
      "name": "Static Aave Ethereum LUSD",
      "decimals": 18,
      "symbol": "stataEthLUSD",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalusd.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x5f98805A4E8be255a32880FDeC7F6728C6568bA0",
        "underlyingAToken": "0x3Fe6a295459FAe07DF8A0ceCC36F37160FE86AA9"
      }
    },
    {
      "chainId": 1,
      "address": "0x7B95Ec873268a6BFC6427e7a28e396Db9D0ebc65",
      "name": "Aave Ethereum CRV",
      "decimals": 18,
      "symbol": "aEthCRV",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acrv.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xD533a949740bb3306d119CC777fa900bA034cd52"
      }
    },
    {
      "chainId": 1,
      "address": "0x149EE12310D499F701B6A5714eDAd2C832008fd2",
      "name": "Static Aave Ethereum CRV",
      "decimals": 18,
      "symbol": "stataEthCRV",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacrv.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xD533a949740bb3306d119CC777fa900bA034cd52",
        "underlyingAToken": "0x7B95Ec873268a6BFC6427e7a28e396Db9D0ebc65"
      }
    },
    {
      "chainId": 1,
      "address": "0x8A458A9dc9048e005d22849F470891b840296619",
      "name": "Aave Ethereum MKR",
      "decimals": 18,
      "symbol": "aEthMKR",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amkr.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2"
      }
    },
    {
      "chainId": 1,
      "address": "0xC7B4c17861357B8ABB91F25581E7263E08DCB59c",
      "name": "Aave Ethereum SNX",
      "decimals": 18,
      "symbol": "aEthSNX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asnx.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xC011a73ee8576Fb46F5E1c5751cA3B9Fe0af2a6F"
      }
    },
    {
      "chainId": 1,
      "address": "0xaECEbdfE454d869A626cAb38226C52a1575D1866",
      "name": "Static Aave Ethereum SNX",
      "decimals": 18,
      "symbol": "stataEthSNX",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statasnx.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xC011a73ee8576Fb46F5E1c5751cA3B9Fe0af2a6F",
        "underlyingAToken": "0xC7B4c17861357B8ABB91F25581E7263E08DCB59c"
      }
    },
    {
      "chainId": 1,
      "address": "0x2516E7B3F76294e03C42AA4c5b5b4DCE9C436fB8",
      "name": "Aave Ethereum BAL",
      "decimals": 18,
      "symbol": "aEthBAL",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abal.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xba100000625a3754423978a60c9317c58a424e3D"
      }
    },
    {
      "chainId": 1,
      "address": "0xF6D2224916DDFbbab6e6bd0D1B7034f4Ae0CaB18",
      "name": "Aave Ethereum UNI",
      "decimals": 18,
      "symbol": "aEthUNI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/auni.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984"
      }
    },
    {
      "chainId": 1,
      "address": "0x78fb5E79D5cb59729D0cd72bEA7879aD2683454D",
      "name": "Static Aave Ethereum UNI",
      "decimals": 18,
      "symbol": "stataEthUNI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statauni.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984",
        "underlyingAToken": "0xF6D2224916DDFbbab6e6bd0D1B7034f4Ae0CaB18"
      }
    },
    {
      "chainId": 1,
      "address": "0x5A98FcBEA516Cf06857215779Fd812CA3beF1B32",
      "name": "Lido DAO Token",
      "decimals": 18,
      "symbol": "LDO",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ldo.svg"
    },
    {
      "chainId": 1,
      "address": "0x9A44fd41566876A39655f74971a3A6eA0a17a454",
      "name": "Aave Ethereum LDO",
      "decimals": 18,
      "symbol": "aEthLDO",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aldo.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x5A98FcBEA516Cf06857215779Fd812CA3beF1B32"
      }
    },
    {
      "chainId": 1,
      "address": "0x1eA6E1ba21601258401d0B9DB24eA0a07948458e",
      "name": "Static Aave Ethereum LDO",
      "decimals": 18,
      "symbol": "stataEthLDO",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataldo.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x5A98FcBEA516Cf06857215779Fd812CA3beF1B32",
        "underlyingAToken": "0x9A44fd41566876A39655f74971a3A6eA0a17a454"
      }
    },
    {
      "chainId": 1,
      "address": "0x545bD6c032eFdde65A377A6719DEF2796C8E0f2e",
      "name": "Aave Ethereum ENS",
      "decimals": 18,
      "symbol": "aEthENS",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aens.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xC18360217D8F7Ab5e7c516566761Ea12Ce7F9D72"
      }
    },
    {
      "chainId": 1,
      "address": "0x2767C27Eeaf3566082E74b963B6A0f5c9a46C8a1",
      "name": "Static Aave Ethereum ENS",
      "decimals": 18,
      "symbol": "stataEthENS",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataens.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xC18360217D8F7Ab5e7c516566761Ea12Ce7F9D72",
        "underlyingAToken": "0x545bD6c032eFdde65A377A6719DEF2796C8E0f2e"
      }
    },
    {
      "chainId": 1,
      "address": "0x71Aef7b30728b9BB371578f36c5A1f1502a5723e",
      "name": "Aave Ethereum 1INCH",
      "decimals": 18,
      "symbol": "aEthONE_INCH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/a1inch.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x111111111117dC0aa78b770fA6A738034120C302"
      }
    },
    {
      "chainId": 1,
      "address": "0xB490fF18e55b8881C9527FE7E358dd363780449F",
      "name": "Static Aave Ethereum 1INCH",
      "decimals": 18,
      "symbol": "stataEthONE_INCH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stata1inch.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x111111111117dC0aa78b770fA6A738034120C302",
        "underlyingAToken": "0x71Aef7b30728b9BB371578f36c5A1f1502a5723e"
      }
    },
    {
      "chainId": 1,
      "address": "0xd4e245848d6E1220DBE62e155d89fa327E43CB06",
      "name": "Aave Ethereum FRAX",
      "decimals": 18,
      "symbol": "aEthFRAX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afrax.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x853d955aCEf822Db058eb8505911ED77F175b99e"
      }
    },
    {
      "chainId": 1,
      "address": "0xEE66abD4D0f9908A48E08AE354B0f425De3e237E",
      "name": "Static Aave Ethereum FRAX",
      "decimals": 18,
      "symbol": "stataEthFRAX",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statafrax.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x853d955aCEf822Db058eb8505911ED77F175b99e",
        "underlyingAToken": "0xd4e245848d6E1220DBE62e155d89fa327E43CB06"
      }
    },
    {
      "chainId": 1,
      "address": "0x40D16FC0246aD3160Ccc09B8D0D3A2cD28aE6C2f",
      "name": "Gho Token",
      "decimals": 18,
      "symbol": "GHO",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/gho.svg"
    },
    {
      "chainId": 1,
      "address": "0x00907f9921424583e7ffBfEdf84F92B7B2Be4977",
      "name": "Aave Ethereum GHO",
      "decimals": 18,
      "symbol": "aEthGHO",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/agho.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x40D16FC0246aD3160Ccc09B8D0D3A2cD28aE6C2f"
      }
    },
    {
      "chainId": 1,
      "address": "0x048459E4fb3402e58d8900aF7283Ad574B91d742",
      "name": "Static Aave Ethereum GHO",
      "decimals": 18,
      "symbol": "stataEthGHO",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statagho.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x40D16FC0246aD3160Ccc09B8D0D3A2cD28aE6C2f",
        "underlyingAToken": "0x00907f9921424583e7ffBfEdf84F92B7B2Be4977"
      }
    },
    {
      "chainId": 1,
      "address": "0xD33526068D116cE69F19A9ee46F0bd304F21A51f",
      "name": "Rocket Pool Protocol",
      "decimals": 18,
      "symbol": "RPL",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/rpl.svg"
    },
    {
      "chainId": 1,
      "address": "0xB76CF92076adBF1D9C39294FA8e7A67579FDe357",
      "name": "Aave Ethereum RPL",
      "decimals": 18,
      "symbol": "aEthRPL",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/arpl.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xD33526068D116cE69F19A9ee46F0bd304F21A51f"
      }
    },
    {
      "chainId": 1,
      "address": "0x95EF7cb3494e65dA4926bA330dBf540a13afFD17",
      "name": "Static Aave Ethereum RPL",
      "decimals": 18,
      "symbol": "stataEthRPL",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statarpl.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xD33526068D116cE69F19A9ee46F0bd304F21A51f",
        "underlyingAToken": "0xB76CF92076adBF1D9C39294FA8e7A67579FDe357"
      }
    },
    {
      "chainId": 1,
      "address": "0x83F20F44975D03b1b09e64809B757c47f942BEeA",
      "name": "Savings Dai",
      "decimals": 18,
      "symbol": "sDAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/sdai.svg"
    },
    {
      "chainId": 1,
      "address": "0x4C612E3B15b96Ff9A6faED838F8d07d479a8dD4c",
      "name": "Aave Ethereum sDAI",
      "decimals": 18,
      "symbol": "aEthsDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asdai.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x83F20F44975D03b1b09e64809B757c47f942BEeA"
      }
    },
    {
      "chainId": 1,
      "address": "0xFa7E3571786CE9489bBC58d9Cb8ecE8aAe6B56F3",
      "name": "Static Aave Ethereum sDAI",
      "decimals": 18,
      "symbol": "stataEthsDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statasdai.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x83F20F44975D03b1b09e64809B757c47f942BEeA",
        "underlyingAToken": "0x4C612E3B15b96Ff9A6faED838F8d07d479a8dD4c"
      }
    },
    {
      "chainId": 1,
      "address": "0xAf5191B0De278C7286d6C7CC6ab6BB8A73bA2Cd6",
      "name": "StargateToken",
      "decimals": 18,
      "symbol": "STG",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stg.svg"
    },
    {
      "chainId": 1,
      "address": "0x1bA9843bD4327c6c77011406dE5fA8749F7E3479",
      "name": "Aave Ethereum STG",
      "decimals": 18,
      "symbol": "aEthSTG",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/astg.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xAf5191B0De278C7286d6C7CC6ab6BB8A73bA2Cd6"
      }
    },
    {
      "chainId": 1,
      "address": "0xdeFA4e8a7bcBA345F687a2f1456F5Edd9CE97202",
      "name": "Kyber Network Crystal v2",
      "decimals": 18,
      "symbol": "KNC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/knc.svg"
    },
    {
      "chainId": 1,
      "address": "0x5b502e3796385E1e9755d7043B9C945C3aCCeC9C",
      "name": "Aave Ethereum KNC",
      "decimals": 18,
      "symbol": "aEthKNC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aknc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xdeFA4e8a7bcBA345F687a2f1456F5Edd9CE97202"
      }
    },
    {
      "chainId": 1,
      "address": "0x3432B6A60D23Ca0dFCa7761B7ab56459D9C964D0",
      "name": "Frax Share",
      "decimals": 18,
      "symbol": "FXS",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/fxs.svg"
    },
    {
      "chainId": 1,
      "address": "0x82F9c5ad306BBa1AD0De49bB5FA6F01bf61085ef",
      "name": "Aave Ethereum FXS",
      "decimals": 18,
      "symbol": "aEthFXS",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afxs.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x3432B6A60D23Ca0dFCa7761B7ab56459D9C964D0"
      }
    },
    {
      "chainId": 1,
      "address": "0xf939E0A03FB07F59A73314E73794Be0E57ac1b4E",
      "name": "Curve.Fi USD Stablecoin",
      "decimals": 18,
      "symbol": "crvUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/crvusd.svg"
    },
    {
      "chainId": 1,
      "address": "0xb82fa9f31612989525992FCfBB09AB22Eff5c85A",
      "name": "Aave Ethereum crvUSD",
      "decimals": 18,
      "symbol": "aEthcrvUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acrvusd.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xf939E0A03FB07F59A73314E73794Be0E57ac1b4E"
      }
    },
    {
      "chainId": 1,
      "address": "0x848107491E029AFDe0AC543779c7790382f15929",
      "name": "Static Aave Ethereum crvUSD",
      "decimals": 18,
      "symbol": "stataEthcrvUSD",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacrvusd.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xf939E0A03FB07F59A73314E73794Be0E57ac1b4E",
        "underlyingAToken": "0xb82fa9f31612989525992FCfBB09AB22Eff5c85A"
      }
    },
    {
      "chainId": 1,
      "address": "0x6c3ea9036406852006290770BEdFcAbA0e23A0e8",
      "name": "PayPal USD",
      "decimals": 6,
      "symbol": "PYUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/pyusd.svg"
    },
    {
      "chainId": 1,
      "address": "0x0C0d01AbF3e6aDfcA0989eBbA9d6e85dD58EaB1E",
      "name": "Aave Ethereum PYUSD",
      "decimals": 6,
      "symbol": "aEthPYUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/apyusd.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x6c3ea9036406852006290770BEdFcAbA0e23A0e8"
      }
    },
    {
      "chainId": 1,
      "address": "0x00F2a835758B33f3aC53516Ebd69f3dc77B0D152",
      "name": "Static Aave Ethereum PYUSD",
      "decimals": 6,
      "symbol": "stataEthPYUSD",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statapyusd.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x6c3ea9036406852006290770BEdFcAbA0e23A0e8",
        "underlyingAToken": "0x0C0d01AbF3e6aDfcA0989eBbA9d6e85dD58EaB1E"
      }
    },
    {
      "chainId": 1,
      "address": "0xb51EDdDD8c47856D81C8681EA71404Cec93E92c6",
      "name": "Wrapped Aave Ethereum PYUSD",
      "decimals": 6,
      "symbol": "waEthPYUSD",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statapyusd.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x6c3ea9036406852006290770BEdFcAbA0e23A0e8",
        "underlyingAToken": "0x0C0d01AbF3e6aDfcA0989eBbA9d6e85dD58EaB1E"
      }
    },
    {
      "chainId": 1,
      "address": "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee",
      "name": "Wrapped eETH",
      "decimals": 18,
      "symbol": "weETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weeth.svg"
    },
    {
      "chainId": 1,
      "address": "0xBdfa7b7893081B35Fb54027489e2Bc7A38275129",
      "name": "Aave Ethereum weETH",
      "decimals": 18,
      "symbol": "aEthweETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweeth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"
      }
    },
    {
      "chainId": 1,
      "address": "0x867b0CDC4B39a19945E616c29639b0390b39db3B",
      "name": "Static Aave Ethereum weETH",
      "decimals": 18,
      "symbol": "stataEthweETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweeth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee",
        "underlyingAToken": "0xBdfa7b7893081B35Fb54027489e2Bc7A38275129"
      }
    },
    {
      "chainId": 1,
      "address": "0xf1C9acDc66974dFB6dEcB12aA385b9cD01190E38",
      "name": "Staked ETH",
      "decimals": 18,
      "symbol": "osETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/oseth.svg"
    },
    {
      "chainId": 1,
      "address": "0x927709711794F3De5DdBF1D176bEE2D55Ba13c21",
      "name": "Aave Ethereum osETH",
      "decimals": 18,
      "symbol": "aEthosETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aoseth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xf1C9acDc66974dFB6dEcB12aA385b9cD01190E38"
      }
    },
    {
      "chainId": 1,
      "address": "0xE5248968166206d14ab57345971E32facD839aDA",
      "name": "Static Aave Ethereum osETH",
      "decimals": 18,
      "symbol": "stataEthosETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataoseth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xf1C9acDc66974dFB6dEcB12aA385b9cD01190E38",
        "underlyingAToken": "0x927709711794F3De5DdBF1D176bEE2D55Ba13c21"
      }
    },
    {
      "chainId": 1,
      "address": "0x4c9EDD5852cd905f086C759E8383e09bff1E68B3",
      "name": "USDe",
      "decimals": 18,
      "symbol": "USDe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usde.svg"
    },
    {
      "chainId": 1,
      "address": "0x4F5923Fc5FD4a93352581b38B7cD26943012DECF",
      "name": "Aave Ethereum USDe",
      "decimals": 18,
      "symbol": "aEthUSDe",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausde.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x4c9EDD5852cd905f086C759E8383e09bff1E68B3"
      }
    },
    {
      "chainId": 1,
      "address": "0x46e5d6A33C8Bd8eD38F3c95991C78C9B2FF3bC99",
      "name": "Static Aave Ethereum USDe",
      "decimals": 18,
      "symbol": "stataEthUSDe",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausde.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x4c9EDD5852cd905f086C759E8383e09bff1E68B3",
        "underlyingAToken": "0x4F5923Fc5FD4a93352581b38B7cD26943012DECF"
      }
    },
    {
      "chainId": 1,
      "address": "0x5F9D59db355b4A60501544637b00e94082cA575b",
      "name": "Wrapped Aave Ethereum USDe",
      "decimals": 18,
      "symbol": "waEthUSDe",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausde.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x4c9EDD5852cd905f086C759E8383e09bff1E68B3",
        "underlyingAToken": "0x4F5923Fc5FD4a93352581b38B7cD26943012DECF"
      }
    },
    {
      "chainId": 1,
      "address": "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b",
      "name": "ETHx",
      "decimals": 18,
      "symbol": "ETHx",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ethx.svg"
    },
    {
      "chainId": 1,
      "address": "0x1c0E06a0b1A4c160c17545FF2A951bfcA57C0002",
      "name": "Aave Ethereum ETHx",
      "decimals": 18,
      "symbol": "aEthETHx",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aethx.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b"
      }
    },
    {
      "chainId": 1,
      "address": "0x7CC6694CF75C18D488d16FB4bf3c71A3B31cc7FB",
      "name": "Static Aave Ethereum ETHx",
      "decimals": 18,
      "symbol": "stataEthETHx",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataethx.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b",
        "underlyingAToken": "0x1c0E06a0b1A4c160c17545FF2A951bfcA57C0002"
      }
    },
    {
      "chainId": 1,
      "address": "0x9D39A5DE30e57443BfF2A8307A4256c8797A3497",
      "name": "Staked USDe",
      "decimals": 18,
      "symbol": "sUSDe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/susde.svg"
    },
    {
      "chainId": 1,
      "address": "0x4579a27aF00A62C0EB156349f31B345c08386419",
      "name": "Aave Ethereum sUSDe",
      "decimals": 18,
      "symbol": "aEthsUSDe",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asusde.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x9D39A5DE30e57443BfF2A8307A4256c8797A3497"
      }
    },
    {
      "chainId": 1,
      "address": "0x54D612b000697bd8B0094889D7d6A92bA0Bf2DEa",
      "name": "Static Aave Ethereum sUSDe",
      "decimals": 18,
      "symbol": "stataEthsUSDe",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statasusde.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x9D39A5DE30e57443BfF2A8307A4256c8797A3497",
        "underlyingAToken": "0x4579a27aF00A62C0EB156349f31B345c08386419"
      }
    },
    {
      "chainId": 1,
      "address": "0x18084fbA666a33d37592fA2633fD49a74DD93a88",
      "name": "tBTC v2",
      "decimals": 18,
      "symbol": "tBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/tbtc.svg"
    },
    {
      "chainId": 1,
      "address": "0x10Ac93971cdb1F5c778144084242374473c350Da",
      "name": "Aave Ethereum tBTC",
      "decimals": 18,
      "symbol": "aEthtBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/atbtc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x18084fbA666a33d37592fA2633fD49a74DD93a88"
      }
    },
    {
      "chainId": 1,
      "address": "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf",
      "name": "Coinbase Wrapped BTC",
      "decimals": 8,
      "symbol": "cbBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/cbbtc.svg"
    },
    {
      "chainId": 1,
      "address": "0x5c647cE0Ae10658ec44FA4E11A51c96e94efd1Dd",
      "name": "Aave Ethereum cbBTC",
      "decimals": 8,
      "symbol": "aEthcbBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acbbtc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"
      }
    },
    {
      "chainId": 1,
      "address": "0xdC035D45d973E3EC169d2276DDab16f1e407384F",
      "name": "USDS Stablecoin",
      "decimals": 18,
      "symbol": "USDS",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usds.svg"
    },
    {
      "chainId": 1,
      "address": "0x32a6268f9Ba3642Dda7892aDd74f1D34469A4259",
      "name": "Aave Ethereum USDS",
      "decimals": 18,
      "symbol": "aEthUSDS",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausds.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xdC035D45d973E3EC169d2276DDab16f1e407384F"
      }
    },
    {
      "chainId": 1,
      "address": "0xb80B3215EA8183a064073f9892eb64236160a4dF",
      "name": "Wrapped Aave Ethereum USDS",
      "decimals": 18,
      "symbol": "waEthUSDS",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausds.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xdC035D45d973E3EC169d2276DDab16f1e407384F",
        "underlyingAToken": "0x32a6268f9Ba3642Dda7892aDd74f1D34469A4259"
      }
    },
    {
      "chainId": 1,
      "address": "0xA1290d69c65A6Fe4DF752f95823fae25cB99e5A7",
      "name": "rsETH",
      "decimals": 18,
      "symbol": "rsETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/rseth.svg"
    },
    {
      "chainId": 1,
      "address": "0x2D62109243b87C4bA3EE7bA1D91B0dD0A074d7b1",
      "name": "Aave Ethereum rsETH",
      "decimals": 18,
      "symbol": "aEthrsETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/arseth.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0xA1290d69c65A6Fe4DF752f95823fae25cB99e5A7"
      }
    },
    {
      "chainId": 1,
      "address": "0x8236a87084f8B84306f72007F36F2618A5634494",
      "name": "Lombard Staked Bitcoin",
      "decimals": 8,
      "symbol": "LBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/lbtc.svg"
    },
    {
      "chainId": 1,
      "address": "0x65906988ADEe75306021C417a1A3458040239602",
      "name": "Aave Ethereum LBTC",
      "decimals": 8,
      "symbol": "aEthLBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/albtc.svg",
      "extensions": {
        "pool": "0x87870Bca3F3fD6335C3F4ce8392D69350B4fA4E2",
        "underlying": "0x8236a87084f8B84306f72007F36F2618A5634494"
      }
    },
    {
      "chainId": 137,
      "address": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE",
      "name": "Aave Polygon DAI",
      "decimals": 18,
      "symbol": "aPolDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063"
      }
    },
    {
      "chainId": 137,
      "address": "0x83c59636e602787A6EEbBdA2915217B416193FcB",
      "name": "Static Aave Polygon DAI",
      "decimals": 18,
      "symbol": "stataPolDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063",
        "underlyingAToken": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE"
      }
    },
    {
      "chainId": 137,
      "address": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530",
      "name": "Aave Polygon LINK",
      "decimals": 18,
      "symbol": "aPolLINK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x53E0bca35eC356BD5ddDFebbD1Fc0fD03FaBad39"
      }
    },
    {
      "chainId": 137,
      "address": "0x37868a45c6741616F9E5a189dC0481AD70056B6a",
      "name": "Static Aave Polygon LINK",
      "decimals": 18,
      "symbol": "stataPolLINK",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x53E0bca35eC356BD5ddDFebbD1Fc0fD03FaBad39",
        "underlyingAToken": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530"
      }
    },
    {
      "chainId": 137,
      "address": "0x625E7708f30cA75bfd92586e17077590C60eb4cD",
      "name": "Aave Polygon USDC",
      "decimals": 6,
      "symbol": "aPolUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
      }
    },
    {
      "chainId": 137,
      "address": "0x1017F4a86Fc3A3c824346d0b8C5e96A5029bDAf9",
      "name": "Static Aave Polygon USDC",
      "decimals": 6,
      "symbol": "stataPolUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
        "underlyingAToken": "0x625E7708f30cA75bfd92586e17077590C60eb4cD"
      }
    },
    {
      "chainId": 137,
      "address": "0x078f358208685046a11C85e8ad32895DED33A249",
      "name": "Aave Polygon WBTC",
      "decimals": 8,
      "symbol": "aPolWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6"
      }
    },
    {
      "chainId": 137,
      "address": "0xbC0f50CCB8514Aa7dFEB297521c4BdEBc9C7d22d",
      "name": "Static Aave Polygon WBTC",
      "decimals": 8,
      "symbol": "stataPolWBTC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x1BFD67037B42Cf73acF2047067bd4F2C47D9BfD6",
        "underlyingAToken": "0x078f358208685046a11C85e8ad32895DED33A249"
      }
    },
    {
      "chainId": 137,
      "address": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8",
      "name": "Aave Polygon WETH",
      "decimals": 18,
      "symbol": "aPolWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619"
      }
    },
    {
      "chainId": 137,
      "address": "0xb3D5Af0A52a35692D3FcbE37669b3B8C31dddE7D",
      "name": "Static Aave Polygon WETH",
      "decimals": 18,
      "symbol": "stataPolWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x7ceB23fD6bC0adD59E62ac25578270cFf1b9f619",
        "underlyingAToken": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8"
      }
    },
    {
      "chainId": 137,
      "address": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620",
      "name": "Aave Polygon USDT",
      "decimals": 6,
      "symbol": "aPolUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xc2132D05D31c914a87C6611C10748AEb04B58e8F"
      }
    },
    {
      "chainId": 137,
      "address": "0x87A1fdc4C726c459f597282be639a045062c0E46",
      "name": "Static Aave Polygon USDT",
      "decimals": 6,
      "symbol": "stataPolUSDT",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdt.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xc2132D05D31c914a87C6611C10748AEb04B58e8F",
        "underlyingAToken": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620"
      }
    },
    {
      "chainId": 137,
      "address": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375",
      "name": "Aave Polygon AAVE",
      "decimals": 18,
      "symbol": "aPolAAVE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xD6DF932A45C0f255f85145f286eA0b292B21C90B"
      }
    },
    {
      "chainId": 137,
      "address": "0xCA2E1E33E5BCF4978E2d683656E1f5610f8C4A7E",
      "name": "Static Aave Polygon AAVE",
      "decimals": 18,
      "symbol": "stataPolAAVE",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xD6DF932A45C0f255f85145f286eA0b292B21C90B",
        "underlyingAToken": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375"
      }
    },
    {
      "chainId": 137,
      "address": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97",
      "name": "Aave Polygon WMATIC",
      "decimals": 18,
      "symbol": "aPolWMATIC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awmatic.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270"
      }
    },
    {
      "chainId": 137,
      "address": "0x98254592408E389D1dd2dBa318656C2C5c305b4E",
      "name": "Static Aave Polygon WMATIC",
      "decimals": 18,
      "symbol": "stataPolWMATIC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawmatic.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270",
        "underlyingAToken": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97"
      }
    },
    {
      "chainId": 137,
      "address": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf",
      "name": "Aave Polygon CRV",
      "decimals": 18,
      "symbol": "aPolCRV",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acrv.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x172370d5Cd63279eFa6d502DAB29171933a610AF"
      }
    },
    {
      "chainId": 137,
      "address": "0x4356941463eD4d75381AC23C9EF799B5d7C52AD8",
      "name": "Static Aave Polygon CRV",
      "decimals": 18,
      "symbol": "stataPolCRV",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacrv.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x172370d5Cd63279eFa6d502DAB29171933a610AF",
        "underlyingAToken": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf"
      }
    },
    {
      "chainId": 137,
      "address": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA",
      "name": "Aave Polygon SUSHI",
      "decimals": 18,
      "symbol": "aPolSUSHI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asushi.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x0b3F868E0BE5597D5DB7fEB59E1CADBb0fdDa50a"
      }
    },
    {
      "chainId": 137,
      "address": "0xe3eDe71d32240b7EC355F0e5DD1131BBe029F934",
      "name": "Static Aave Polygon SUSHI",
      "decimals": 18,
      "symbol": "stataPolSUSHI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statasushi.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x0b3F868E0BE5597D5DB7fEB59E1CADBb0fdDa50a",
        "underlyingAToken": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA"
      }
    },
    {
      "chainId": 137,
      "address": "0x8Eb270e296023E9D92081fdF967dDd7878724424",
      "name": "Aave Polygon GHST",
      "decimals": 18,
      "symbol": "aPolGHST",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aghst.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x385Eeac5cB85A38A9a07A70c73e0a3271CfB54A7"
      }
    },
    {
      "chainId": 137,
      "address": "0x123319636A6a9c85D9959399304F4cB23F64327e",
      "name": "Static Aave Polygon GHST",
      "decimals": 18,
      "symbol": "stataPolGHST",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataghst.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x385Eeac5cB85A38A9a07A70c73e0a3271CfB54A7",
        "underlyingAToken": "0x8Eb270e296023E9D92081fdF967dDd7878724424"
      }
    },
    {
      "chainId": 137,
      "address": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692",
      "name": "Aave Polygon BAL",
      "decimals": 18,
      "symbol": "aPolBAL",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abal.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x9a71012B13CA4d3D0Cdc72A177DF3ef03b0E76A3"
      }
    },
    {
      "chainId": 137,
      "address": "0x1a8969FD39AbaF228e690B172C4C3Eb7c67F95E1",
      "name": "Static Aave Polygon BAL",
      "decimals": 18,
      "symbol": "stataPolBAL",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statabal.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x9a71012B13CA4d3D0Cdc72A177DF3ef03b0E76A3",
        "underlyingAToken": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692"
      }
    },
    {
      "chainId": 137,
      "address": "0x724dc807b04555b71ed48a6896b6F41593b8C637",
      "name": "Aave Polygon DPI",
      "decimals": 18,
      "symbol": "aPolDPI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adpi.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x85955046DF4668e1DD369D2DE9f3AEB98DD2A369"
      }
    },
    {
      "chainId": 137,
      "address": "0x73B788ACA5f4F0EeB3c6Da453cDf31041a77b36D",
      "name": "Static Aave Polygon DPI",
      "decimals": 18,
      "symbol": "stataPolDPI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadpi.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x85955046DF4668e1DD369D2DE9f3AEB98DD2A369",
        "underlyingAToken": "0x724dc807b04555b71ed48a6896b6F41593b8C637"
      }
    },
    {
      "chainId": 137,
      "address": "0xE111178A87A3BFf0c8d18DECBa5798827539Ae99",
      "name": "STASIS EURS Token (PoS)",
      "decimals": 2,
      "symbol": "EURS",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/eurs.svg"
    },
    {
      "chainId": 137,
      "address": "0x38d693cE1dF5AaDF7bC62595A37D667aD57922e5",
      "name": "Aave Polygon EURS",
      "decimals": 2,
      "symbol": "aPolEURS",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aeurs.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xE111178A87A3BFf0c8d18DECBa5798827539Ae99"
      }
    },
    {
      "chainId": 137,
      "address": "0x02E26888Ed3240BB38f26A2adF96Af9B52b167ea",
      "name": "Static Aave Polygon EURS",
      "decimals": 2,
      "symbol": "stataPolEURS",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataeurs.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xE111178A87A3BFf0c8d18DECBa5798827539Ae99",
        "underlyingAToken": "0x38d693cE1dF5AaDF7bC62595A37D667aD57922e5"
      }
    },
    {
      "chainId": 137,
      "address": "0x4e3Decbb3645551B8A19f0eA1678079FCB33fB4c",
      "name": "Jarvis Synthetic Euro",
      "decimals": 18,
      "symbol": "jEUR",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/jeur.svg"
    },
    {
      "chainId": 137,
      "address": "0x6533afac2E7BCCB20dca161449A13A32D391fb00",
      "name": "Aave Polygon JEUR",
      "decimals": 18,
      "symbol": "aPolJEUR",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ajeur.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x4e3Decbb3645551B8A19f0eA1678079FCB33fB4c"
      }
    },
    {
      "chainId": 137,
      "address": "0xD992DaC78Ef3F34614E6a7d325b7b6A320FC0AB5",
      "name": "Static Aave Polygon JEUR",
      "decimals": 18,
      "symbol": "stataPolJEUR",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statajeur.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x4e3Decbb3645551B8A19f0eA1678079FCB33fB4c",
        "underlyingAToken": "0x6533afac2E7BCCB20dca161449A13A32D391fb00"
      }
    },
    {
      "chainId": 137,
      "address": "0xE0B52e49357Fd4DAf2c15e02058DCE6BC0057db4",
      "name": "EURA (previously agEUR)",
      "decimals": 18,
      "symbol": "EURA",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/eura.svg"
    },
    {
      "chainId": 137,
      "address": "0x8437d7C167dFB82ED4Cb79CD44B7a32A1dd95c77",
      "name": "Aave Polygon AGEUR",
      "decimals": 18,
      "symbol": "aPolAGEUR",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aeura.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xE0B52e49357Fd4DAf2c15e02058DCE6BC0057db4"
      }
    },
    {
      "chainId": 137,
      "address": "0xd3eb8796Ed36f58E03B7b4b5AD417FA74931d2c4",
      "name": "Static Aave Polygon AGEUR",
      "decimals": 18,
      "symbol": "stataPolAGEUR",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataeura.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xE0B52e49357Fd4DAf2c15e02058DCE6BC0057db4",
        "underlyingAToken": "0x8437d7C167dFB82ED4Cb79CD44B7a32A1dd95c77"
      }
    },
    {
      "chainId": 137,
      "address": "0xa3Fa99A148fA48D14Ed51d610c367C61876997F1",
      "name": "miMATIC",
      "decimals": 18,
      "symbol": "miMATIC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/mai.svg"
    },
    {
      "chainId": 137,
      "address": "0xeBe517846d0F36eCEd99C735cbF6131e1fEB775D",
      "name": "Aave Polygon MIMATIC",
      "decimals": 18,
      "symbol": "aPolMIMATIC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xa3Fa99A148fA48D14Ed51d610c367C61876997F1"
      }
    },
    {
      "chainId": 137,
      "address": "0x8486B49433cCed038b51d18Ae3772CDB7E31CA5e",
      "name": "Static Aave Polygon MIMATIC",
      "decimals": 18,
      "symbol": "stataPolMIMATIC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statamai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xa3Fa99A148fA48D14Ed51d610c367C61876997F1",
        "underlyingAToken": "0xeBe517846d0F36eCEd99C735cbF6131e1fEB775D"
      }
    },
    {
      "chainId": 137,
      "address": "0x3A58a54C066FdC0f2D55FC9C89F0415C92eBf3C4",
      "name": "Staked MATIC (PoS)",
      "decimals": 18,
      "symbol": "stMATIC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stmatic.svg"
    },
    {
      "chainId": 137,
      "address": "0xEA1132120ddcDDA2F119e99Fa7A27a0d036F7Ac9",
      "name": "Aave Polygon STMATIC",
      "decimals": 18,
      "symbol": "aPolSTMATIC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/astmatic.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3A58a54C066FdC0f2D55FC9C89F0415C92eBf3C4"
      }
    },
    {
      "chainId": 137,
      "address": "0x867A180B7060fDC27610dC9096E93534F638A315",
      "name": "Static Aave Polygon STMATIC",
      "decimals": 18,
      "symbol": "stataPolSTMATIC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statastmatic.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3A58a54C066FdC0f2D55FC9C89F0415C92eBf3C4",
        "underlyingAToken": "0xEA1132120ddcDDA2F119e99Fa7A27a0d036F7Ac9"
      }
    },
    {
      "chainId": 137,
      "address": "0xfa68FB4628DFF1028CFEc22b4162FCcd0d45efb6",
      "name": "Liquid Staking Matic (PoS)",
      "decimals": 18,
      "symbol": "MaticX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/maticx.svg"
    },
    {
      "chainId": 137,
      "address": "0x80cA0d8C38d2e2BcbaB66aA1648Bd1C7160500FE",
      "name": "Aave Polygon MATICX",
      "decimals": 18,
      "symbol": "aPolMATICX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amaticx.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xfa68FB4628DFF1028CFEc22b4162FCcd0d45efb6"
      }
    },
    {
      "chainId": 137,
      "address": "0xbcDd5709641Af4BE99b1470A2B3A5203539132Ec",
      "name": "Static Aave Polygon MATICX",
      "decimals": 18,
      "symbol": "stataPolMATICX",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statamaticx.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xfa68FB4628DFF1028CFEc22b4162FCcd0d45efb6",
        "underlyingAToken": "0x80cA0d8C38d2e2BcbaB66aA1648Bd1C7160500FE"
      }
    },
    {
      "chainId": 137,
      "address": "0x03b54A6e9a984069379fae1a4fC4dBAE93B3bCCD",
      "name": "Wrapped liquid staked Ether 2.0 (PoS)",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 137,
      "address": "0xf59036CAEBeA7dC4b86638DFA2E3C97dA9FcCd40",
      "name": "Aave Polygon wstETH",
      "decimals": 18,
      "symbol": "aPolwstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x03b54A6e9a984069379fae1a4fC4dBAE93B3bCCD"
      }
    },
    {
      "chainId": 137,
      "address": "0x5274453F4CD5dD7280011a1Cca3B9e1b78EC59A6",
      "name": "Static Aave Polygon wstETH",
      "decimals": 18,
      "symbol": "stataPolwstETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x03b54A6e9a984069379fae1a4fC4dBAE93B3bCCD",
        "underlyingAToken": "0xf59036CAEBeA7dC4b86638DFA2E3C97dA9FcCd40"
      }
    },
    {
      "chainId": 137,
      "address": "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDCn",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 137,
      "address": "0xA4D94019934D8333Ef880ABFFbF2FDd611C762BD",
      "name": "Aave Polygon USDCn",
      "decimals": 6,
      "symbol": "aPolUSDCn",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"
      }
    },
    {
      "chainId": 137,
      "address": "0x2dCa80061632f3F87c9cA28364d1d0c30cD79a19",
      "name": "Static Aave Polygon USDCn",
      "decimals": 6,
      "symbol": "stataPolUSDCn",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359",
        "underlyingAToken": "0xA4D94019934D8333Ef880ABFFbF2FDd611C762BD"
      }
    },
    {
      "chainId": 43114,
      "address": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE",
      "name": "Aave Avalanche DAI",
      "decimals": 18,
      "symbol": "aAvaDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xd586E7F844cEa2F87f50152665BCbc2C279D8d70"
      }
    },
    {
      "chainId": 43114,
      "address": "0x02F3f6c8A432C1e49f3359d7d36887C25d8A5888",
      "name": "Static Aave Avalanche DAI",
      "decimals": 18,
      "symbol": "stataAvaDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xd586E7F844cEa2F87f50152665BCbc2C279D8d70",
        "underlyingAToken": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE"
      }
    },
    {
      "chainId": 43114,
      "address": "0x5947BB275c521040051D82396192181b413227A3",
      "name": "Chainlink Token",
      "decimals": 18,
      "symbol": "LINKe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/link.svg"
    },
    {
      "chainId": 43114,
      "address": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530",
      "name": "Aave Avalanche LINK",
      "decimals": 18,
      "symbol": "aAvaLINK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x5947BB275c521040051D82396192181b413227A3"
      }
    },
    {
      "chainId": 43114,
      "address": "0x8B773Ab77Dff01985D438961dBCE58382a70cA52",
      "name": "Static Aave Avalanche LINK",
      "decimals": 18,
      "symbol": "stataAvaLINK",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x5947BB275c521040051D82396192181b413227A3",
        "underlyingAToken": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530"
      }
    },
    {
      "chainId": 43114,
      "address": "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 43114,
      "address": "0x625E7708f30cA75bfd92586e17077590C60eb4cD",
      "name": "Aave Avalanche USDC",
      "decimals": 6,
      "symbol": "aAvaUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"
      }
    },
    {
      "chainId": 43114,
      "address": "0xC509aB7bB4eDbF193b82264D499a7Fc526Cd01F4",
      "name": "Static Aave Avalanche USDC",
      "decimals": 6,
      "symbol": "stataAvaUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
        "underlyingAToken": "0x625E7708f30cA75bfd92586e17077590C60eb4cD"
      }
    },
    {
      "chainId": 43114,
      "address": "0x078f358208685046a11C85e8ad32895DED33A249",
      "name": "Aave Avalanche WBTC",
      "decimals": 8,
      "symbol": "aAvaWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x50b7545627a5162F82A992c33b87aDc75187B218"
      }
    },
    {
      "chainId": 43114,
      "address": "0xE3C0f42EAF1a4BFe37CbA105e5463564BA7730aE",
      "name": "Static Aave Avalanche WBTC",
      "decimals": 8,
      "symbol": "stataAvaWBTC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x50b7545627a5162F82A992c33b87aDc75187B218",
        "underlyingAToken": "0x078f358208685046a11C85e8ad32895DED33A249"
      }
    },
    {
      "chainId": 43114,
      "address": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8",
      "name": "Aave Avalanche WETH",
      "decimals": 18,
      "symbol": "aAvaWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x49D5c2BdFfac6CE2BFdB6640F4F80f226bc10bAB"
      }
    },
    {
      "chainId": 43114,
      "address": "0xf8E24175D01653fd6AA203C2C17B1e4Dd1CA2731",
      "name": "Static Aave Avalanche WETH",
      "decimals": 18,
      "symbol": "stataAvaWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x49D5c2BdFfac6CE2BFdB6640F4F80f226bc10bAB",
        "underlyingAToken": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8"
      }
    },
    {
      "chainId": 43114,
      "address": "0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7",
      "name": "TetherToken",
      "decimals": 6,
      "symbol": "USDt",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 43114,
      "address": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620",
      "name": "Aave Avalanche USDT",
      "decimals": 6,
      "symbol": "aAvaUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7"
      }
    },
    {
      "chainId": 43114,
      "address": "0x5525Ee69BC1e354B356864187De486fab5AD67d7",
      "name": "Static Aave Avalanche USDT",
      "decimals": 6,
      "symbol": "stataAvaUSDT",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdt.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7",
        "underlyingAToken": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620"
      }
    },
    {
      "chainId": 43114,
      "address": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375",
      "name": "Aave Avalanche AAVE",
      "decimals": 18,
      "symbol": "aAvaAAVE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x63a72806098Bd3D9520cC43356dD78afe5D386D9"
      }
    },
    {
      "chainId": 43114,
      "address": "0xac0746AfD13DEbe2a43a6c8745Fb83Fd2A2909cA",
      "name": "Static Aave Avalanche AAVE",
      "decimals": 18,
      "symbol": "stataAvaAAVE",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x63a72806098Bd3D9520cC43356dD78afe5D386D9",
        "underlyingAToken": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375"
      }
    },
    {
      "chainId": 43114,
      "address": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97",
      "name": "Aave Avalanche WAVAX",
      "decimals": 18,
      "symbol": "aAvaWAVAX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awavax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"
      }
    },
    {
      "chainId": 43114,
      "address": "0x6A02C7a974F1F13A67980C80F774eC1d2eD8f98d",
      "name": "Static Aave Avalanche WAVAX",
      "decimals": 18,
      "symbol": "stataAvaWAVAX",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawavax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
        "underlyingAToken": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97"
      }
    },
    {
      "chainId": 43114,
      "address": "0x2b2C81e08f1Af8835a78Bb2A90AE924ACE0eA4bE",
      "name": "Staked AVAX",
      "decimals": 18,
      "symbol": "sAVAX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/savax.svg"
    },
    {
      "chainId": 43114,
      "address": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf",
      "name": "Aave Avalanche SAVAX",
      "decimals": 18,
      "symbol": "aAvaSAVAX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asavax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2b2C81e08f1Af8835a78Bb2A90AE924ACE0eA4bE"
      }
    },
    {
      "chainId": 43114,
      "address": "0x4F059cA8a2a5BF8895Ee731f2E901cCB769FB95f",
      "name": "Static Aave Avalanche SAVAX",
      "decimals": 18,
      "symbol": "stataAvaSAVAX",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statasavax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2b2C81e08f1Af8835a78Bb2A90AE924ACE0eA4bE",
        "underlyingAToken": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf"
      }
    },
    {
      "chainId": 43114,
      "address": "0xD24C2Ad096400B6FBcd2ad8B24E7acBc21A1da64",
      "name": "Frax",
      "decimals": 18,
      "symbol": "FRAX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/frax.svg"
    },
    {
      "chainId": 43114,
      "address": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA",
      "name": "Aave Avalanche FRAX",
      "decimals": 18,
      "symbol": "aAvaFRAX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afrax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xD24C2Ad096400B6FBcd2ad8B24E7acBc21A1da64"
      }
    },
    {
      "chainId": 43114,
      "address": "0xA3c2ffE702F4cD265B2249AB5f84Fab81FFf6c73",
      "name": "Static Aave Avalanche FRAX",
      "decimals": 18,
      "symbol": "stataAvaFRAX",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statafrax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xD24C2Ad096400B6FBcd2ad8B24E7acBc21A1da64",
        "underlyingAToken": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA"
      }
    },
    {
      "chainId": 43114,
      "address": "0x5c49b268c9841AFF1Cc3B0a418ff5c3442eE3F3b",
      "name": "Mai Stablecoin",
      "decimals": 18,
      "symbol": "MAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/mai.svg"
    },
    {
      "chainId": 43114,
      "address": "0x8Eb270e296023E9D92081fdF967dDd7878724424",
      "name": "Aave Avalanche MAI",
      "decimals": 18,
      "symbol": "aAvaMAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x5c49b268c9841AFF1Cc3B0a418ff5c3442eE3F3b"
      }
    },
    {
      "chainId": 43114,
      "address": "0x08cC59E51BB0Bc322B4D251f7262dB864d6150ce",
      "name": "Static Aave Avalanche MAI",
      "decimals": 18,
      "symbol": "stataAvaMAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statamai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x5c49b268c9841AFF1Cc3B0a418ff5c3442eE3F3b",
        "underlyingAToken": "0x8Eb270e296023E9D92081fdF967dDd7878724424"
      }
    },
    {
      "chainId": 43114,
      "address": "0x152b9d0FdC40C096757F570A51E494bd4b943E50",
      "name": "Bitcoin",
      "decimals": 8,
      "symbol": "BTCb",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/btc.svg"
    },
    {
      "chainId": 43114,
      "address": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692",
      "name": "Aave Avalanche BTC.b",
      "decimals": 8,
      "symbol": "aAvaBTCb",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x152b9d0FdC40C096757F570A51E494bd4b943E50"
      }
    },
    {
      "chainId": 43114,
      "address": "0x34d768cc830c32DcD743321c09A2A702651bF9a2",
      "name": "Static Aave Avalanche BTC.b",
      "decimals": 8,
      "symbol": "stataAvaBTCb",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statabtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x152b9d0FdC40C096757F570A51E494bd4b943E50",
        "underlyingAToken": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692"
      }
    },
    {
      "chainId": 43114,
      "address": "0x00000000eFE302BEAA2b3e6e1b18d08D69a9012a",
      "name": "AUSD",
      "decimals": 6,
      "symbol": "AUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausd.svg"
    },
    {
      "chainId": 43114,
      "address": "0x724dc807b04555b71ed48a6896b6F41593b8C637",
      "name": "Aave Avalanche AUSD",
      "decimals": 6,
      "symbol": "aAvaAUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aausd.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x00000000eFE302BEAA2b3e6e1b18d08D69a9012a"
      }
    },
    {
      "chainId": 8453,
      "address": "0x4200000000000000000000000000000000000006",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 8453,
      "address": "0xD4a0e0b9149BCee3C920d2E00b5dE09138fd8bb7",
      "name": "Aave Base WETH",
      "decimals": 18,
      "symbol": "aBasWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x4200000000000000000000000000000000000006"
      }
    },
    {
      "chainId": 8453,
      "address": "0x468973e3264F2aEba0417A8f2cD0Ec397E738898",
      "name": "Static Aave Base WETH",
      "decimals": 18,
      "symbol": "stataBasWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x4200000000000000000000000000000000000006",
        "underlyingAToken": "0xD4a0e0b9149BCee3C920d2E00b5dE09138fd8bb7"
      }
    },
    {
      "chainId": 8453,
      "address": "0xe298b938631f750DD409fB18227C4a23dCdaab9b",
      "name": "Wrapped Aave Base WETH",
      "decimals": 18,
      "symbol": "waBasWETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x4200000000000000000000000000000000000006",
        "underlyingAToken": "0xD4a0e0b9149BCee3C920d2E00b5dE09138fd8bb7"
      }
    },
    {
      "chainId": 8453,
      "address": "0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22",
      "name": "Coinbase Wrapped Staked ETH",
      "decimals": 18,
      "symbol": "cbETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/cbeth.svg"
    },
    {
      "chainId": 8453,
      "address": "0xcf3D55c10DB69f28fD1A75Bd73f3D8A2d9c595ad",
      "name": "Aave Base cbETH",
      "decimals": 18,
      "symbol": "aBascbETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acbeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22"
      }
    },
    {
      "chainId": 8453,
      "address": "0x16A004065dfb11276DcB29Dc03fb8A85f9A43C6e",
      "name": "Static Aave Base cbETH",
      "decimals": 18,
      "symbol": "stataBascbETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacbeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22",
        "underlyingAToken": "0xcf3D55c10DB69f28fD1A75Bd73f3D8A2d9c595ad"
      }
    },
    {
      "chainId": 8453,
      "address": "0x5e8B674127B321DC344c078e58BBACc3f3008962",
      "name": "Wrapped Aave Base cbETH",
      "decimals": 18,
      "symbol": "waBascbETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacbeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22",
        "underlyingAToken": "0xcf3D55c10DB69f28fD1A75Bd73f3D8A2d9c595ad"
      }
    },
    {
      "chainId": 8453,
      "address": "0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA",
      "name": "USD Base Coin",
      "decimals": 6,
      "symbol": "USDbC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdbc.svg"
    },
    {
      "chainId": 8453,
      "address": "0x0a1d576f3eFeF75b330424287a95A366e8281D54",
      "name": "Aave Base USDbC",
      "decimals": 6,
      "symbol": "aBasUSDbC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdbc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA"
      }
    },
    {
      "chainId": 8453,
      "address": "0x6fCe2756794128B1771324caA860965801DCbCdB",
      "name": "Static Aave Base USDbC",
      "decimals": 6,
      "symbol": "stataBasUSDbC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdbc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA",
        "underlyingAToken": "0x0a1d576f3eFeF75b330424287a95A366e8281D54"
      }
    },
    {
      "chainId": 8453,
      "address": "0x74D4D1D440c9679b1013999Bd91507eAa2fff651",
      "name": "Wrapped Aave Base USDbC",
      "decimals": 6,
      "symbol": "waBasUSDbC",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdbc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xd9aAEc86B65D86f6A7B5B1b0c42FFA531710b6CA",
        "underlyingAToken": "0x0a1d576f3eFeF75b330424287a95A366e8281D54"
      }
    },
    {
      "chainId": 8453,
      "address": "0xc1CBa3fCea344f92D9239c08C0568f6F2F0ee452",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 8453,
      "address": "0x99CBC45ea5bb7eF3a5BC08FB1B7E56bB2442Ef0D",
      "name": "Aave Base wstETH",
      "decimals": 18,
      "symbol": "aBaswstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xc1CBa3fCea344f92D9239c08C0568f6F2F0ee452"
      }
    },
    {
      "chainId": 8453,
      "address": "0x03916e49f794Ab877eFA23597627eE8094E6cbB0",
      "name": "Static Aave Base wstETH",
      "decimals": 18,
      "symbol": "stataBaswstETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xc1CBa3fCea344f92D9239c08C0568f6F2F0ee452",
        "underlyingAToken": "0x99CBC45ea5bb7eF3a5BC08FB1B7E56bB2442Ef0D"
      }
    },
    {
      "chainId": 8453,
      "address": "0x0830820D1A9aa1554364752d6D8F55C836871B74",
      "name": "Wrapped Aave Base wstETH",
      "decimals": 18,
      "symbol": "waBaswstETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xc1CBa3fCea344f92D9239c08C0568f6F2F0ee452",
        "underlyingAToken": "0x99CBC45ea5bb7eF3a5BC08FB1B7E56bB2442Ef0D"
      }
    },
    {
      "chainId": 8453,
      "address": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 8453,
      "address": "0x4e65fE4DbA92790696d040ac24Aa414708F5c0AB",
      "name": "Aave Base USDC",
      "decimals": 6,
      "symbol": "aBasUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
      }
    },
    {
      "chainId": 8453,
      "address": "0x4EA71A20e655794051D1eE8b6e4A3269B13ccaCc",
      "name": "Static Aave Base USDC",
      "decimals": 6,
      "symbol": "stataBasUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
        "underlyingAToken": "0x4e65fE4DbA92790696d040ac24Aa414708F5c0AB"
      }
    },
    {
      "chainId": 8453,
      "address": "0xC768c589647798a6EE01A91FdE98EF2ed046DBD6",
      "name": "Wrapped Aave Base USDC",
      "decimals": 6,
      "symbol": "waBasUSDC",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
        "underlyingAToken": "0x4e65fE4DbA92790696d040ac24Aa414708F5c0AB"
      }
    },
    {
      "chainId": 8453,
      "address": "0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A",
      "name": "Wrapped eETH",
      "decimals": 18,
      "symbol": "weETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weeth.svg"
    },
    {
      "chainId": 8453,
      "address": "0x7C307e128efA31F540F2E2d976C995E0B65F51F6",
      "name": "Aave Base weETH",
      "decimals": 18,
      "symbol": "aBasweETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A"
      }
    },
    {
      "chainId": 8453,
      "address": "0x588159E0d360ffAA978330812f9234818ab46E8E",
      "name": "Static Aave Base weETH",
      "decimals": 18,
      "symbol": "stataBasweETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A",
        "underlyingAToken": "0x7C307e128efA31F540F2E2d976C995E0B65F51F6"
      }
    },
    {
      "chainId": 8453,
      "address": "0x6acD0a165fD70A84b6b50d955ff3628700bAAf4b",
      "name": "Wrapped Aave Base weETH",
      "decimals": 18,
      "symbol": "waBasweETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A",
        "underlyingAToken": "0x7C307e128efA31F540F2E2d976C995E0B65F51F6"
      }
    },
    {
      "chainId": 8453,
      "address": "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf",
      "name": "Coinbase Wrapped BTC",
      "decimals": 8,
      "symbol": "cbBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/cbbtc.svg"
    },
    {
      "chainId": 8453,
      "address": "0xBdb9300b7CDE636d9cD4AFF00f6F009fFBBc8EE6",
      "name": "Aave Base cbBTC",
      "decimals": 8,
      "symbol": "aBascbBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acbbtc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf"
      }
    },
    {
      "chainId": 8453,
      "address": "0xeaCFa728623d0958e3C386bACed79138BCAfC50F",
      "name": "Static Aave Base cbBTC",
      "decimals": 8,
      "symbol": "stataBascbBTC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacbbtc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf",
        "underlyingAToken": "0xBdb9300b7CDE636d9cD4AFF00f6F009fFBBc8EE6"
      }
    },
    {
      "chainId": 8453,
      "address": "0xFA2A03b6f4A65fB1Af64f7d935fDBf78693df9aF",
      "name": "Wrapped Aave Base cbBTC",
      "decimals": 8,
      "symbol": "waBascbBTC",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacbbtc.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf",
        "underlyingAToken": "0xBdb9300b7CDE636d9cD4AFF00f6F009fFBBc8EE6"
      }
    },
    {
      "chainId": 8453,
      "address": "0x2416092f143378750bb29b79eD961ab195CcEea5",
      "name": "Renzo Restaked ETH",
      "decimals": 18,
      "symbol": "ezETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ezeth.svg"
    },
    {
      "chainId": 8453,
      "address": "0xDD5745756C2de109183c6B5bB886F9207bEF114D",
      "name": "Aave Base ezETH",
      "decimals": 18,
      "symbol": "aBasezETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aezeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x2416092f143378750bb29b79eD961ab195CcEea5"
      }
    },
    {
      "chainId": 8453,
      "address": "0xF8F10f39116716e89498c1c5E94137ADa11b2BC7",
      "name": "Wrapped Aave Base ezETH",
      "decimals": 18,
      "symbol": "waBasezETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataezeth.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x2416092f143378750bb29b79eD961ab195CcEea5",
        "underlyingAToken": "0xDD5745756C2de109183c6B5bB886F9207bEF114D"
      }
    },
    {
      "chainId": 8453,
      "address": "0x6Bb7a212910682DCFdbd5BCBb3e28FB4E8da10Ee",
      "name": "Gho Token",
      "decimals": 18,
      "symbol": "GHO",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/gho.svg"
    },
    {
      "chainId": 8453,
      "address": "0x067ae75628177FD257c2B1e500993e1a0baBcBd1",
      "name": "Aave Base GHO",
      "decimals": 18,
      "symbol": "aBasGHO",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/agho.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x6Bb7a212910682DCFdbd5BCBb3e28FB4E8da10Ee"
      }
    },
    {
      "chainId": 8453,
      "address": "0x88b1Cd4b430D95b406E382C3cDBaE54697a0286E",
      "name": "Wrapped Aave Base GHO",
      "decimals": 18,
      "symbol": "waBasGHO",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statagho.svg",
      "extensions": {
        "pool": "0xA238Dd80C259a72e81d7e4664a9801593F98d1c5",
        "underlying": "0x6Bb7a212910682DCFdbd5BCBb3e28FB4E8da10Ee",
        "underlyingAToken": "0x067ae75628177FD257c2B1e500993e1a0baBcBd1"
      }
    },
    {
      "chainId": 1088,
      "address": "0x4c078361FC9BbB78DF910800A991C7c3DD2F6ce0",
      "name": "DAI Token",
      "decimals": 18,
      "symbol": "mDAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 1088,
      "address": "0x85ABAdDcae06efee2CB5F75f33b6471759eFDE24",
      "name": "Aave Metis mDAI",
      "decimals": 18,
      "symbol": "aMetmDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0x4c078361FC9BbB78DF910800A991C7c3DD2F6ce0"
      }
    },
    {
      "chainId": 1088,
      "address": "0x66a2E4cff95BDE6403Ed5541B396aA0B171e5509",
      "name": "Static Aave Metis mDAI",
      "decimals": 18,
      "symbol": "stataMetmDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadai.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0x4c078361FC9BbB78DF910800A991C7c3DD2F6ce0",
        "underlyingAToken": "0x85ABAdDcae06efee2CB5F75f33b6471759eFDE24"
      }
    },
    {
      "chainId": 1088,
      "address": "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
      "name": "Metis Token",
      "decimals": 18,
      "symbol": "Metis",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/metis.svg"
    },
    {
      "chainId": 1088,
      "address": "0x7314Ef2CA509490f65F52CC8FC9E0675C66390b8",
      "name": "Aave Metis METIS",
      "decimals": 18,
      "symbol": "aMetMETIS",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ametis.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000"
      }
    },
    {
      "chainId": 1088,
      "address": "0x5DE732A094A0ceF0eBFEcF0A916bDAB29650a784",
      "name": "Static Aave Metis METIS",
      "decimals": 18,
      "symbol": "stataMetMETIS",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statametis.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
        "underlyingAToken": "0x7314Ef2CA509490f65F52CC8FC9E0675C66390b8"
      }
    },
    {
      "chainId": 1088,
      "address": "0xEA32A96608495e54156Ae48931A7c20f0dcc1a21",
      "name": "USDC Token",
      "decimals": 6,
      "symbol": "mUSDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 1088,
      "address": "0x885C8AEC5867571582545F894A5906971dB9bf27",
      "name": "Aave Metis mUSDC",
      "decimals": 6,
      "symbol": "aMetmUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0xEA32A96608495e54156Ae48931A7c20f0dcc1a21"
      }
    },
    {
      "chainId": 1088,
      "address": "0xb24451C231C6e6A60aC46f45E98a267caae898f4",
      "name": "Static Aave Metis mUSDC",
      "decimals": 6,
      "symbol": "stataMetmUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0xEA32A96608495e54156Ae48931A7c20f0dcc1a21",
        "underlyingAToken": "0x885C8AEC5867571582545F894A5906971dB9bf27"
      }
    },
    {
      "chainId": 1088,
      "address": "0xbB06DCA3AE6887fAbF931640f67cab3e3a16F4dC",
      "name": "USDT Token",
      "decimals": 6,
      "symbol": "mUSDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 1088,
      "address": "0xd9fa75D14c26720d5ce7eE2530793a823e8f07b9",
      "name": "Aave Metis mUSDT",
      "decimals": 6,
      "symbol": "aMetmUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0xbB06DCA3AE6887fAbF931640f67cab3e3a16F4dC"
      }
    },
    {
      "chainId": 1088,
      "address": "0xAAea6F041425B813760dA201d08d46487034A266",
      "name": "Static Aave Metis mUSDT",
      "decimals": 6,
      "symbol": "stataMetmUSDT",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdt.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0xbB06DCA3AE6887fAbF931640f67cab3e3a16F4dC",
        "underlyingAToken": "0xd9fa75D14c26720d5ce7eE2530793a823e8f07b9"
      }
    },
    {
      "chainId": 1088,
      "address": "0x420000000000000000000000000000000000000A",
      "name": "Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 1088,
      "address": "0x8acAe35059C9aE27709028fF6689386a44c09f3a",
      "name": "Aave Metis WETH",
      "decimals": 18,
      "symbol": "aMetWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0x420000000000000000000000000000000000000A"
      }
    },
    {
      "chainId": 1088,
      "address": "0x2f1606864d6322c54b50a1762D4a1ca67f42d23d",
      "name": "Static Aave Metis WETH",
      "decimals": 18,
      "symbol": "stataMetWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x90df02551bB792286e8D4f13E0e357b4Bf1D6a57",
        "underlying": "0x420000000000000000000000000000000000000A",
        "underlyingAToken": "0x8acAe35059C9aE27709028fF6689386a44c09f3a"
      }
    },
    {
      "chainId": 100,
      "address": "0x6A023CCd1ff6F2045C3309768eAd9E68F978f6e1",
      "name": "Wrapped Ether on xDai",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 100,
      "address": "0xa818F1B57c201E092C4A2017A91815034326Efd1",
      "name": "Aave Gnosis WETH",
      "decimals": 18,
      "symbol": "aGnoWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x6A023CCd1ff6F2045C3309768eAd9E68F978f6e1"
      }
    },
    {
      "chainId": 100,
      "address": "0xD843FB478c5aA9759FeA3f3c98D467e2F136190a",
      "name": "Static Aave Gnosis WETH",
      "decimals": 18,
      "symbol": "stataGnoWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x6A023CCd1ff6F2045C3309768eAd9E68F978f6e1",
        "underlyingAToken": "0xa818F1B57c201E092C4A2017A91815034326Efd1"
      }
    },
    {
      "chainId": 100,
      "address": "0x57f664882F762FA37903FC864e2B633D384B411A",
      "name": "Wrapped Aave Gnosis WETH",
      "decimals": 18,
      "symbol": "waGnoWETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x6A023CCd1ff6F2045C3309768eAd9E68F978f6e1",
        "underlyingAToken": "0xa818F1B57c201E092C4A2017A91815034326Efd1"
      }
    },
    {
      "chainId": 100,
      "address": "0x6C76971f98945AE98dD7d4DFcA8711ebea946eA6",
      "name": "Wrapped liquid staked Ether 2.0 from ...",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 100,
      "address": "0x23e4E76D01B2002BE436CE8d6044b0aA2f68B68a",
      "name": "Aave Gnosis wstETH",
      "decimals": 18,
      "symbol": "aGnowstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x6C76971f98945AE98dD7d4DFcA8711ebea946eA6"
      }
    },
    {
      "chainId": 100,
      "address": "0xECfD0638175e291BA3F784A58FB9D38a25418904",
      "name": "Static Aave Gnosis wstETH",
      "decimals": 18,
      "symbol": "stataGnowstETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x6C76971f98945AE98dD7d4DFcA8711ebea946eA6",
        "underlyingAToken": "0x23e4E76D01B2002BE436CE8d6044b0aA2f68B68a"
      }
    },
    {
      "chainId": 100,
      "address": "0x773CDA0CADe2A3d86E6D4e30699d40bB95174ff2",
      "name": "Wrapped Aave Gnosis wstETH",
      "decimals": 18,
      "symbol": "waGnowstETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x6C76971f98945AE98dD7d4DFcA8711ebea946eA6",
        "underlyingAToken": "0x23e4E76D01B2002BE436CE8d6044b0aA2f68B68a"
      }
    },
    {
      "chainId": 100,
      "address": "0x9C58BAcC331c9aa871AFD802DB6379a98e80CEdb",
      "name": "Gnosis Token on xDai",
      "decimals": 18,
      "symbol": "GNO",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/gno.svg"
    },
    {
      "chainId": 100,
      "address": "0xA1Fa064A85266E2Ca82DEe5C5CcEC84DF445760e",
      "name": "Aave Gnosis GNO",
      "decimals": 18,
      "symbol": "aGnoGNO",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/agno.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x9C58BAcC331c9aa871AFD802DB6379a98e80CEdb"
      }
    },
    {
      "chainId": 100,
      "address": "0x2D737e2B0e175f05D0904C208d6C4e40da570f65",
      "name": "Static Aave Gnosis GNO",
      "decimals": 18,
      "symbol": "stataGnoGNO",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statagno.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x9C58BAcC331c9aa871AFD802DB6379a98e80CEdb",
        "underlyingAToken": "0xA1Fa064A85266E2Ca82DEe5C5CcEC84DF445760e"
      }
    },
    {
      "chainId": 100,
      "address": "0x7c16F0185A26Db0AE7a9377f23BC18ea7ce5d644",
      "name": "Wrapped Aave Gnosis GNO",
      "decimals": 18,
      "symbol": "waGnoGNO",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statagno.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x9C58BAcC331c9aa871AFD802DB6379a98e80CEdb",
        "underlyingAToken": "0xA1Fa064A85266E2Ca82DEe5C5CcEC84DF445760e"
      }
    },
    {
      "chainId": 100,
      "address": "0xDDAfbb505ad214D7b80b1f830fcCc89B60fb7A83",
      "name": "USD//C on xDai",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 100,
      "address": "0xc6B7AcA6DE8a6044E0e32d0c841a89244A10D284",
      "name": "Aave Gnosis USDC",
      "decimals": 6,
      "symbol": "aGnoUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xDDAfbb505ad214D7b80b1f830fcCc89B60fb7A83"
      }
    },
    {
      "chainId": 100,
      "address": "0x270bA1f35D8b87510D24F693fcCc0da02e6E4EeB",
      "name": "Static Aave Gnosis USDC",
      "decimals": 6,
      "symbol": "stataGnoUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xDDAfbb505ad214D7b80b1f830fcCc89B60fb7A83",
        "underlyingAToken": "0xc6B7AcA6DE8a6044E0e32d0c841a89244A10D284"
      }
    },
    {
      "chainId": 100,
      "address": "0xe91D153E0b41518A2Ce8Dd3D7944Fa863463a97d",
      "name": "Wrapped XDAI",
      "decimals": 18,
      "symbol": "WXDAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wxdai.svg"
    },
    {
      "chainId": 100,
      "address": "0xd0Dd6cEF72143E22cCED4867eb0d5F2328715533",
      "name": "Aave Gnosis WXDAI",
      "decimals": 18,
      "symbol": "aGnoWXDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awxdai.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xe91D153E0b41518A2Ce8Dd3D7944Fa863463a97d"
      }
    },
    {
      "chainId": 100,
      "address": "0x7f0EAE87Df30C468E0680c83549D0b3DE7664D4B",
      "name": "Static Aave Gnosis WXDAI",
      "decimals": 18,
      "symbol": "stataGnoWXDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawxdai.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xe91D153E0b41518A2Ce8Dd3D7944Fa863463a97d",
        "underlyingAToken": "0xd0Dd6cEF72143E22cCED4867eb0d5F2328715533"
      }
    },
    {
      "chainId": 100,
      "address": "0xcB444e90D8198415266c6a2724b7900fb12FC56E",
      "name": "Monerium EUR emoney",
      "decimals": 18,
      "symbol": "EURe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/eure.svg"
    },
    {
      "chainId": 100,
      "address": "0xEdBC7449a9b594CA4E053D9737EC5Dc4CbCcBfb2",
      "name": "Aave Gnosis EURe",
      "decimals": 18,
      "symbol": "aGnoEURe",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aeure.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xcB444e90D8198415266c6a2724b7900fb12FC56E"
      }
    },
    {
      "chainId": 100,
      "address": "0x8418D17640a74F1614AC3E1826F29e78714488a1",
      "name": "Static Aave Gnosis EURe",
      "decimals": 18,
      "symbol": "stataGnoEURe",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataeure.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xcB444e90D8198415266c6a2724b7900fb12FC56E",
        "underlyingAToken": "0xEdBC7449a9b594CA4E053D9737EC5Dc4CbCcBfb2"
      }
    },
    {
      "chainId": 100,
      "address": "0xaf204776c7245bF4147c2612BF6e5972Ee483701",
      "name": "Savings xDAI",
      "decimals": 18,
      "symbol": "sDAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/sdai.svg"
    },
    {
      "chainId": 100,
      "address": "0x7a5c3860a77a8DC1b225BD46d0fb2ac1C6D191BC",
      "name": "Aave Gnosis sDAI",
      "decimals": 18,
      "symbol": "aGnosDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asdai.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xaf204776c7245bF4147c2612BF6e5972Ee483701"
      }
    },
    {
      "chainId": 100,
      "address": "0xf3f45960f8dE00D8ED614D445a5a268c6F6Dec4f",
      "name": "Static Aave Gnosis sDAI",
      "decimals": 18,
      "symbol": "stataGnosDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statasdai.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0xaf204776c7245bF4147c2612BF6e5972Ee483701",
        "underlyingAToken": "0x7a5c3860a77a8DC1b225BD46d0fb2ac1C6D191BC"
      }
    },
    {
      "chainId": 100,
      "address": "0x2a22f9c3b484c3629090FeED35F17Ff8F88f76F0",
      "name": "Bridged USDC (Gnosis)",
      "decimals": 6,
      "symbol": "USDCe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 100,
      "address": "0xC0333cb85B59a788d8C7CAe5e1Fd6E229A3E5a65",
      "name": "Aave Gnosis USDCe",
      "decimals": 6,
      "symbol": "aGnoUSDCe",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x2a22f9c3b484c3629090FeED35F17Ff8F88f76F0"
      }
    },
    {
      "chainId": 100,
      "address": "0xf0E7eC247b918311afa054E0AEdb99d74c31b809",
      "name": "Static Aave Gnosis USDCe",
      "decimals": 6,
      "symbol": "stataGnoUSDCe",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x2a22f9c3b484c3629090FeED35F17Ff8F88f76F0",
        "underlyingAToken": "0xC0333cb85B59a788d8C7CAe5e1Fd6E229A3E5a65"
      }
    },
    {
      "chainId": 100,
      "address": "0x51350d88c1bd32Cc6A79368c9Fb70373Fb71F375",
      "name": "Wrapped Aave Gnosis USDCe",
      "decimals": 6,
      "symbol": "waGnoUSDCe",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0xb50201558B00496A145fE76f7424749556E326D8",
        "underlying": "0x2a22f9c3b484c3629090FeED35F17Ff8F88f76F0",
        "underlyingAToken": "0xC0333cb85B59a788d8C7CAe5e1Fd6E229A3E5a65"
      }
    },
    {
      "chainId": 56,
      "address": "0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82",
      "name": "PancakeSwap Token",
      "decimals": 18,
      "symbol": "Cake",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/cake.svg"
    },
    {
      "chainId": 56,
      "address": "0x4199CC1F5ed0d796563d7CcB2e036253E2C18281",
      "name": "Aave BNB Smart Chain CAKE",
      "decimals": 18,
      "symbol": "aBnbCAKE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acake.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82"
      }
    },
    {
      "chainId": 56,
      "address": "0x3854354CE3681da1D7F550073061E92a4a7d1B27",
      "name": "Static Aave BNB Smart Chain CAKE",
      "decimals": 18,
      "symbol": "stataBnbCAKE",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statacake.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x0E09FaBB73Bd3Ade0a17ECC321fD13a19e81cE82",
        "underlyingAToken": "0x4199CC1F5ed0d796563d7CcB2e036253E2C18281"
      }
    },
    {
      "chainId": 56,
      "address": "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c",
      "name": "Wrapped BNB",
      "decimals": 18,
      "symbol": "WBNB",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbnb.svg"
    },
    {
      "chainId": 56,
      "address": "0x9B00a09492a626678E5A3009982191586C444Df9",
      "name": "Aave BNB Smart Chain WBNB",
      "decimals": 18,
      "symbol": "aBnbWBNB",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbnb.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
      }
    },
    {
      "chainId": 56,
      "address": "0x436baCb4C66583de4Cb16e13a1A0D9A3075DE425",
      "name": "Static Aave BNB Smart Chain WBNB",
      "decimals": 18,
      "symbol": "stataBnbWBNB",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawbnb.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c",
        "underlyingAToken": "0x9B00a09492a626678E5A3009982191586C444Df9"
      }
    },
    {
      "chainId": 56,
      "address": "0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c",
      "name": "BTCB Token",
      "decimals": 18,
      "symbol": "BTCB",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/btc.svg"
    },
    {
      "chainId": 56,
      "address": "0x56a7ddc4e848EbF43845854205ad71D5D5F72d3D",
      "name": "Aave BNB Smart Chain BTCB",
      "decimals": 18,
      "symbol": "aBnbBTCB",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abtc.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c"
      }
    },
    {
      "chainId": 56,
      "address": "0x1F66b530084079d35478A069d9c4424F9c9C320c",
      "name": "Static Aave BNB Smart Chain BTCB",
      "decimals": 18,
      "symbol": "stataBnbBTCB",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statabtc.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c",
        "underlyingAToken": "0x56a7ddc4e848EbF43845854205ad71D5D5F72d3D"
      }
    },
    {
      "chainId": 56,
      "address": "0x2170Ed0880ac9A755fd29B2688956BD959F933F8",
      "name": "Ethereum Token",
      "decimals": 18,
      "symbol": "ETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/eth.svg"
    },
    {
      "chainId": 56,
      "address": "0x2E94171493fAbE316b6205f1585779C887771E2F",
      "name": "Aave BNB Smart Chain ETH",
      "decimals": 18,
      "symbol": "aBnbETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aeth.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x2170Ed0880ac9A755fd29B2688956BD959F933F8"
      }
    },
    {
      "chainId": 56,
      "address": "0x52077433fB7053D747E2846aD0C18ff5015C368E",
      "name": "Static Aave BNB Smart Chain ETH",
      "decimals": 18,
      "symbol": "stataBnbETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataeth.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x2170Ed0880ac9A755fd29B2688956BD959F933F8",
        "underlyingAToken": "0x2E94171493fAbE316b6205f1585779C887771E2F"
      }
    },
    {
      "chainId": 56,
      "address": "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
      "name": "USD Coin",
      "decimals": 18,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 56,
      "address": "0x00901a076785e0906d1028c7d6372d247bec7d61",
      "name": "Aave BNB Smart Chain USDC",
      "decimals": 18,
      "symbol": "aBnbUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"
      }
    },
    {
      "chainId": 56,
      "address": "0x3906cDdfb781f02B21f21BD81ed7Fd8DC37075E1",
      "name": "Static Aave BNB Smart Chain USDC",
      "decimals": 18,
      "symbol": "stataBnbUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
        "underlyingAToken": "0x00901a076785e0906d1028c7d6372d247bec7d61"
      }
    },
    {
      "chainId": 56,
      "address": "0x55d398326f99059fF775485246999027B3197955",
      "name": "Tether USD",
      "decimals": 18,
      "symbol": "USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 56,
      "address": "0xa9251ca9DE909CB71783723713B21E4233fbf1B1",
      "name": "Aave BNB Smart Chain USDT",
      "decimals": 18,
      "symbol": "aBnbUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x55d398326f99059fF775485246999027B3197955"
      }
    },
    {
      "chainId": 56,
      "address": "0x0471D185cc7Be61E154277cAB2396cD397663da6",
      "name": "Static Aave BNB Smart Chain USDT",
      "decimals": 18,
      "symbol": "stataBnbUSDT",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdt.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x55d398326f99059fF775485246999027B3197955",
        "underlyingAToken": "0xa9251ca9DE909CB71783723713B21E4233fbf1B1"
      }
    },
    {
      "chainId": 56,
      "address": "0xc5f0f7b66764F6ec8C8Dff7BA683102295E16409",
      "name": "First Digital USD",
      "decimals": 18,
      "symbol": "FDUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/fdusd.svg"
    },
    {
      "chainId": 56,
      "address": "0x75bd1A659bdC62e4C313950d44A2416faB43E785",
      "name": "Aave BNB Smart Chain FDUSD",
      "decimals": 18,
      "symbol": "aBnbFDUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afdusd.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0xc5f0f7b66764F6ec8C8Dff7BA683102295E16409"
      }
    },
    {
      "chainId": 56,
      "address": "0x4d074aAa0821073dA827f7bf6a02cF905b394ed0",
      "name": "Static Aave BNB Smart Chain FDUSD",
      "decimals": 18,
      "symbol": "stataBnbFDUSD",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statafdusd.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0xc5f0f7b66764F6ec8C8Dff7BA683102295E16409",
        "underlyingAToken": "0x75bd1A659bdC62e4C313950d44A2416faB43E785"
      }
    },
    {
      "chainId": 56,
      "address": "0x26c5e01524d2E6280A48F2c50fF6De7e52E9611C",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 56,
      "address": "0xBDFd4E51D3c14a232135f04988a42576eFb31519",
      "name": "Aave BNB Smart Chain wstETH",
      "decimals": 18,
      "symbol": "aBnbwstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x6807dc923806fE8Fd134338EABCA509979a7e0cB",
        "underlying": "0x26c5e01524d2E6280A48F2c50fF6De7e52E9611C"
      }
    },
    {
      "chainId": 42161,
      "address": "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1",
      "name": "Dai Stablecoin",
      "decimals": 18,
      "symbol": "DAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 42161,
      "address": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE",
      "name": "Aave Arbitrum DAI",
      "decimals": 18,
      "symbol": "aArbDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1"
      }
    },
    {
      "chainId": 42161,
      "address": "0xc91c5297d7E161aCC74b482aAfCc75B85cc0bfeD",
      "name": "Static Aave Arbitrum DAI",
      "decimals": 18,
      "symbol": "stataArbDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1",
        "underlyingAToken": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE"
      }
    },
    {
      "chainId": 42161,
      "address": "0xf253BD61aEd0E9D62523eA76CD6F38B4a51dA145",
      "name": "Wrapped Aave Arbitrum DAI",
      "decimals": 18,
      "symbol": "waArbDAI",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1",
        "underlyingAToken": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE"
      }
    },
    {
      "chainId": 42161,
      "address": "0xf97f4df75117a78c1A5a0DBb814Af92458539FB4",
      "name": "ChainLink Token",
      "decimals": 18,
      "symbol": "LINK",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/link.svg"
    },
    {
      "chainId": 42161,
      "address": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530",
      "name": "Aave Arbitrum LINK",
      "decimals": 18,
      "symbol": "aArbLINK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xf97f4df75117a78c1A5a0DBb814Af92458539FB4"
      }
    },
    {
      "chainId": 42161,
      "address": "0x27dE098EF2772386cBCf1a4c8BEb886368b7F9a9",
      "name": "Static Aave Arbitrum LINK",
      "decimals": 18,
      "symbol": "stataArbLINK",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xf97f4df75117a78c1A5a0DBb814Af92458539FB4",
        "underlyingAToken": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530"
      }
    },
    {
      "chainId": 42161,
      "address": "0xEAB84053B99f2ec4433F5121A1CB1524c8c998F8",
      "name": "Wrapped Aave Arbitrum LINK",
      "decimals": 18,
      "symbol": "waArbLINK",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xf97f4df75117a78c1A5a0DBb814Af92458539FB4",
        "underlyingAToken": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530"
      }
    },
    {
      "chainId": 42161,
      "address": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8",
      "name": "USD Coin (Arb1)",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 42161,
      "address": "0x625E7708f30cA75bfd92586e17077590C60eb4cD",
      "name": "Aave Arbitrum USDC",
      "decimals": 6,
      "symbol": "aArbUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"
      }
    },
    {
      "chainId": 42161,
      "address": "0x0Bc9E52051f553E75550CA22C196bf132c52Cf0B",
      "name": "Static Aave Arbitrum USDC",
      "decimals": 6,
      "symbol": "stataArbUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8",
        "underlyingAToken": "0x625E7708f30cA75bfd92586e17077590C60eb4cD"
      }
    },
    {
      "chainId": 42161,
      "address": "0xE6D5923281c89DC989D00817387292387552d5C1",
      "name": "Wrapped Aave Arbitrum USDC",
      "decimals": 6,
      "symbol": "waArbUSDC",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8",
        "underlyingAToken": "0x625E7708f30cA75bfd92586e17077590C60eb4cD"
      }
    },
    {
      "chainId": 42161,
      "address": "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
      "name": "Wrapped BTC",
      "decimals": 8,
      "symbol": "WBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbtc.svg"
    },
    {
      "chainId": 42161,
      "address": "0x078f358208685046a11C85e8ad32895DED33A249",
      "name": "Aave Arbitrum WBTC",
      "decimals": 8,
      "symbol": "aArbWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f"
      }
    },
    {
      "chainId": 42161,
      "address": "0x32B95Fbe04e5a51cF99FeeF4e57Cf7e3FC9c5A93",
      "name": "Static Aave Arbitrum WBTC",
      "decimals": 8,
      "symbol": "stataArbWBTC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
        "underlyingAToken": "0x078f358208685046a11C85e8ad32895DED33A249"
      }
    },
    {
      "chainId": 42161,
      "address": "0x52Dc1FEeFA4f9a99221F93D79da46Ae89b8c0967",
      "name": "Wrapped Aave Arbitrum WBTC",
      "decimals": 8,
      "symbol": "waArbWBTC",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2f2a2543B76A4166549F7aaB2e75Bef0aefC5B0f",
        "underlyingAToken": "0x078f358208685046a11C85e8ad32895DED33A249"
      }
    },
    {
      "chainId": 42161,
      "address": "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 42161,
      "address": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8",
      "name": "Aave Arbitrum WETH",
      "decimals": 18,
      "symbol": "aArbWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1"
      }
    },
    {
      "chainId": 42161,
      "address": "0x352F3475716261dCC991Bd5F2aF973eB3D0F5878",
      "name": "Static Aave Arbitrum WETH",
      "decimals": 18,
      "symbol": "stataArbWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
        "underlyingAToken": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8"
      }
    },
    {
      "chainId": 42161,
      "address": "0x4cE13a79f45C1Be00BdABD38B764aC28C082704E",
      "name": "Wrapped Aave Arbitrum WETH",
      "decimals": 18,
      "symbol": "waArbWETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
        "underlyingAToken": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8"
      }
    },
    {
      "chainId": 42161,
      "address": "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
      "name": "Tether USD",
      "decimals": 6,
      "symbol": "USDT",
      "tags": ["underlying"]
    },
    {
      "chainId": 42161,
      "address": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620",
      "name": "Aave Arbitrum USDT",
      "decimals": 6,
      "symbol": "aArbUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"
      }
    },
    {
      "chainId": 42161,
      "address": "0xb165a74407fE1e519d6bCbDeC1Ed3202B35a4140",
      "name": "Static Aave Arbitrum USDT",
      "decimals": 6,
      "symbol": "stataArbUSDT",
      "tags": ["aaveV3", "staticAT"],
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
        "underlyingAToken": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620"
      }
    },
    {
      "chainId": 42161,
      "address": "0xa6D12574eFB239FC1D2099732bd8b5dC6306897F",
      "name": "Wrapped Aave Arbitrum USDT",
      "decimals": 6,
      "symbol": "waArbUSDT",
      "tags": ["aaveV3", "stataToken"],
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
        "underlyingAToken": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620"
      }
    },
    {
      "chainId": 42161,
      "address": "0xba5DdD1f9d7F570dc94a51479a000E3BCE967196",
      "name": "Aave Token",
      "decimals": 18,
      "symbol": "AAVE",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg"
    },
    {
      "chainId": 42161,
      "address": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375",
      "name": "Aave Arbitrum AAVE",
      "decimals": 18,
      "symbol": "aArbAAVE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xba5DdD1f9d7F570dc94a51479a000E3BCE967196"
      }
    },
    {
      "chainId": 42161,
      "address": "0x1C0c8EcED17aE093b3C1a1a8fFeBE2E9513a9346",
      "name": "Static Aave Arbitrum AAVE",
      "decimals": 18,
      "symbol": "stataArbAAVE",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xba5DdD1f9d7F570dc94a51479a000E3BCE967196",
        "underlyingAToken": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375"
      }
    },
    {
      "chainId": 42161,
      "address": "0xD22a58f79e9481D1a88e00c343885A588b34b68B",
      "name": "STASIS EURS Token",
      "decimals": 2,
      "symbol": "EURS",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/eurs.svg"
    },
    {
      "chainId": 42161,
      "address": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97",
      "name": "Aave Arbitrum EURS",
      "decimals": 2,
      "symbol": "aArbEURS",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aeurs.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xD22a58f79e9481D1a88e00c343885A588b34b68B"
      }
    },
    {
      "chainId": 42161,
      "address": "0x9a40747BE51185A416B181789B671E78a8d045dD",
      "name": "Static Aave Arbitrum EURS",
      "decimals": 2,
      "symbol": "stataArbEURS",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataeurs.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xD22a58f79e9481D1a88e00c343885A588b34b68B",
        "underlyingAToken": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97"
      }
    },
    {
      "chainId": 42161,
      "address": "0x5979D7b546E38E414F7E9822514be443A4800529",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 42161,
      "address": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf",
      "name": "Aave Arbitrum wstETH",
      "decimals": 18,
      "symbol": "aArbwstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x5979D7b546E38E414F7E9822514be443A4800529"
      }
    },
    {
      "chainId": 42161,
      "address": "0x7775d4Ae4Dbb79a624fB96AAcDB8Ca74F671c0DF",
      "name": "Static Aave Arbitrum wstETH",
      "decimals": 18,
      "symbol": "stataArbwstETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x5979D7b546E38E414F7E9822514be443A4800529",
        "underlyingAToken": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf"
      }
    },
    {
      "chainId": 42161,
      "address": "0xe98fc055c99DECD8Da0c111B090885d5d15C774E",
      "name": "Wrapped Aave Arbitrum wstETH",
      "decimals": 18,
      "symbol": "waArbwstETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x5979D7b546E38E414F7E9822514be443A4800529",
        "underlyingAToken": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf"
      }
    },
    {
      "chainId": 42161,
      "address": "0x3F56e0c36d275367b8C502090EDF38289b3dEa0d",
      "name": "Mai Stablecoin",
      "decimals": 18,
      "symbol": "MAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/mai.svg"
    },
    {
      "chainId": 42161,
      "address": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA",
      "name": "Aave Arbitrum MAI",
      "decimals": 18,
      "symbol": "aArbMAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3F56e0c36d275367b8C502090EDF38289b3dEa0d"
      }
    },
    {
      "chainId": 42161,
      "address": "0xB4a0a2692D82301703B27082Cda45B083F68CAcE",
      "name": "Static Aave Arbitrum MAI",
      "decimals": 18,
      "symbol": "stataArbMAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statamai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3F56e0c36d275367b8C502090EDF38289b3dEa0d",
        "underlyingAToken": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA"
      }
    },
    {
      "chainId": 42161,
      "address": "0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8",
      "name": "Rocket Pool ETH",
      "decimals": 18,
      "symbol": "rETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/reth.svg"
    },
    {
      "chainId": 42161,
      "address": "0x8Eb270e296023E9D92081fdF967dDd7878724424",
      "name": "Aave Arbitrum rETH",
      "decimals": 18,
      "symbol": "aArbrETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/areth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8"
      }
    },
    {
      "chainId": 42161,
      "address": "0x68235105d6d33A19369D24b746cb7481FB2b34fd",
      "name": "Static Aave Arbitrum rETH",
      "decimals": 18,
      "symbol": "stataArbrETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statareth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8",
        "underlyingAToken": "0x8Eb270e296023E9D92081fdF967dDd7878724424"
      }
    },
    {
      "chainId": 42161,
      "address": "0xbB8A61425DFE172AA3a6f882aAFaBA00B32b7d59",
      "name": "Wrapped Aave Arbitrum rETH",
      "decimals": 18,
      "symbol": "waArbrETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statareth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8",
        "underlyingAToken": "0x8Eb270e296023E9D92081fdF967dDd7878724424"
      }
    },
    {
      "chainId": 42161,
      "address": "0x93b346b6BC2548dA6A1E7d98E9a421B42541425b",
      "name": "LUSD Stablecoin",
      "decimals": 18,
      "symbol": "LUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/lusd.svg"
    },
    {
      "chainId": 42161,
      "address": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692",
      "name": "Aave Arbitrum LUSD",
      "decimals": 18,
      "symbol": "aArbLUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alusd.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x93b346b6BC2548dA6A1E7d98E9a421B42541425b"
      }
    },
    {
      "chainId": 42161,
      "address": "0xDbB6314b5b07E63B7101844c0346309B79f8C20A",
      "name": "Static Aave Arbitrum LUSD",
      "decimals": 18,
      "symbol": "stataArbLUSD",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalusd.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x93b346b6BC2548dA6A1E7d98E9a421B42541425b",
        "underlyingAToken": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692"
      }
    },
    {
      "chainId": 42161,
      "address": "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDCn",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 42161,
      "address": "0x724dc807b04555b71ed48a6896b6F41593b8C637",
      "name": "Aave Arbitrum USDCn",
      "decimals": 6,
      "symbol": "aArbUSDCn",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"
      }
    },
    {
      "chainId": 42161,
      "address": "0x7CFaDFD5645B50bE87d546f42699d863648251ad",
      "name": "Static Aave Arbitrum USDCn",
      "decimals": 6,
      "symbol": "stataArbUSDCn",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
        "underlyingAToken": "0x724dc807b04555b71ed48a6896b6F41593b8C637"
      }
    },
    {
      "chainId": 42161,
      "address": "0x7F6501d3B98eE91f9b9535E4b0ac710Fb0f9e0bc",
      "name": "Wrapped Aave Arbitrum USDCn",
      "decimals": 6,
      "symbol": "waArbUSDCn",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
        "underlyingAToken": "0x724dc807b04555b71ed48a6896b6F41593b8C637"
      }
    },
    {
      "chainId": 42161,
      "address": "0x17FC002b466eEc40DaE837Fc4bE5c67993ddBd6F",
      "name": "Frax",
      "decimals": 18,
      "symbol": "FRAX",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/frax.svg"
    },
    {
      "chainId": 42161,
      "address": "0x38d693cE1dF5AaDF7bC62595A37D667aD57922e5",
      "name": "Aave Arbitrum FRAX",
      "decimals": 18,
      "symbol": "aArbFRAX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afrax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x17FC002b466eEc40DaE837Fc4bE5c67993ddBd6F"
      }
    },
    {
      "chainId": 42161,
      "address": "0x89AEc2023f89E26Dbb7eaa7a98fe3996f9d112A8",
      "name": "Static Aave Arbitrum FRAX",
      "decimals": 18,
      "symbol": "stataArbFRAX",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statafrax.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x17FC002b466eEc40DaE837Fc4bE5c67993ddBd6F",
        "underlyingAToken": "0x38d693cE1dF5AaDF7bC62595A37D667aD57922e5"
      }
    },
    {
      "chainId": 42161,
      "address": "0x912CE59144191C1204E64559FE8253a0e49E6548",
      "name": "Arbitrum",
      "decimals": 18,
      "symbol": "ARB",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/arb.svg"
    },
    {
      "chainId": 42161,
      "address": "0x6533afac2E7BCCB20dca161449A13A32D391fb00",
      "name": "Aave Arbitrum ARB",
      "decimals": 18,
      "symbol": "aArbARB",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aarb.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x912CE59144191C1204E64559FE8253a0e49E6548"
      }
    },
    {
      "chainId": 42161,
      "address": "0x9b5637d7952BC9fa2D693aAE51f3103760Bf2693",
      "name": "Static Aave Arbitrum ARB",
      "decimals": 18,
      "symbol": "stataArbARB",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataarb.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x912CE59144191C1204E64559FE8253a0e49E6548",
        "underlyingAToken": "0x6533afac2E7BCCB20dca161449A13A32D391fb00"
      }
    },
    {
      "chainId": 42161,
      "address": "0xf09EDbF2655B2A56753bD60D22CeAB2AC5D04188",
      "name": "Wrapped Aave Arbitrum ARB",
      "decimals": 18,
      "symbol": "waArbARB",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataarb.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x912CE59144191C1204E64559FE8253a0e49E6548",
        "underlyingAToken": "0x6533afac2E7BCCB20dca161449A13A32D391fb00"
      }
    },
    {
      "chainId": 42161,
      "address": "0x35751007a407ca6FEFfE80b3cB397736D2cf4dbe",
      "name": "Wrapped eETH",
      "decimals": 18,
      "symbol": "weETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weeth.svg"
    },
    {
      "chainId": 42161,
      "address": "0x8437d7C167dFB82ED4Cb79CD44B7a32A1dd95c77",
      "name": "Aave Arbitrum weETH",
      "decimals": 18,
      "symbol": "aArbweETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweeth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x35751007a407ca6FEFfE80b3cB397736D2cf4dbe"
      }
    },
    {
      "chainId": 42161,
      "address": "0xD9E3Ef2c12de90E3b03F7b7E3964956a71920d40",
      "name": "Wrapped Aave Arbitrum weETH",
      "decimals": 18,
      "symbol": "waArbweETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweeth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x35751007a407ca6FEFfE80b3cB397736D2cf4dbe",
        "underlyingAToken": "0x8437d7C167dFB82ED4Cb79CD44B7a32A1dd95c77"
      }
    },
    {
      "chainId": 42161,
      "address": "0x7dfF72693f6A4149b17e7C6314655f6A9F7c8B33",
      "name": "Gho Token",
      "decimals": 18,
      "symbol": "GHO",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/gho.svg"
    },
    {
      "chainId": 42161,
      "address": "0xeBe517846d0F36eCEd99C735cbF6131e1fEB775D",
      "name": "Aave Arbitrum GHO",
      "decimals": 18,
      "symbol": "aArbGHO",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/agho.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x7dfF72693f6A4149b17e7C6314655f6A9F7c8B33"
      }
    },
    {
      "chainId": 42161,
      "address": "0xD9FBA68D89178e3538e708939332c79efC540179",
      "name": "Static Aave Arbitrum GHO",
      "decimals": 18,
      "symbol": "stataArbGHO",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statagho.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x7dfF72693f6A4149b17e7C6314655f6A9F7c8B33",
        "underlyingAToken": "0xeBe517846d0F36eCEd99C735cbF6131e1fEB775D"
      }
    },
    {
      "chainId": 42161,
      "address": "0xD089B4cb88Dacf4e27be869A00e9f7e2E3C18193",
      "name": "Wrapped Aave Arbitrum GHO",
      "decimals": 18,
      "symbol": "waArbGHO",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statagho.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x7dfF72693f6A4149b17e7C6314655f6A9F7c8B33",
        "underlyingAToken": "0xeBe517846d0F36eCEd99C735cbF6131e1fEB775D"
      }
    },
    {
      "chainId": 42161,
      "address": "0x2416092f143378750bb29b79eD961ab195CcEea5",
      "name": "Renzo Restaked ETH",
      "decimals": 18,
      "symbol": "ezETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ezeth.svg"
    },
    {
      "chainId": 42161,
      "address": "0xEA1132120ddcDDA2F119e99Fa7A27a0d036F7Ac9",
      "name": "Aave Arbitrum ezETH",
      "decimals": 18,
      "symbol": "aArbezETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aezeth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2416092f143378750bb29b79eD961ab195CcEea5"
      }
    },
    {
      "chainId": 42161,
      "address": "0x4ff50C17df0D1b788d021ACd85039810a1aA68A1",
      "name": "Wrapped Aave Arbitrum ezETH",
      "decimals": 18,
      "symbol": "waArbezETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataezeth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x2416092f143378750bb29b79eD961ab195CcEea5",
        "underlyingAToken": "0xEA1132120ddcDDA2F119e99Fa7A27a0d036F7Ac9"
      }
    },
    {
      "chainId": 10,
      "address": "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1",
      "name": "Dai Stablecoin",
      "decimals": 18,
      "symbol": "DAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 10,
      "address": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE",
      "name": "Aave Optimism DAI",
      "decimals": 18,
      "symbol": "aOptDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1"
      }
    },
    {
      "chainId": 10,
      "address": "0x6dDc64289bE8a71A707fB057d5d07Cc756055d6e",
      "name": "Static Aave Optimism DAI",
      "decimals": 18,
      "symbol": "stataOptDAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statadai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1",
        "underlyingAToken": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE"
      }
    },
    {
      "chainId": 10,
      "address": "0x350a791Bfc2C21F9Ed5d10980Dad2e2638ffa7f6",
      "name": "ChainLink Token",
      "decimals": 18,
      "symbol": "LINK",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/link.svg"
    },
    {
      "chainId": 10,
      "address": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530",
      "name": "Aave Optimism LINK",
      "decimals": 18,
      "symbol": "aOptLINK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x350a791Bfc2C21F9Ed5d10980Dad2e2638ffa7f6"
      }
    },
    {
      "chainId": 10,
      "address": "0x39BCf217ACc4Bf2fCaF7BC8800E69D986912c75e",
      "name": "Static Aave Optimism LINK",
      "decimals": 18,
      "symbol": "stataOptLINK",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x350a791Bfc2C21F9Ed5d10980Dad2e2638ffa7f6",
        "underlyingAToken": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530"
      }
    },
    {
      "chainId": 10,
      "address": "0x7F5c764cBc14f9669B88837ca1490cCa17c31607",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 10,
      "address": "0x625E7708f30cA75bfd92586e17077590C60eb4cD",
      "name": "Aave Optimism USDC",
      "decimals": 6,
      "symbol": "aOptUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x7F5c764cBc14f9669B88837ca1490cCa17c31607"
      }
    },
    {
      "chainId": 10,
      "address": "0x9F281eb58fd98ad98EDe0fc4C553AD4D73e7Ca2C",
      "name": "Static Aave Optimism USDC",
      "decimals": 6,
      "symbol": "stataOptUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x7F5c764cBc14f9669B88837ca1490cCa17c31607",
        "underlyingAToken": "0x625E7708f30cA75bfd92586e17077590C60eb4cD"
      }
    },
    {
      "chainId": 10,
      "address": "0x68f180fcCe6836688e9084f035309E29Bf0A2095",
      "name": "Wrapped BTC",
      "decimals": 8,
      "symbol": "WBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbtc.svg"
    },
    {
      "chainId": 10,
      "address": "0x078f358208685046a11C85e8ad32895DED33A249",
      "name": "Aave Optimism WBTC",
      "decimals": 8,
      "symbol": "aOptWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x68f180fcCe6836688e9084f035309E29Bf0A2095"
      }
    },
    {
      "chainId": 10,
      "address": "0x6d998FeEFC7B3664eaD09CAf02b5a0fc2E365F18",
      "name": "Static Aave Optimism WBTC",
      "decimals": 8,
      "symbol": "stataOptWBTC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x68f180fcCe6836688e9084f035309E29Bf0A2095",
        "underlyingAToken": "0x078f358208685046a11C85e8ad32895DED33A249"
      }
    },
    {
      "chainId": 10,
      "address": "0x4200000000000000000000000000000000000006",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 10,
      "address": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8",
      "name": "Aave Optimism WETH",
      "decimals": 18,
      "symbol": "aOptWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x4200000000000000000000000000000000000006"
      }
    },
    {
      "chainId": 10,
      "address": "0x98d69620C31869fD4822ceb6ADAB31180475FD37",
      "name": "Static Aave Optimism WETH",
      "decimals": 18,
      "symbol": "stataOptWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x4200000000000000000000000000000000000006",
        "underlyingAToken": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8"
      }
    },
    {
      "chainId": 10,
      "address": "0x94b008aA00579c1307B0EF2c499aD98a8ce58e58",
      "name": "Tether USD",
      "decimals": 6,
      "symbol": "USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 10,
      "address": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620",
      "name": "Aave Optimism USDT",
      "decimals": 6,
      "symbol": "aOptUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x94b008aA00579c1307B0EF2c499aD98a8ce58e58"
      }
    },
    {
      "chainId": 10,
      "address": "0x035c93db04E5aAea54E6cd0261C492a3e0638b37",
      "name": "Static Aave Optimism USDT",
      "decimals": 6,
      "symbol": "stataOptUSDT",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdt.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x94b008aA00579c1307B0EF2c499aD98a8ce58e58",
        "underlyingAToken": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620"
      }
    },
    {
      "chainId": 10,
      "address": "0x76FB31fb4af56892A25e32cFC43De717950c9278",
      "name": "Aave Token",
      "decimals": 18,
      "symbol": "AAVE",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg"
    },
    {
      "chainId": 10,
      "address": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375",
      "name": "Aave Optimism AAVE",
      "decimals": 18,
      "symbol": "aOptAAVE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x76FB31fb4af56892A25e32cFC43De717950c9278"
      }
    },
    {
      "chainId": 10,
      "address": "0xae0Ca1B1Bc6cac26981B5e2b9c40f8Ce8A9082eE",
      "name": "Static Aave Optimism AAVE",
      "decimals": 18,
      "symbol": "stataOptAAVE",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x76FB31fb4af56892A25e32cFC43De717950c9278",
        "underlyingAToken": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375"
      }
    },
    {
      "chainId": 10,
      "address": "0x8c6f28f2F1A3C87F0f938b96d27520d9751ec8d9",
      "name": "Synth sUSD",
      "decimals": 18,
      "symbol": "sUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/susd.svg"
    },
    {
      "chainId": 10,
      "address": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97",
      "name": "Aave Optimism SUSD",
      "decimals": 18,
      "symbol": "aOptSUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asusd.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x8c6f28f2F1A3C87F0f938b96d27520d9751ec8d9"
      }
    },
    {
      "chainId": 10,
      "address": "0x3A956E2Fcc7e71Ea14b0257d40BEbdB287d19652",
      "name": "Static Aave Optimism SUSD",
      "decimals": 18,
      "symbol": "stataOptSUSD",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statasusd.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x8c6f28f2F1A3C87F0f938b96d27520d9751ec8d9",
        "underlyingAToken": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97"
      }
    },
    {
      "chainId": 10,
      "address": "0x4200000000000000000000000000000000000042",
      "name": "Optimism",
      "decimals": 18,
      "symbol": "OP",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/op.svg"
    },
    {
      "chainId": 10,
      "address": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf",
      "name": "Aave Optimism OP",
      "decimals": 18,
      "symbol": "aOptOP",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aop.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x4200000000000000000000000000000000000042"
      }
    },
    {
      "chainId": 10,
      "address": "0xd4F1Cf9A038269FE8F03745C2875591Ad6438ab1",
      "name": "Static Aave Optimism OP",
      "decimals": 18,
      "symbol": "stataOptOP",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataop.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x4200000000000000000000000000000000000042",
        "underlyingAToken": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf"
      }
    },
    {
      "chainId": 10,
      "address": "0x1F32b1c2345538c0c6f582fCB022739c4A194Ebb",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 10,
      "address": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA",
      "name": "Aave Optimism wstETH",
      "decimals": 18,
      "symbol": "aOptwstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x1F32b1c2345538c0c6f582fCB022739c4A194Ebb"
      }
    },
    {
      "chainId": 10,
      "address": "0xb972abef80046A57409e37a7DF5dEf2638917516",
      "name": "Static Aave Optimism wstETH",
      "decimals": 18,
      "symbol": "stataOptwstETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x1F32b1c2345538c0c6f582fCB022739c4A194Ebb",
        "underlyingAToken": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA"
      }
    },
    {
      "chainId": 10,
      "address": "0xc40F949F8a4e094D1b49a23ea9241D289B7b2819",
      "name": "LUSD Stablecoin",
      "decimals": 18,
      "symbol": "LUSD",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/lusd.svg"
    },
    {
      "chainId": 10,
      "address": "0x8Eb270e296023E9D92081fdF967dDd7878724424",
      "name": "Aave Optimism LUSD",
      "decimals": 18,
      "symbol": "aOptLUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alusd.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xc40F949F8a4e094D1b49a23ea9241D289B7b2819"
      }
    },
    {
      "chainId": 10,
      "address": "0x84648dc3Cefb601bc28a49A07a1A8Bad04D30Ad3",
      "name": "Static Aave Optimism LUSD",
      "decimals": 18,
      "symbol": "stataOptLUSD",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statalusd.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xc40F949F8a4e094D1b49a23ea9241D289B7b2819",
        "underlyingAToken": "0x8Eb270e296023E9D92081fdF967dDd7878724424"
      }
    },
    {
      "chainId": 10,
      "address": "0xdFA46478F9e5EA86d57387849598dbFB2e964b02",
      "name": "Mai Stablecoin",
      "decimals": 18,
      "symbol": "MAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/mai.svg"
    },
    {
      "chainId": 10,
      "address": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692",
      "name": "Aave Optimism MAI",
      "decimals": 18,
      "symbol": "aOptMAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/amai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xdFA46478F9e5EA86d57387849598dbFB2e964b02"
      }
    },
    {
      "chainId": 10,
      "address": "0x60495bC8D8Baf7E866888ecC00491e37B47dfF24",
      "name": "Static Aave Optimism MAI",
      "decimals": 18,
      "symbol": "stataOptMAI",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statamai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xdFA46478F9e5EA86d57387849598dbFB2e964b02",
        "underlyingAToken": "0x8ffDf2DE812095b1D19CB146E4c004587C0A0692"
      }
    },
    {
      "chainId": 10,
      "address": "0x9Bcef72be871e61ED4fBbc7630889beE758eb81D",
      "name": "Rocket Pool ETH",
      "decimals": 18,
      "symbol": "rETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/reth.svg"
    },
    {
      "chainId": 10,
      "address": "0x724dc807b04555b71ed48a6896b6F41593b8C637",
      "name": "Aave Optimism rETH",
      "decimals": 18,
      "symbol": "aOptrETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/areth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x9Bcef72be871e61ED4fBbc7630889beE758eb81D"
      }
    },
    {
      "chainId": 10,
      "address": "0xf9ce3c97b4b54F3D16861420f4816D9f68190B7B",
      "name": "Static Aave Optimism rETH",
      "decimals": 18,
      "symbol": "stataOptrETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statareth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x9Bcef72be871e61ED4fBbc7630889beE758eb81D",
        "underlyingAToken": "0x724dc807b04555b71ed48a6896b6F41593b8C637"
      }
    },
    {
      "chainId": 10,
      "address": "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDCn",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 10,
      "address": "0x38d693cE1dF5AaDF7bC62595A37D667aD57922e5",
      "name": "Aave Optimism USDCn",
      "decimals": 6,
      "symbol": "aOptUSDCn",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85"
      }
    },
    {
      "chainId": 10,
      "address": "0x4DD03dfD36548C840B563745e3FBeC320F37BA7e",
      "name": "Static Aave Optimism USDCn",
      "decimals": 6,
      "symbol": "stataOptUSDCn",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85",
        "underlyingAToken": "0x38d693cE1dF5AaDF7bC62595A37D667aD57922e5"
      }
    },
    {
      "chainId": 10,
      "address": "0x41B334E9F2C0ED1f30fD7c351874a6071C53a78E",
      "name": "Wrapped Aave Optimism USDCn",
      "decimals": 6,
      "symbol": "waOptUSDCn",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85",
        "underlyingAToken": "0x38d693cE1dF5AaDF7bC62595A37D667aD57922e5"
      }
    },
    {
      "chainId": 534352,
      "address": "0x5300000000000000000000000000000000000004",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 534352,
      "address": "0xf301805bE1Df81102C957f6d4Ce29d2B8c056B2a",
      "name": "Aave Scroll WETH",
      "decimals": 18,
      "symbol": "aScrWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0x5300000000000000000000000000000000000004"
      }
    },
    {
      "chainId": 534352,
      "address": "0x6b9DfaC194fa78a1882680E2cE19194D006AeEfd",
      "name": "Static Aave Scroll WETH",
      "decimals": 18,
      "symbol": "stataScrWETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0x5300000000000000000000000000000000000004",
        "underlyingAToken": "0xf301805bE1Df81102C957f6d4Ce29d2B8c056B2a"
      }
    },
    {
      "chainId": 534352,
      "address": "0x06eFdBFf2a14a7c8E15944D1F4A48F9F95F663A4",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 534352,
      "address": "0x1D738a3436A8C49CefFbaB7fbF04B660fb528CbD",
      "name": "Aave Scroll USDC",
      "decimals": 6,
      "symbol": "aScrUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0x06eFdBFf2a14a7c8E15944D1F4A48F9F95F663A4"
      }
    },
    {
      "chainId": 534352,
      "address": "0x9fA123bC7E6b61cC8a9D893673a4C6E5392FF4A7",
      "name": "Static Aave Scroll USDC",
      "decimals": 6,
      "symbol": "stataScrUSDC",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0x06eFdBFf2a14a7c8E15944D1F4A48F9F95F663A4",
        "underlyingAToken": "0x1D738a3436A8C49CefFbaB7fbF04B660fb528CbD"
      }
    },
    {
      "chainId": 534352,
      "address": "0xf610A9dfB7C89644979b4A0f27063E9e7d7Cda32",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 534352,
      "address": "0x5B1322eeb46240b02e20062b8F0F9908d525B09c",
      "name": "Aave Scroll wstETH",
      "decimals": 18,
      "symbol": "aScrwstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0xf610A9dfB7C89644979b4A0f27063E9e7d7Cda32"
      }
    },
    {
      "chainId": 534352,
      "address": "0x6e368c4dBf083e18a29aE63FC06AF9deDb3242F0",
      "name": "Static Aave Scroll wstETH",
      "decimals": 18,
      "symbol": "stataScrwstETH",
      "tags": ["aaveV3", "staticAT"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0xf610A9dfB7C89644979b4A0f27063E9e7d7Cda32",
        "underlyingAToken": "0x5B1322eeb46240b02e20062b8F0F9908d525B09c"
      }
    },
    {
      "chainId": 534352,
      "address": "0x01f0a31698C4d065659b9bdC21B3610292a1c506",
      "name": "Wrapped eETH",
      "decimals": 18,
      "symbol": "weETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weeth.svg"
    },
    {
      "chainId": 534352,
      "address": "0xd80A5e16DBDC52Bd1C947CEDfA22c562Be9129C8",
      "name": "Aave Scroll weETH",
      "decimals": 18,
      "symbol": "aScrweETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweeth.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0x01f0a31698C4d065659b9bdC21B3610292a1c506"
      }
    },
    {
      "chainId": 534352,
      "address": "0xd29687c813D741E2F938F4aC377128810E217b1b",
      "name": "Scroll",
      "decimals": 18,
      "symbol": "SCR",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/scr.svg"
    },
    {
      "chainId": 534352,
      "address": "0x25718130C2a8eb94e2e1FAFB5f1cDd4b459aCf64",
      "name": "Aave Scroll SCR",
      "decimals": 18,
      "symbol": "aScrSCR",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ascr.svg",
      "extensions": {
        "pool": "0x11fCfe756c05AD438e312a7fd934381537D3cFfe",
        "underlying": "0xd29687c813D741E2F938F4aC377128810E217b1b"
      }
    },
    {
      "chainId": 324,
      "address": "0x1d17CBcF0D6D143135aE902365D2E5e2A16538D4",
      "name": "USDC",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 324,
      "address": "0xE977F9B2a5ccf0457870a67231F23BE4DaecfbDb",
      "name": "Aave ZkSync USDC",
      "decimals": 6,
      "symbol": "aZksUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x78e30497a3c7527d953c6B1E3541b021A98Ac43c",
        "underlying": "0x1d17CBcF0D6D143135aE902365D2E5e2A16538D4"
      }
    },
    {
      "chainId": 324,
      "address": "0x493257fD37EDB34451f62EDf8D2a0C418852bA4C",
      "name": "Tether USD",
      "decimals": 6,
      "symbol": "USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 324,
      "address": "0xC48574bc5358c967d9447e7Df70230Fdb469e4E7",
      "name": "Aave ZkSync USDT",
      "decimals": 6,
      "symbol": "aZksUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x78e30497a3c7527d953c6B1E3541b021A98Ac43c",
        "underlying": "0x493257fD37EDB34451f62EDf8D2a0C418852bA4C"
      }
    },
    {
      "chainId": 324,
      "address": "0x5AEa5775959fBC2557Cc8789bC1bf90A239D9a91",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 324,
      "address": "0xb7b93bCf82519bB757Fd18b23A389245Dbd8ca64",
      "name": "Aave ZkSync WETH",
      "decimals": 18,
      "symbol": "aZksWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x78e30497a3c7527d953c6B1E3541b021A98Ac43c",
        "underlying": "0x5AEa5775959fBC2557Cc8789bC1bf90A239D9a91"
      }
    },
    {
      "chainId": 324,
      "address": "0x703b52F2b28fEbcB60E1372858AF5b18849FE867",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 324,
      "address": "0xd4e607633F3d984633E946aEA4eb71f92564c1c9",
      "name": "Aave ZkSync wstETH",
      "decimals": 18,
      "symbol": "aZkswstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x78e30497a3c7527d953c6B1E3541b021A98Ac43c",
        "underlying": "0x703b52F2b28fEbcB60E1372858AF5b18849FE867"
      }
    },
    {
      "chainId": 324,
      "address": "0x5A7d6b2F92C77FAD6CCaBd7EE0624E64907Eaf3E",
      "name": "ZKsync",
      "decimals": 18,
      "symbol": "ZK",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/zk.svg"
    },
    {
      "chainId": 324,
      "address": "0xd6cD2c0fC55936498726CacC497832052A9B2D1B",
      "name": "Aave ZkSync ZK",
      "decimals": 18,
      "symbol": "aZksZK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/azk.svg",
      "extensions": {
        "pool": "0x78e30497a3c7527d953c6B1E3541b021A98Ac43c",
        "underlying": "0x5A7d6b2F92C77FAD6CCaBd7EE0624E64907Eaf3E"
      }
    },
    {
      "chainId": 324,
      "address": "0xc1Fa6E2E8667d9bE0Ca938a54c7E0285E9Df924a",
      "name": "Wrapped eETH",
      "decimals": 18,
      "symbol": "weETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weeth.svg"
    },
    {
      "chainId": 324,
      "address": "0xE818A67EE5c0531AFaa31Aa6e20bcAC36227A641",
      "name": "Aave ZkSync weETH",
      "decimals": 18,
      "symbol": "aZksweETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweeth.svg",
      "extensions": {
        "pool": "0x78e30497a3c7527d953c6B1E3541b021A98Ac43c",
        "underlying": "0xc1Fa6E2E8667d9bE0Ca938a54c7E0285E9Df924a"
      }
    },
    {
      "chainId": 324,
      "address": "0xAD17Da2f6Ac76746EF261E835C50b2651ce36DA8",
      "name": "Staked USDe",
      "decimals": 18,
      "symbol": "sUSDe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/susde.svg"
    },
    {
      "chainId": 324,
      "address": "0xF3c9d58B76AC6Ee6811520021e9A9318c49E4CFa",
      "name": "Aave ZkSync sUSDe",
      "decimals": 18,
      "symbol": "aZkssUSDe",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asusde.svg",
      "extensions": {
        "pool": "0x78e30497a3c7527d953c6B1E3541b021A98Ac43c",
        "underlying": "0xAD17Da2f6Ac76746EF261E835C50b2651ce36DA8"
      }
    },
    {
      "chainId": 250,
      "address": "0x8D11eC38a3EB5E956B052f67Da8Bdc9bef8Abf3E",
      "name": "Dai Stablecoin",
      "decimals": 18,
      "symbol": "DAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 250,
      "address": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE",
      "name": "Aave Fantom DAI",
      "decimals": 18,
      "symbol": "aFanDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x8D11eC38a3EB5E956B052f67Da8Bdc9bef8Abf3E"
      }
    },
    {
      "chainId": 250,
      "address": "0xb3654dc3D10Ea7645f8319668E8F54d2574FBdC8",
      "name": "ChainLink",
      "decimals": 18,
      "symbol": "LINK",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/link.svg"
    },
    {
      "chainId": 250,
      "address": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530",
      "name": "Aave Fantom LINK",
      "decimals": 18,
      "symbol": "aFanLINK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xb3654dc3D10Ea7645f8319668E8F54d2574FBdC8"
      }
    },
    {
      "chainId": 250,
      "address": "0x04068DA6C83AFCFA0e13ba15A6696662335D5B75",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 250,
      "address": "0x625E7708f30cA75bfd92586e17077590C60eb4cD",
      "name": "Aave Fantom USDC",
      "decimals": 6,
      "symbol": "aFanUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x04068DA6C83AFCFA0e13ba15A6696662335D5B75"
      }
    },
    {
      "chainId": 250,
      "address": "0x321162Cd933E2Be498Cd2267a90534A804051b11",
      "name": "Bitcoin",
      "decimals": 8,
      "symbol": "BTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/btc.svg"
    },
    {
      "chainId": 250,
      "address": "0x078f358208685046a11C85e8ad32895DED33A249",
      "name": "Aave Fantom WBTC",
      "decimals": 8,
      "symbol": "aFanWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/abtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x321162Cd933E2Be498Cd2267a90534A804051b11"
      }
    },
    {
      "chainId": 250,
      "address": "0x74b23882a30290451A17c44f4F05243b6b58C76d",
      "name": "Ethereum",
      "decimals": 18,
      "symbol": "ETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/eth.svg"
    },
    {
      "chainId": 250,
      "address": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8",
      "name": "Aave Fantom WETH",
      "decimals": 18,
      "symbol": "aFanWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aeth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x74b23882a30290451A17c44f4F05243b6b58C76d"
      }
    },
    {
      "chainId": 250,
      "address": "0x049d68029688eAbF473097a2fC38ef61633A3C7A",
      "name": "Frapped USDT",
      "decimals": 6,
      "symbol": "USDT",
      "tags": ["underlying"]
    },
    {
      "chainId": 250,
      "address": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620",
      "name": "Aave Fantom USDT",
      "decimals": 6,
      "symbol": "aFanUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x049d68029688eAbF473097a2fC38ef61633A3C7A"
      }
    },
    {
      "chainId": 250,
      "address": "0x6a07A792ab2965C72a5B8088d3a069A7aC3a993B",
      "name": "Aave",
      "decimals": 18,
      "symbol": "AAVE",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg"
    },
    {
      "chainId": 250,
      "address": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375",
      "name": "Aave Fantom AAVE",
      "decimals": 18,
      "symbol": "aFanAAVE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x6a07A792ab2965C72a5B8088d3a069A7aC3a993B"
      }
    },
    {
      "chainId": 250,
      "address": "0x21be370D5312f44cB42ce377BC9b8a0cEF1A4C83",
      "name": "Wrapped Fantom",
      "decimals": 18,
      "symbol": "WFTM",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wftm.svg"
    },
    {
      "chainId": 250,
      "address": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97",
      "name": "Aave Fantom WFTM",
      "decimals": 18,
      "symbol": "aFanWFTM",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awftm.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x21be370D5312f44cB42ce377BC9b8a0cEF1A4C83"
      }
    },
    {
      "chainId": 250,
      "address": "0x1E4F97b9f9F913c46F1632781732927B9019C68b",
      "name": "Curve DAO",
      "decimals": 18,
      "symbol": "CRV",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/crv.svg"
    },
    {
      "chainId": 250,
      "address": "0x513c7E3a9c69cA3e22550eF58AC1C0088e918FFf",
      "name": "Aave Fantom CRV",
      "decimals": 18,
      "symbol": "aFanCRV",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/acrv.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x1E4F97b9f9F913c46F1632781732927B9019C68b"
      }
    },
    {
      "chainId": 250,
      "address": "0xae75A438b2E0cB8Bb01Ec1E1e376De11D44477CC",
      "name": "Sushi",
      "decimals": 18,
      "symbol": "SUSHI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/sushi.svg"
    },
    {
      "chainId": 250,
      "address": "0xc45A479877e1e9Dfe9FcD4056c699575a1045dAA",
      "name": "Aave Fantom SUSHI",
      "decimals": 18,
      "symbol": "aFanSUSHI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asushi.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xae75A438b2E0cB8Bb01Ec1E1e376De11D44477CC"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0xEf977d2f931C1978Db5F6747666fa1eACB0d0339",
      "name": "Dai Stablecoin",
      "decimals": 18,
      "symbol": "ONE_DAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0x82E64f49Ed5EC1bC6e43DAD4FC8Af9bb3A2312EE",
      "name": "Aave Harmony DAI",
      "decimals": 18,
      "symbol": "aHarDAI",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/adai.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xEf977d2f931C1978Db5F6747666fa1eACB0d0339"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0x218532a12a389a4a92fC0C5Fb22901D1c19198aA",
      "name": "ChainLink Token",
      "decimals": 18,
      "symbol": "LINK",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/link.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0x191c10Aa4AF7C30e871E70C95dB0E4eb77237530",
      "name": "Aave Harmony LINK",
      "decimals": 18,
      "symbol": "aHarLINK",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/alink.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x218532a12a389a4a92fC0C5Fb22901D1c19198aA"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0x985458E523dB3d53125813eD68c274899e9DfAb4",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "ONE_USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0x625E7708f30cA75bfd92586e17077590C60eb4cD",
      "name": "Aave Harmony USDC",
      "decimals": 6,
      "symbol": "aHarUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x985458E523dB3d53125813eD68c274899e9DfAb4"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0x3095c7557bCb296ccc6e363DE01b760bA031F2d9",
      "name": "Wrapped BTC",
      "decimals": 8,
      "symbol": "ONE_WBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbtc.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0x078f358208685046a11C85e8ad32895DED33A249",
      "name": "Aave Harmony WBTC",
      "decimals": 8,
      "symbol": "aHarWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3095c7557bCb296ccc6e363DE01b760bA031F2d9"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0x6983D1E6DEf3690C4d616b13597A09e6193EA013",
      "name": "ETH",
      "decimals": 18,
      "symbol": "ONE_ETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/eth.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0xe50fA9b3c56FfB159cB0FCA61F5c9D750e8128c8",
      "name": "Aave Harmony WETH",
      "decimals": 18,
      "symbol": "aHarWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aeth.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x6983D1E6DEf3690C4d616b13597A09e6193EA013"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0x3C2B8Be99c50593081EAA2A724F0B8285F5aba8f",
      "name": "Tether USD",
      "decimals": 6,
      "symbol": "ONE_USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0x6ab707Aca953eDAeFBc4fD23bA73294241490620",
      "name": "Aave Harmony USDT",
      "decimals": 6,
      "symbol": "aHarUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0x3C2B8Be99c50593081EAA2A724F0B8285F5aba8f"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0xcF323Aad9E522B93F11c352CaA519Ad0E14eB40F",
      "name": "Aave Token",
      "decimals": 18,
      "symbol": "ONE_AAVE",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0xf329e36C7bF6E5E86ce2150875a84Ce77f477375",
      "name": "Aave Harmony AAVE",
      "decimals": 18,
      "symbol": "aHarAAVE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aaave.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xcF323Aad9E522B93F11c352CaA519Ad0E14eB40F"
      }
    },
    {
      "chainId": 1666600000,
      "address": "0xcF664087a5bB0237a0BAd6742852ec6c8d69A27a",
      "name": "Wrapped ONE",
      "decimals": 18,
      "symbol": "WONE",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wone.svg"
    },
    {
      "chainId": 1666600000,
      "address": "0x6d80113e533a2C0fe82EaBD35f1875DcEA89Ea97",
      "name": "Aave Harmony WONE",
      "decimals": 18,
      "symbol": "aHarWONE",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awone.svg",
      "extensions": {
        "pool": "0x794a61358D6845594F94dc1DB02A252b5b4814aD",
        "underlying": "0xcF664087a5bB0237a0BAd6742852ec6c8d69A27a"
      }
    },
    {
      "chainId": 1,
      "address": "0xC035a7cf15375cE2706766804551791aD035E0C2",
      "name": "Aave Ethereum Lido wstETH",
      "decimals": 18,
      "symbol": "aEthLidowstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0"
      }
    },
    {
      "chainId": 1,
      "address": "0x775F661b0bD1739349b9A2A3EF60be277c5d2D29",
      "name": "Wrapped Aave Ethereum Lido wstETH",
      "decimals": 18,
      "symbol": "waEthLidowstETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statawsteth.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
        "underlyingAToken": "0xC035a7cf15375cE2706766804551791aD035E0C2"
      }
    },
    {
      "chainId": 1,
      "address": "0xfA1fDbBD71B0aA16162D76914d69cD8CB3Ef92da",
      "name": "Aave Ethereum Lido WETH",
      "decimals": 18,
      "symbol": "aEthLidoWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
      }
    },
    {
      "chainId": 1,
      "address": "0x0FE906e030a44eF24CA8c7dC7B7c53A6C4F00ce9",
      "name": "Wrapped Aave Ethereum Lido WETH",
      "decimals": 18,
      "symbol": "waEthLidoWETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
        "underlyingAToken": "0xfA1fDbBD71B0aA16162D76914d69cD8CB3Ef92da"
      }
    },
    {
      "chainId": 1,
      "address": "0x09AA30b182488f769a9824F15E6Ce58591Da4781",
      "name": "Aave Ethereum Lido USDS",
      "decimals": 18,
      "symbol": "aEthLidoUSDS",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausds.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0xdC035D45d973E3EC169d2276DDab16f1e407384F"
      }
    },
    {
      "chainId": 1,
      "address": "0x2A1FBcb52Ed4d9b23daD17E1E8Aed4BB0E6079b8",
      "name": "Aave Ethereum Lido USDC",
      "decimals": 6,
      "symbol": "aEthLidoUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
      }
    },
    {
      "chainId": 1,
      "address": "0xbf5495Efe5DB9ce00f80364C8B423567e58d2110",
      "name": "Renzo Restaked ETH",
      "decimals": 18,
      "symbol": "ezETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ezeth.svg"
    },
    {
      "chainId": 1,
      "address": "0x74e5664394998f13B07aF42446380ACef637969f",
      "name": "Aave Ethereum Lido ezETH",
      "decimals": 18,
      "symbol": "aEthLidoezETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aezeth.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0xbf5495Efe5DB9ce00f80364C8B423567e58d2110"
      }
    },
    {
      "chainId": 1,
      "address": "0xc2015641564a5914A17CB9A92eC8d8feCfa8f2D0",
      "name": "Aave Ethereum Lido sUSDe",
      "decimals": 18,
      "symbol": "aEthLidosUSDe",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/asusde.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0x9D39A5DE30e57443BfF2A8307A4256c8797A3497"
      }
    },
    {
      "chainId": 1,
      "address": "0x18eFE565A5373f430e2F809b97De30335B3ad96A",
      "name": "Aave Ethereum Lido GHO",
      "decimals": 18,
      "symbol": "aEthLidoGHO",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/agho.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0x40D16FC0246aD3160Ccc09B8D0D3A2cD28aE6C2f"
      }
    },
    {
      "chainId": 1,
      "address": "0xC71Ea051a5F82c67ADcF634c36FFE6334793D24C",
      "name": "Wrapped Aave Ethereum Lido GHO",
      "decimals": 18,
      "symbol": "waEthLidoGHO",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statagho.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0x40D16FC0246aD3160Ccc09B8D0D3A2cD28aE6C2f",
        "underlyingAToken": "0x18eFE565A5373f430e2F809b97De30335B3ad96A"
      }
    },
    {
      "chainId": 1,
      "address": "0x56D919E7B25aA42F3F8a4BC77b8982048F2E84B4",
      "name": "Aave Ethereum Lido rsETH",
      "decimals": 18,
      "symbol": "aEthLidorsETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/arseth.svg",
      "extensions": {
        "pool": "0x4e033931ad43597d96D6bcc25c280717730B58B1",
        "underlying": "0xA1290d69c65A6Fe4DF752f95823fae25cB99e5A7"
      }
    },
    {
      "chainId": 1,
      "address": "0xbe1F842e7e0afd2c2322aae5d34bA899544b29db",
      "name": "Aave Ethereum EtherFi weETH",
      "decimals": 18,
      "symbol": "aEthEtherFiweETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweeth.svg",
      "extensions": {
        "pool": "0x0AA97c284e98396202b6A04024F5E2c65026F3c0",
        "underlying": "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"
      }
    },
    {
      "chainId": 1,
      "address": "0x7380c583cDe4409eFF5DD3320D93a45D96B80E2e",
      "name": "Aave Ethereum EtherFi USDC",
      "decimals": 6,
      "symbol": "aEthEtherFiUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x0AA97c284e98396202b6A04024F5E2c65026F3c0",
        "underlying": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
      }
    },
    {
      "chainId": 1,
      "address": "0xdF7f48892244C6106EA784609f7de10AB36F9c7e",
      "name": "Aave Ethereum EtherFi PYUSD",
      "decimals": 6,
      "symbol": "aEthEtherFiPYUSD",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/apyusd.svg",
      "extensions": {
        "pool": "0x0AA97c284e98396202b6A04024F5E2c65026F3c0",
        "underlying": "0x6c3ea9036406852006290770BEdFcAbA0e23A0e8"
      }
    },
    {
      "chainId": 1,
      "address": "0x6914ECCf50837dC61b43ee478a9BD9B439648956",
      "name": "Aave Ethereum EtherFi FRAX",
      "decimals": 18,
      "symbol": "aEthEtherFiFRAX",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/afrax.svg",
      "extensions": {
        "pool": "0x0AA97c284e98396202b6A04024F5E2c65026F3c0",
        "underlying": "0x853d955aCEf822Db058eb8505911ED77F175b99e"
      }
    },
    {
      "chainId": 59144,
      "address": "0xe5D7C2a44FfDDf6b295A15c148167daaAf5Cf34f",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 59144,
      "address": "0x787897dF92703BB3Fc4d9Ee98e15C0b8130Bf163",
      "name": "Aave Linea WETH",
      "decimals": 18,
      "symbol": "aLinWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0xc47b8C00b0f69a36fa203Ffeac0334874574a8Ac",
        "underlying": "0xe5D7C2a44FfDDf6b295A15c148167daaAf5Cf34f"
      }
    },
    {
      "chainId": 59144,
      "address": "0x3aAB2285ddcDdaD8edf438C1bAB47e1a9D05a9b4",
      "name": "Wrapped BTC",
      "decimals": 8,
      "symbol": "WBTC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wbtc.svg"
    },
    {
      "chainId": 59144,
      "address": "0x37f7E06359F98162615e016d0008023D910bB576",
      "name": "Aave Linea WBTC",
      "decimals": 8,
      "symbol": "aLinWBTC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awbtc.svg",
      "extensions": {
        "pool": "0xc47b8C00b0f69a36fa203Ffeac0334874574a8Ac",
        "underlying": "0x3aAB2285ddcDdaD8edf438C1bAB47e1a9D05a9b4"
      }
    },
    {
      "chainId": 59144,
      "address": "0x176211869cA2b568f2A7D4EE941E073a821EE1ff",
      "name": "USDC",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 59144,
      "address": "0x374D7860c4f2f604De0191298dD393703Cce84f3",
      "name": "Aave Linea USDC",
      "decimals": 6,
      "symbol": "aLinUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0xc47b8C00b0f69a36fa203Ffeac0334874574a8Ac",
        "underlying": "0x176211869cA2b568f2A7D4EE941E073a821EE1ff"
      }
    },
    {
      "chainId": 59144,
      "address": "0xA219439258ca9da29E9Cc4cE5596924745e12B93",
      "name": "Tether USD",
      "decimals": 6,
      "symbol": "USDT",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdt.svg"
    },
    {
      "chainId": 59144,
      "address": "0x88231dfEC71D4FF5c1e466D08C321944A7adC673",
      "name": "Aave Linea USDT",
      "decimals": 6,
      "symbol": "aLinUSDT",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdt.svg",
      "extensions": {
        "pool": "0xc47b8C00b0f69a36fa203Ffeac0334874574a8Ac",
        "underlying": "0xA219439258ca9da29E9Cc4cE5596924745e12B93"
      }
    },
    {
      "chainId": 59144,
      "address": "0xB5beDd42000b71FddE22D3eE8a79Bd49A568fC8F",
      "name": "Wrapped liquid staked Ether 2.0",
      "decimals": 18,
      "symbol": "wstETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/wsteth.svg"
    },
    {
      "chainId": 59144,
      "address": "0x58943d20e010d9E34C4511990e232783460d0219",
      "name": "Aave Linea wstETH",
      "decimals": 18,
      "symbol": "aLinwstETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/awsteth.svg",
      "extensions": {
        "pool": "0xc47b8C00b0f69a36fa203Ffeac0334874574a8Ac",
        "underlying": "0xB5beDd42000b71FddE22D3eE8a79Bd49A568fC8F"
      }
    },
    {
      "chainId": 59144,
      "address": "0x2416092f143378750bb29b79eD961ab195CcEea5",
      "name": "Renzo Restaked ETH",
      "decimals": 18,
      "symbol": "ezETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ezeth.svg"
    },
    {
      "chainId": 59144,
      "address": "0x935EfCBeFc1dF0541aFc3fE145134f8c9a0beB89",
      "name": "Aave Linea ezETH",
      "decimals": 18,
      "symbol": "aLinezETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aezeth.svg",
      "extensions": {
        "pool": "0xc47b8C00b0f69a36fa203Ffeac0334874574a8Ac",
        "underlying": "0x2416092f143378750bb29b79eD961ab195CcEea5"
      }
    },
    {
      "chainId": 59144,
      "address": "0x1Bf74C010E6320bab11e2e5A532b5AC15e0b8aA6",
      "name": "Wrapped eETH",
      "decimals": 18,
      "symbol": "weETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weeth.svg"
    },
    {
      "chainId": 59144,
      "address": "0x0C7921aB4888fd06731898b3fffFeB06781D5F4F",
      "name": "Aave Linea weETH",
      "decimals": 18,
      "symbol": "aLinweETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweeth.svg",
      "extensions": {
        "pool": "0xc47b8C00b0f69a36fa203Ffeac0334874574a8Ac",
        "underlying": "0x1Bf74C010E6320bab11e2e5A532b5AC15e0b8aA6"
      }
    },
    {
      "chainId": 146,
      "address": "0x50c42dEAcD8Fc9773493ED674b675bE577f2634b",
      "name": "Wrapped Ether on Sonic",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 146,
      "address": "0xe18Ab82c81E7Eecff32B8A82B1b7d2d23F1EcE96",
      "name": "Aave Sonic WETH",
      "decimals": 18,
      "symbol": "aSonWETH",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x50c42dEAcD8Fc9773493ED674b675bE577f2634b"
      }
    },
    {
      "chainId": 146,
      "address": "0xeB5e9B0ae5bb60274786C747A1A2A798c11271E0",
      "name": "Wrapped Aave Sonic WETH",
      "decimals": 18,
      "symbol": "waSonWETH",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/stataweth.svg",
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x50c42dEAcD8Fc9773493ED674b675bE577f2634b",
        "underlyingAToken": "0xe18Ab82c81E7Eecff32B8A82B1b7d2d23F1EcE96"
      }
    },
    {
      "chainId": 146,
      "address": "0x29219dd400f2Bf60E5a23d13Be72B486D4038894",
      "name": "Bridged USDC (Sonic Labs)",
      "decimals": 6,
      "symbol": "USDCe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 146,
      "address": "0x578Ee1ca3a8E1b54554Da1Bf7C583506C4CD11c6",
      "name": "Aave Sonic USDC",
      "decimals": 6,
      "symbol": "aSonUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x29219dd400f2Bf60E5a23d13Be72B486D4038894"
      }
    },
    {
      "chainId": 146,
      "address": "0x6646248971427B80ce531bdD793e2Eb859347E55",
      "name": "Wrapped Aave Sonic USDC",
      "decimals": 6,
      "symbol": "waSonUSDC",
      "tags": ["aaveV3", "stataToken"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/statausdc.svg",
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x29219dd400f2Bf60E5a23d13Be72B486D4038894",
        "underlyingAToken": "0x578Ee1ca3a8E1b54554Da1Bf7C583506C4CD11c6"
      }
    },
    {
      "chainId": 146,
      "address": "0x039e2fB66102314Ce7b64Ce5Ce3E5183bc94aD38",
      "name": "Wrapped Sonic",
      "decimals": 18,
      "symbol": "wS",
      "tags": ["underlying"]
    },
    {
      "chainId": 146,
      "address": "0x6C5E14A212c1C3e4Baf6f871ac9B1a969918c131",
      "name": "Aave Sonic wS",
      "decimals": 18,
      "symbol": "aSonwS",
      "tags": ["aTokenV3", "aaveV3"],
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x039e2fB66102314Ce7b64Ce5Ce3E5183bc94aD38"
      }
    },
    {
      "chainId": 146,
      "address": "0x18B7B8695165290f2767BC63c36D3dFEa4C0F9bB",
      "name": "Wrapped Aave Sonic wS",
      "decimals": 18,
      "symbol": "waSonwS",
      "tags": ["aaveV3", "stataToken"],
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x039e2fB66102314Ce7b64Ce5Ce3E5183bc94aD38",
        "underlyingAToken": "0x6C5E14A212c1C3e4Baf6f871ac9B1a969918c131"
      }
    }
  ],
  "version": { "major": 3, "minor": 0, "patch": 88 },
  "timestamp": "2025-03-10T07:28:08.643Z"
}
`,
}
