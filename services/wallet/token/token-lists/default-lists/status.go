package defaulttokenlists

import (
	"time"

	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
)

var StatusTokenList = fetcher.FetchedTokenList{
	TokenList: fetcher.TokenList{
		ID:        "status",
		SourceURL: "https://github.com/status-im/status-go/blob/develop/services/wallet/token/token-lists/default-lists/status.go",
	},
	Fetched: time.Unix(1742471186, 0),
	JsonData: `
	{
  "name": "Status Token List",
  "timestamp": "2023-10-18T07:10:03.000Z",
  "version": {
    "major": 1,
    "minor": 0,
    "patch": 0
  },
  "tags": {},
  "keywords": [
    "status",
    "default"
  ],
  "tokens": [
    {
      "address": "0x89d24a6b4ccb1b6faa2625fe562bdd9a23260359",
      "name": "Sai Stablecoin v1.0",
      "symbol": "SAI",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x86fa049857e0209aa7d9e616f7eb3b3b78ecfdb0",
      "name": "EOS",
      "symbol": "EOS",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xd4fa1460f537bb9085d22c7bccb5dd450ef28e3a",
      "name": "Populous Platform",
      "symbol": "PPT",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xb97048628db6b661d4c2aa833e95dbe1a905b280",
      "name": "TenX Pay Token",
      "symbol": "PAY",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x92e78dae1315067a8819efd6dca432de9dcde2e9",
      "name": "Veros",
      "symbol": "VRS",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0xa74476443119a942de498590fe1f2454d7d4ac0d",
      "name": "Golem Network Token",
      "symbol": "GNT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x4156d3342d5c385a87d264f90653733592000581",
      "name": "Salt",
      "symbol": "SALT",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xb8c77482e45f1f44de1745f52c74426c631bdd52",
      "name": "BNB",
      "symbol": "BNB",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xb683D83a532e2Cb7DFa5275eED3698436371cc9f",
      "name": "BTU Protocol",
      "symbol": "BTU",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xe0b7927c4af23765cb51314a0e0521a9645f0e2a",
      "name": "Digix DAO",
      "symbol": "DGD",
      "decimals": 9,
      "chainId": 1
    },
    {
      "address": "0x5ca9a71b1d01849c0a95490cc00559717fcf0d1d",
      "name": "Aeternity",
      "symbol": "AE",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xf230b790e05390fc8295f4d3f60332c93bed42e2",
      "name": "Tronix",
      "symbol": "TRX",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0x255aa6df07540cb5d3d297f0d0d4d84cb52bc8e6",
      "name": "Raiden Token",
      "symbol": "RDN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xaec2e87e0a235266d9c5adc9deb4b2e29b54d009",
      "name": "SingularDTV",
      "symbol": "SNGLS",
      "decimals": 0,
      "chainId": 1
    },
    {
      "address": "0x419d0d8bdd9af5e606ae2232ed285aff190e711b",
      "name": "FunFair",
      "symbol": "FUN",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x888666ca69e0f178ded6d75b5726cee99a87d698",
      "name": "ICONOMI",
      "symbol": "ICN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xb7cb1c96db6b22b0d3d9536e0108d062bd488f74",
      "name": "Walton Token",
      "symbol": "WTC",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xcb97e65f07da24d46bcdd078ebebd7c6e6e3d750",
      "name": "Bytom",
      "symbol": "BTM",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xc42209accc14029c1012fb5680d95fbd6036e2a0",
      "name": "PayPie",
      "symbol": "PPP",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x818fc6c2ec5986bc6e2cbf00939d90556ab12ce5",
      "name": "Kin",
      "symbol": "KIN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x40395044ac3c0c57051906da938b54bd6557f212",
      "name": "MobileGo Token",
      "symbol": "MGO",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xb63b606ac810a52cca15e44bb630fd42d8d1d83d",
      "name": "Monaco",
      "symbol": "MCO",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x7a41e0517a5eca4fdbc7fbeba4d4c47b9ff6dc63",
      "name": "Zeus Shield Coin",
      "symbol": "ZSC",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x0cf0ee63788a0849fe5297f3407f701e122cc023",
      "name": "Streamr (old)",
      "symbol": "XDATA",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xf970b8e36e23f7fc3fd752eea86f8be8d83375a6",
      "name": "Ripio Credit Network Token",
      "symbol": "RCN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x667088b212ce3d06a1b553a7221e1fd19000d9af",
      "name": "WINGS",
      "symbol": "WINGS",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x08711d3b02c8758f2fb3ab4e80228418a7f8e39c",
      "name": "Edgeless",
      "symbol": "EDG",
      "decimals": 0,
      "chainId": 1
    },
    {
      "address": "0x51db5ad35c671a87207d88fc11d593ac0c8415bd",
      "name": "Moeda Loyalty Points",
      "symbol": "MDA",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xe3818504c1b32bf1557b16c238b2e01fd3149c17",
      "name": "PILLAR",
      "symbol": "PLR",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x697beac28b09e122c4332d163985e8a73121b97f",
      "name": "QRL",
      "symbol": "QRL",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x957c30ab0426e0c93cd8241e2c60392d08c6ac8e",
      "name": "Modum Token",
      "symbol": "MOD",
      "decimals": 0,
      "chainId": 1
    },
    {
      "address": "0xe7775a6e9bcf904eb39da2b68c5efb4f9360e08c",
      "name": "Token-as-a-Service",
      "symbol": "TAAS",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0x12b19d3e2ccc14da04fae33e63652ce469b3f2fd",
      "name": "GRID Token",
      "symbol": "GRID",
      "decimals": 12,
      "chainId": 1
    },
    {
      "address": "0x7c5a0ce9267ed19b22f8cae653f198e3e8daf098",
      "name": "SANtiment network token",
      "symbol": "SAN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x983f6d60db79ea8ca4eb9968c6aff8cfa04b3c63",
      "name": "SONM Token",
      "symbol": "SNM",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x12480e24eb5bec1a9d4369cab6a80cad3c0a377a",
      "name": "Substratum",
      "symbol": "SUB",
      "decimals": 2,
      "chainId": 1
    },
    {
      "address": "0x48f775efbe4f5ece6e0df2f7b5932df56823b990",
      "name": "R token",
      "symbol": "R",
      "decimals": 0,
      "chainId": 1
    },
    {
      "address": "0xaf30d2a7e90d7dc361c8c4585e9bb7d2f6f15bc7",
      "name": "FirstBlood Token",
      "symbol": "1ST",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x12fef5e57bf45873cd9b62e9dbd7bfb99e32d73e",
      "name": "Cofoundit",
      "symbol": "CFI",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xf0ee6b27b759c9893ce4f094b49ad28fd15a23e4",
      "name": "Enigma",
      "symbol": "ENG",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x4dc3643dbc642b72c158e7f3d2ff232df61cb6ce",
      "name": "Amber Token",
      "symbol": "AMB",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x90528aeb3a2b736b780fd1b6c478bb7e1d643170",
      "name": "XPlay Token",
      "symbol": "XPA",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x881ef48211982d01e2cb7092c915e647cd40d85c",
      "name": "Open Trading Network",
      "symbol": "OTN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xcb94be6f13a1182e4a4b6140cb7bf2025d28e41b",
      "name": "Trustcoin",
      "symbol": "TRST",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0xaaaf91d9b90df800df4f55c205fd6989c977e73a",
      "name": "Monolith TKN",
      "symbol": "TKN",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x168296bb09e24a88805cb9c33356536b980d3fc5",
      "name": "RHOC",
      "symbol": "RHOC",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xac3da587eac229c9896d919abc235ca4fd7f72c1",
      "name": "Target Coin",
      "symbol": "TGT",
      "decimals": 1,
      "chainId": 1
    },
    {
      "address": "0xf3db5fa2c66b7af3eb0c0b782510816cbe4813b8",
      "name": "Everex",
      "symbol": "EVX",
      "decimals": 4,
      "chainId": 1
    },
    {
      "address": "0x014b50466590340d41307cc54dcee990c8d58aa8",
      "name": "ICOS",
      "symbol": "ICOS",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0x08d32b0da63e2c3bcf8019c9c5d849d7a9d791e6",
      "name": "Dentacoin",
      "symbol": "DCN",
      "decimals": 0,
      "chainId": 1
    },
    {
      "address": "0xced4e93198734ddaff8492d525bd258d49eb388e",
      "name": "Eidoo Token",
      "symbol": "EDO",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x29d75277ac7f0335b2165d0895e8725cbf658d73",
      "name": "BitDice",
      "symbol": "CSNO",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xb2f7eb1f2c37645be61d73953035360e768d81e6",
      "name": "Cobinhood Token",
      "symbol": "COB",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xd4c435f5b09f855c3317c8524cb1f586e42795fa",
      "name": "Cindicator Token",
      "symbol": "CND",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x4df812f6064def1e5e029f1ca858777cc98d2d81",
      "name": "Xaurum",
      "symbol": "XAUR",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x2c974b2d0ba1716e644c1fc59982a89ddd2ff724",
      "name": "Vibe",
      "symbol": "VIB",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x7728dfef5abd468669eb7f9b48a7f70a501ed29d",
      "name": "PRG",
      "symbol": "PRG",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0x6c2adc2073994fb2ccc5032cc2906fa221e9b391",
      "name": "Delphy Token",
      "symbol": "DPY",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x2fe6ab85ebbf7776fee46d191ee4cea322cecf51",
      "name": "CoinDash Token",
      "symbol": "CDT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x08f5a9235b08173b7569f83645d2c7fb55e8ccd8",
      "name": "Tierion Network Token",
      "symbol": "TNT",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x9af4f26941677c706cfecf6d3379ff01bb85d5ab",
      "name": "DomRaiderToken",
      "symbol": "DRT",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x42d6622dece394b54999fbd73d108123806f6a18",
      "name": "SPANK",
      "symbol": "SPANK",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x80046305aaab08f6033b56a360c184391165dc2d",
      "name": "Berlin Coin",
      "symbol": "BRLN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x2c4e8f2d746113d0696ce89b35f0d8bf88e0aeca",
      "name": "Simple Token",
      "symbol": "ST",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x107c4504cd79c5d2696ea0030a8dd4e92601b82e",
      "name": "Bloom Token",
      "symbol": "BLT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x96a65609a7b84e8842732deb08f56c3e21ac6f8a",
      "name": "Centra token",
      "symbol": "Centra",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x2e071d2966aa7d8decb1005885ba1977d6038a65",
      "name": "DICE",
      "symbol": "ROL",
      "decimals": 16,
      "chainId": 1
    },
    {
      "address": "0x9b11efcaaa1890f6ee52c6bb7cf8153ac5d74139",
      "name": "Attention Token of Media",
      "symbol": "ATM",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x80fB784B7eD66730e8b1DBd9820aFD29931aab03",
      "name": "EthLend Token",
      "symbol": "LEND",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xA15C7Ebe1f07CaF6bFF097D8a589fb8AC49Ae5B3",
      "name": "Pundi X Token",
      "symbol": "NPXS",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x0e0989b1f9B8A38983c2BA8053269Ca62Ec9B195",
      "name": "Po.et",
      "symbol": "POE",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xFA1a856Cfa3409CFa145Fa4e20Eb270dF3EB21ab",
      "name": "IOSToken",
      "symbol": "IOST",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xEA26c4aC16D4a5A106820BC8AEE85fd0b7b2b664",
      "name": "QuarkChain Token",
      "symbol": "QKC",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x865ec58b06bF6305B886793AA20A2da31D034E68",
      "name": "Moss Coin",
      "symbol": "MOC",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x8400D94A5cb0fa0D041a3788e395285d61c9ee5e",
      "name": "UniBright",
      "symbol": "UBT",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0x4f3AfEC4E5a3F2A6a1A411DEF7D7dFe50eE057bF",
      "name": "Digix Gold Token",
      "symbol": "DGX",
      "decimals": 9,
      "chainId": 1
    },
    {
      "address": "0xEA38eAa3C86c8F9B751533Ba2E562deb9acDED40",
      "name": "Fuel Token",
      "symbol": "FUEL",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x00000100F2A2bd000715001920eB70D229700085",
      "name": "TrueCAD",
      "symbol": "TCAD",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x6710c63432A2De02954fc0f851db07146a6c0312",
      "name": "SyncFab Smart Manufacturing Blockchain",
      "symbol": "MFG",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x543Ff227F64Aa17eA132Bf9886cAb5DB55DCAddf",
      "name": "DAOstack",
      "symbol": "GEN",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x0E8d6b471e332F140e7d9dbB99E5E3822F728DA6",
      "name": "ABYSS",
      "symbol": "ABYSS",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xB62132e35a6c13ee1EE0f84dC5d40bad8d815206",
      "name": "Nexo",
      "symbol": "NEXO",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x0000000000085d4780B73119b644AE5ecd22b376",
      "name": "TrueUSD",
      "symbol": "TUSD",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xD0a4b8946Cb52f0661273bfbC6fD0E0C75Fc6433",
      "name": "Storm Token",
      "symbol": "STORM",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xaF4DcE16Da2877f8c9e00544c93B62Ac40631F16",
      "name": "Monetha",
      "symbol": "MTH",
      "decimals": 5,
      "chainId": 1
    },
    {
      "address": "0x00000000441378008EA67F4284A57932B1c000a5",
      "name": "TrueGBP",
      "symbol": "TGBP",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xbf2179859fc6D5BEE9Bf9158632Dc51678a4100e",
      "name": "ELF Token",
      "symbol": "ELF",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x20F7A3DdF244dc9299975b4Da1C39F8D5D75f05A",
      "name": "Sapien Network",
      "symbol": "SPN",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0x1a7a8BD9106F2B8D977E08582DC7d24c723ab0DB",
      "name": "AppCoins",
      "symbol": "APPC",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xa3d58c4E56fedCae3a7c43A725aeE9A71F0ece4e",
      "name": "Metronome",
      "symbol": "MET",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x6f259637dcD74C767781E37Bc6133cd6A68aa161",
      "name": "HuobiToken",
      "symbol": "HT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x8f3470A7388c05eE4e7AF3d01D8C722b0FF52374",
      "name": "Veritaseum",
      "symbol": "VERI",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x00006100F7090010005F1bd7aE6122c3C2CF0090",
      "name": "TrueAUD",
      "symbol": "TAUD",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x66497A283E0a007bA3974e837784C6AE323447de",
      "name": "PornToken",
      "symbol": "PT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xB24754bE79281553dc1adC160ddF5Cd9b74361a4",
      "name": "RIALTO",
      "symbol": "XRL",
      "decimals": 9,
      "chainId": 1
    },
    {
      "address": "0x07e3c70653548B04f0A75970C1F81B4CBbFB606f",
      "name": "Delta",
      "symbol": "DLT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x554C20B7c486beeE439277b4540A434566dC4C02",
      "name": "Decision Token",
      "symbol": "HST",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x286BDA1413a2Df81731D4930ce2F862a35A609fE",
      "name": "WaBi",
      "symbol": "WaBi",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xE5a3229CCb22b6484594973A03a3851dCd948756",
      "name": "RAE Token",
      "symbol": "RAE",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x24692791Bc444c5Cd0b81e3CBCaba4b04Acd1F3B",
      "name": "UnikoinGold",
      "symbol": "UKG",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xD46bA6D942050d489DBd938a2C909A5d5039A161",
      "name": "Ampleforth",
      "symbol": "AMPL",
      "decimals": 9,
      "chainId": 1
    },
    {
      "address": "0xA4Bdb11dc0a2bEC88d24A3aa1E6Bb17201112eBe",
      "name": "Stably USD Classic",
      "symbol": "USDSC",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0x81c9151de0C8bafCd325a57E3dB5a5dF1CEBf79c",
      "name": "Datum Token",
      "symbol": "DAT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xa6a840E50bCaa50dA017b91A0D86B8b2d41156EE",
      "name": "EchoLink",
      "symbol": "EKO",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x4a57E687b9126435a9B19E4A802113e266AdeBde",
      "name": "Flexacoin",
      "symbol": "FXC",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xC86D054809623432210c107af2e3F619DcFbf652",
      "name": "SENTINEL PROTOCOL",
      "symbol": "UPP",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x69b148395ce0015c13e36bffbad63f49ef874e03",
      "name": "Data Token",
      "symbol": "DTA",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x5d3a536E4D6DbD6114cc1Ead35777bAB948E3643",
      "name": "Compound Dai",
      "symbol": "cDAI",
      "decimals": 8,
      "chainId": 1
    },
    {
      "address": "0xa7fc5d2453e3f68af0cc1b78bcfee94a1b293650",
      "name": "Spiking",
      "symbol": "SPIKE",
      "decimals": 10,
      "chainId": 1
    },
    {
      "address": "0x8ab7404063ec4dbcfd4598215992dc3f8ec853d7",
      "name": "Akropolis",
      "symbol": "AKRO",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0x9ba00d6856a4edf4665bca2c2309936572473b7e",
      "name": "Aave Interest bearing USDC",
      "symbol": "aUSDC",
      "decimals": 6,
      "chainId": 1
    },
    {
      "address": "0xEEF9f339514298C6A857EfCfC1A762aF84438dEE",
      "name": "Hermez Network Token",
      "symbol": "HEZ",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xaa6e8127831c9de45ae56bb1b0d4d4da6e5665bd",
      "name": "ETH 2x Flexible Leverage Index",
      "symbol": "ETH2x-FLI",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xDd1Ad9A21Ce722C151A836373baBe42c868cE9a4",
      "name": "Universal Basic Income",
      "symbol": "UBI",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xae78736cd615f374d3085123a210448e74fc6393",
      "name": "Rocket Pool",
      "symbol": "rETH",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xb0c7a3ba49c7a6eaba6cd4a96c55a1391070ac9a",
      "name": "Magic Proxy",
      "symbol": "MAGIC",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xc5102fe9359fd9a28f877a67e36b0f050d81a3cc",
      "name": "Hop",
      "symbol": "HOP",
      "decimals": 18,
      "chainId": 1
    },
    {
      "address": "0xc55cf4b03948d7ebc8b9e8bad92643703811d162",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 3
    },
    {
      "address": "0xdee43a267e8726efd60c2e7d5b81552dcd4fa35c",
      "name": "Handy Test Token",
      "symbol": "HND",
      "decimals": 0,
      "chainId": 3
    },
    {
      "address": "0x703d7dc0bc8e314d65436adf985dda51e09ad43b",
      "name": "Lucky Test Token",
      "symbol": "LXS",
      "decimals": 2,
      "chainId": 3
    },
    {
      "address": "0xe639e24346d646e927f323558e6e0031bfc93581",
      "name": "Adi Test Token",
      "symbol": "ADI",
      "decimals": 7,
      "chainId": 3
    },
    {
      "address": "0x2e7cd05f437eb256f363417fd8f920e2efa77540",
      "name": "Wagner Test Token",
      "symbol": "WGN",
      "decimals": 10,
      "chainId": 3
    },
    {
      "address": "0x57cc9b83730e6d22b224e9dc3e370967b44a2de0",
      "name": "Modest Test Token",
      "symbol": "MDS",
      "decimals": 18,
      "chainId": 3
    },
    {
      "address": "0x6ba7dc8dd10880ab83041e60c4ede52bb607864b",
      "name": "Moksha Coin",
      "symbol": "MOKSHA",
      "decimals": 18,
      "chainId": 4
    },
    {
      "address": "0x7d4ccf6af2f0fdad48ee7958bcc28bdef7b732c7",
      "name": "WIBB",
      "symbol": "WIBB",
      "decimals": 18,
      "chainId": 4
    },
    {
      "address": "0x43d5adc3b49130a575ae6e4b00dfa4bc55c71621",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 4
    },
    {
      "address": "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 5
    },
    {
      "address": "0x98339d8c260052b7ad81c28c16c0b98420f2b46a",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 5
    },
    {
      "address": "0x022e292b44b5a146f2e8ee36ff44d3dd863c915c",
      "name": "Xeenus 💪",
      "symbol": "XEENUS",
      "decimals": 18,
      "chainId": 5
    },
    {
      "address": "0xc6fde3fd2cc2b173aec24cc3f267cb3cd78a26b7",
      "name": "Yeenus 💪",
      "symbol": "YEENUS",
      "decimals": 8,
      "chainId": 5
    },
    {
      "address": "0x1f9061b953bba0e36bf50f21876132dcf276fc6e",
      "name": "Zeenus 💪",
      "symbol": "ZEENUS",
      "decimals": 0,
      "chainId": 5
    },
    {
      "address": "0xf4B2cbc3bA04c478F0dC824f4806aC39982Dce73",
      "name": "Tether USD",
      "symbol": "USDT",
      "decimals": 6,
      "chainId": 5
    },
    {
      "address": "0xf2edF1c091f683E3fb452497d9a98A49cBA84666",
      "name": "DAI Stablecoin",
      "symbol": "DAI",
      "decimals": 18,
      "chainId": 5
    },
    {
      "address": "0xc5102fe9359fd9a28f877a67e36b0f050d81a3cc",
      "name": "Hop",
      "symbol": "HOP",
      "decimals": 18,
      "chainId": 10
    },
    {
      "address": "0x9Bcef72be871e61ED4fBbc7630889beE758eb81D",
      "name": "Rocket Pool",
      "symbol": "rETH",
      "decimals": 18,
      "chainId": 10
    },
    {
      "address": "0x3e50bf6703fc132a94e4baff068db2055655f11b",
      "name": "buffiDai",
      "symbol": "BUFF",
      "decimals": 18,
      "chainId": 100
    },
    {
      "address": "0xcb4ceefce514b2d910d3ac529076d18e3add3775",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 420
    },
    {
      "address": "0x4d15a3a2286d883af0aa1b3f21367843fac63e07",
      "name": "True USD",
      "symbol": "TUSD",
      "decimals": 18,
      "chainId": 42161
    },
    {
      "address": "0x680447595e8b7b3aa1b43beb9f6098c79ac2ab3f",
      "name": "Decentralized USD",
      "symbol": "USDD",
      "decimals": 18,
      "chainId": 42161
    },
    {
      "address": "0xc5102fe9359fd9a28f877a67e36b0f050d81a3cc",
      "name": "Hop",
      "symbol": "HOP",
      "decimals": 18,
      "chainId": 42161
    },
    {
      "address": "0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8",
      "name": "Rocket Pool",
      "symbol": "rETH",
      "decimals": 18,
      "chainId": 42161
    },
    {
      "address": "0x17078F231AA8dc256557b49a8f2F72814A71f633",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 421613
    },
    {
      "address": "0x265B25e22bcd7f10a5bD6E6410F10537Cc7567e8",
      "name": "Tether USD",
      "symbol": "USDT",
      "decimals": 6,
      "chainId": 421613
    },
    {
      "address": "0xE452027cdEF746c7Cd3DB31CB700428b16cD8E51",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "address": "0xfDB3b57944943a7724fCc0520eE2B10659969a06",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 84532
    },
    {
      "address": "0x3e622317f8c93f7328350cf0b56d9ed4c620c5d6",
      "name": "Dai Stablecoin",
      "symbol": "DAI",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "address": "0x7439E9Bb6D8a84dd3A23fe621A30F95403F87fB9",
      "name": "WEENUS Token",
      "symbol": "WEENUS",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "address": "0xc21d97673B9E0B3AA53a06439F71fDc1facE393B",
      "name": "XEENUS Token",
      "symbol": "XEENUS",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "address": "0x93fCA4c6E2525C09c95269055B46f16b1459BF9d",
      "name": "YEENUS Token",
      "symbol": "YEENUS",
      "decimals": 8,
      "chainId": 11155111
    },
    {
      "address": "0xe9EF74A6568E9f0e42a587C9363C9BcC582dcC6c",
      "name": "ZEENUS Token",
      "symbol": "ZEENUS",
      "decimals": 0,
      "chainId": 11155111
    },
    {
      "address": "0x07391dbE03e7a0DEa0fce6699500da081537B6c3",
      "name": "WETH9 Token",
      "symbol": "WETH9",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "address": "0x08210F9170F89Ab7658F0B5E3fF39b0E03C594D4",
      "name": "Euro Coin",
      "symbol": "EURC",
      "decimals": 6,
      "chainId": 11155111
    },
    {
      "address": "0x808456652fdb597867f38412077A9182bf77359F",
      "name": "Euro Coin",
      "symbol": "EURC",
      "decimals": 6,
      "chainId": 84532
    },
    {
      "address": "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 11155111
    },
    {
      "address": "0x75faf114eafb1BDbe2F0316DF893fd58CE46AA4d",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 421614
    },
    {
      "address": "0x5fd84259d66Cd46123540766Be93DFE6D43130D7",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 11155420
    },
    {
      "address": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 84532
    },
    {
      "address": "0x5fbdb2315678afecb367f032d93f642f64180aa3",
      "name": "Status",
      "symbol": "SNT",
      "decimals": 18,
      "chainId": 31337
    },
    {
      "address": "0x1C3Ac2a186c6149Ae7Cb4D716eBbD0766E4f898a",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 1660990954
    }
  ]
}
	`,
}
