package hop

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

// List taken from Hop:
// https://github.com/hop-protocol/hop/blob/ef1ca4f8fac002c81fc0dc37ba021125947c6bc2/packages/sdk/src/addresses/mainnet.ts
// https://github.com/hop-protocol/hop/blob/ef1ca4f8fac002c81fc0dc37ba021125947c6bc2/packages/sdk/src/addresses/sepolia.ts

const (
	L1CanonicalToken       = "l1CanonicalToken"
	L1Bridge               = "l1Bridge"
	L1CanonicalBridge      = "l1CanonicalBridge"
	L1MessengerWrapper     = "l1MessengerWrapper"
	CctpL1Bridge           = "cctpL1Bridge"
	CctpMessageTransmitter = "cctpMessageTransmitter"

	L2CanonicalToken  = "l2CanonicalToken"
	L2Bridge          = "l2Bridge"
	L2CanonicalBridge = "l2CanonicalBridge"
	L2HopBridgeToken  = "l2HopBridgeToken"
	L2AmmWrapper      = "l2AmmWrapper"
	L2SaddleSwap      = "l2SaddleSwap"
	L2SaddleLpToken   = "l2SaddleLpToken"
	CctpL2Bridge      = "cctpL2Bridge"
)

var hopBridgeContractAddresses = map[string]map[uint64]map[string]common.Address{
	"USDC": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
			CctpL1Bridge:           common.HexToAddress("0x7e77461CA2a9d82d26FD5e0Da2243BF72eA45747"),
			CctpMessageTransmitter: common.HexToAddress("0x0a992d191deec32afe36203ad87d7d289a738f81"),
		},
		walletCommon.OptimismMainnet: {
			L2CanonicalToken:       common.HexToAddress("0x0b2c639c533813f4aa9d7837caf62653d097ff85"),
			CctpL2Bridge:           common.HexToAddress("0x469147af8Bde580232BE9DC84Bb4EC84d348De24"),
			CctpMessageTransmitter: common.HexToAddress("0x4d41f22c5a0e5c74090899e5a8fb597a8842b3e8"),
		},
		walletCommon.ArbitrumMainnet: {
			L2CanonicalToken:       common.HexToAddress("0xaf88d065e77c8cc2239327c5edb3a432268e5831"),
			CctpL2Bridge:           common.HexToAddress("0x6504BFcaB789c35325cA4329f1f41FaC340bf982"),
			CctpMessageTransmitter: common.HexToAddress("0xC30362313FBBA5cf9163F0bb16a0e01f01A896ca"),
		},
		walletCommon.EthereumSepolia: {
			L1CanonicalToken: common.HexToAddress("0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238"),
			CctpL1Bridge:     common.HexToAddress("0x05fda2db623fa6a89a2db33550848ab2006a4427"),
		},
		walletCommon.OptimismSepolia: {
			L2CanonicalToken: common.HexToAddress("0x5fd84259d66Cd46123540766Be93DFE6D43130D7"),
			CctpL2Bridge:     common.HexToAddress("0x9f3B8679c73C2Fef8b59B4f3444d4e156fb70AA5"),
		},
		walletCommon.ArbitrumSepolia: {
			L2CanonicalToken: common.HexToAddress("0x75faf114eafb1BDbe2F0316DF893fd58CE46AA4d"),
			CctpL2Bridge:     common.HexToAddress("0x9f3B8679c73C2Fef8b59B4f3444d4e156fb70AA5"),
		},
	},
	"USDC.e": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken:       common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
			L1Bridge:               common.HexToAddress("0x3666f603Cc164936C1b87e207F36BEBa4AC5f18a"),
			CctpL1Bridge:           common.HexToAddress("0x7e77461CA2a9d82d26FD5e0Da2243BF72eA45747"),
			CctpMessageTransmitter: common.HexToAddress("0x0a992d191deec32afe36203ad87d7d289a738f81"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper:     common.HexToAddress("0x6587a6164B091a058aCba2e91f971454Ec172940"),
			L2CanonicalBridge:      common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:       common.HexToAddress("0x7F5c764cBc14f9669B88837ca1490cCa17c31607"),
			L2Bridge:               common.HexToAddress("0xa81D244A1814468C734E5b4101F7b9c0c577a8fC"),
			CctpL2Bridge:           common.HexToAddress("0x469147af8Bde580232BE9DC84Bb4EC84d348De24"),
			CctpMessageTransmitter: common.HexToAddress("0x4d41f22c5a0e5c74090899e5a8fb597a8842b3e8"),
			L2HopBridgeToken:       common.HexToAddress("0x25D8039bB044dC227f741a9e381CA4cEAE2E6aE8"),
			L2AmmWrapper:           common.HexToAddress("0x2ad09850b0CA4c7c1B33f5AcD6cBAbCaB5d6e796"),
			L2SaddleSwap:           common.HexToAddress("0x3c0FFAca566fCcfD9Cc95139FEF6CBA143795963"),
			L2SaddleLpToken:        common.HexToAddress("0x2e17b8193566345a2Dd467183526dEdc42d2d5A8"),
		},
		walletCommon.ArbitrumMainnet: {
			L1CanonicalBridge:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper:     common.HexToAddress("0x39Bf4A32E689B6a79360854b7c901e991085D6a3"),
			L2CanonicalBridge:      common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2CanonicalToken:       common.HexToAddress("0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8"),
			L2Bridge:               common.HexToAddress("0x0e0E3d2C5c292161999474247956EF542caBF8dd"),
			CctpL2Bridge:           common.HexToAddress("0x6504BFcaB789c35325cA4329f1f41FaC340bf982"),
			CctpMessageTransmitter: common.HexToAddress("0xC30362313FBBA5cf9163F0bb16a0e01f01A896ca"),
			L2HopBridgeToken:       common.HexToAddress("0x0ce6c85cF43553DE10FC56cecA0aef6Ff0DD444d"),
			L2AmmWrapper:           common.HexToAddress("0xe22D2beDb3Eca35E6397e0C6D62857094aA26F52"),
			L2SaddleSwap:           common.HexToAddress("0x10541b07d8Ad2647Dc6cD67abd4c03575dade261"),
			L2SaddleLpToken:        common.HexToAddress("0xB67c014FA700E69681a673876eb8BAFAA36BFf71"),
		},
		walletCommon.EthereumSepolia: {
			L1CanonicalToken: common.HexToAddress("0x95B01328BA6f4de261C4907fB35eE3c4968e9CEF"),
			CctpL1Bridge:     common.HexToAddress("0x98bc5b835686e1a00e6c2168af162905899e93d6"),
		},
		walletCommon.OptimismSepolia: {
			L2CanonicalToken: common.HexToAddress("0xB15312eA17d95375E64317C363A0e6304330D82e"),
		},
	},
	"USDT": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
			L1Bridge:         common.HexToAddress("0x3E4a3a4796d16c0Cd582C382691998f7c06420B6"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x9fc22E269c3752620EB281ce470855886b982501"),
			L2CanonicalBridge:  common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:   common.HexToAddress("0x94b008aA00579c1307B0EF2c499aD98a8ce58e58"),
			L2Bridge:           common.HexToAddress("0x46ae9BaB8CEA96610807a275EBD36f8e916b5C61"),
			L2HopBridgeToken:   common.HexToAddress("0x2057C8ECB70Afd7Bee667d76B4CD373A325b1a20"),
			L2AmmWrapper:       common.HexToAddress("0x7D269D3E0d61A05a0bA976b7DBF8805bF844AF3F"),
			L2SaddleSwap:       common.HexToAddress("0xeC4B41Af04cF917b54AEb6Df58c0f8D78895b5Ef"),
			L2SaddleLpToken:    common.HexToAddress("0xF753A50fc755c6622BBCAa0f59F0522f264F006e"),
		},
		walletCommon.ArbitrumMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x967F8E2B66D624Ad544CB59a230b867Ac3dC60dc"),
			L2CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2CanonicalToken:   common.HexToAddress("0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"),
			L2Bridge:           common.HexToAddress("0x72209Fe68386b37A40d6bCA04f78356fd342491f"),
			L2HopBridgeToken:   common.HexToAddress("0x12e59C59D282D2C00f3166915BED6DC2F5e2B5C7"),
			L2AmmWrapper:       common.HexToAddress("0xCB0a4177E0A60247C0ad18Be87f8eDfF6DD30283"),
			L2SaddleSwap:       common.HexToAddress("0x18f7402B673Ba6Fb5EA4B95768aABb8aaD7ef18a"),
			L2SaddleLpToken:    common.HexToAddress("0xCe3B19D820CB8B9ae370E423B0a329c4314335fE"),
		},
	},
	"MATIC": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0"),
			L1Bridge:         common.HexToAddress("0x22B1Cbb8D98a01a3B71D034BB899775A76Eb1cc2"),
		},
	},
	"DAI": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
			L1Bridge:         common.HexToAddress("0x3d4Cc8A61c7528Fd86C55cfe061a78dCBA48EDd1"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x115F423b958A2847af0F5bF314DB0f27c644c308"),
			L2CanonicalBridge:  common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:   common.HexToAddress("0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1"),
			L2Bridge:           common.HexToAddress("0x7191061D5d4C60f598214cC6913502184BAddf18"),
			L2HopBridgeToken:   common.HexToAddress("0x56900d66D74Cb14E3c86895789901C9135c95b16"),
			L2AmmWrapper:       common.HexToAddress("0xb3C68a491608952Cb1257FC9909a537a0173b63B"),
			L2SaddleSwap:       common.HexToAddress("0xF181eD90D6CfaC84B8073FdEA6D34Aa744B41810"),
			L2SaddleLpToken:    common.HexToAddress("0x22D63A26c730d49e5Eab461E4f5De1D8BdF89C92"),
		},
		walletCommon.ArbitrumMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x2d6fd82C7f531328BCaCA96EF985325C0894dB62"),
			L2CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2CanonicalToken:   common.HexToAddress("0xDA10009cBd5D07dd0CeCc66161FC93D7c9000da1"),
			L2Bridge:           common.HexToAddress("0x7aC115536FE3A185100B2c4DE4cb328bf3A58Ba6"),
			L2HopBridgeToken:   common.HexToAddress("0x46ae9BaB8CEA96610807a275EBD36f8e916b5C61"),
			L2AmmWrapper:       common.HexToAddress("0xe7F40BF16AB09f4a6906Ac2CAA4094aD2dA48Cc2"),
			L2SaddleSwap:       common.HexToAddress("0xa5A33aB9063395A90CCbEa2D86a62EcCf27B5742"),
			L2SaddleLpToken:    common.HexToAddress("0x68f5d998F00bB2460511021741D098c05721d8fF"),
		},
	},
	"ETH": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1Bridge:         common.HexToAddress("0xb8901acB165ed027E32754E0FFe830802919727f"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0xa45DF1A388049fb8d76E72D350d24E2C3F7aEBd1"),
			L2CanonicalBridge:  common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:   common.HexToAddress("0x4200000000000000000000000000000000000006"),
			L2Bridge:           common.HexToAddress("0x83f6244Bd87662118d96D9a6D44f09dffF14b30E"),
			L2HopBridgeToken:   common.HexToAddress("0xE38faf9040c7F09958c638bBDB977083722c5156"),
			L2AmmWrapper:       common.HexToAddress("0x86cA30bEF97fB651b8d866D45503684b90cb3312"),
			L2SaddleSwap:       common.HexToAddress("0xaa30D6bba6285d0585722e2440Ff89E23EF68864"),
			L2SaddleLpToken:    common.HexToAddress("0x5C2048094bAaDe483D0b1DA85c3Da6200A88a849"),
		},
		walletCommon.ArbitrumMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0xDD378a11475D588908001E0E99E4fD89ABda5434"),
			L2CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2CanonicalToken:   common.HexToAddress("0x82aF49447D8a07e3bd95BD0d56f35241523fBab1"),
			L2Bridge:           common.HexToAddress("0x3749C4f034022c39ecafFaBA182555d4508caCCC"),
			L2HopBridgeToken:   common.HexToAddress("0xDa7c0de432a9346bB6e96aC74e3B61A36d8a77eB"),
			L2AmmWrapper:       common.HexToAddress("0x33ceb27b39d2Bb7D2e61F7564d3Df29344020417"),
			L2SaddleSwap:       common.HexToAddress("0x652d27c0F72771Ce5C76fd400edD61B406Ac6D97"),
			L2SaddleLpToken:    common.HexToAddress("0x59745774Ed5EfF903e615F5A2282Cae03484985a"),
		},
	},
	"HOP": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0xc5102fE9359FD9a28f877a67E36B0F050d81a3CC"),
			L1Bridge:         common.HexToAddress("0x914f986a44AcB623A277d6Bd17368171FCbe4273"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x9D3A7fB18CA7F1237F977Dc5572883f8b24F5638"),
			L2CanonicalBridge:  common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:   common.HexToAddress("0xc5102fE9359FD9a28f877a67E36B0F050d81a3CC"),
			L2Bridge:           common.HexToAddress("0x03D7f750777eC48d39D080b020D83Eb2CB4e3547"),
			L2HopBridgeToken:   common.HexToAddress("0xc5102fE9359FD9a28f877a67E36B0F050d81a3CC"),
			L2AmmWrapper:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2SaddleSwap:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2SaddleLpToken:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
		},
		walletCommon.ArbitrumMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x41BF5Fd5D1C85f00fd1F23C77740F1A7eBa6A35c"),
			L2CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2CanonicalToken:   common.HexToAddress("0xc5102fE9359FD9a28f877a67E36B0F050d81a3CC"),
			L2Bridge:           common.HexToAddress("0x25FB92E505F752F730cAD0Bd4fa17ecE4A384266"),
			L2HopBridgeToken:   common.HexToAddress("0xc5102fE9359FD9a28f877a67E36B0F050d81a3CC"),
			L2AmmWrapper:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2SaddleSwap:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2SaddleLpToken:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
		},
	},
	"SNX": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f"),
			L1Bridge:         common.HexToAddress("0x893246FACF345c99e4235E5A7bbEE7404c988b96"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0xf0727B1eB1A4c9319A5c34A68bcD5E6530850D47"),
			L2CanonicalBridge:  common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:   common.HexToAddress("0x8700dAec35aF8Ff88c16BdF0418774CB3D7599B4"),
			L2Bridge:           common.HexToAddress("0x16284c7323c35F4960540583998C98B1CfC581a7"),
			L2HopBridgeToken:   common.HexToAddress("0x13B7F51BD865410c3AcC4d56083C5B56aB38D203"),
			L2AmmWrapper:       common.HexToAddress("0xf11EBB94EC986EA891Aec29cfF151345C83b33Ec"),
			L2SaddleSwap:       common.HexToAddress("0x1990BC6dfe2ef605Bfc08f5A23564dB75642Ad73"),
			L2SaddleLpToken:    common.HexToAddress("0xe63337211DdE2569C348D9B3A0acb5637CFa8aB3"),
		},
	},
	"sUSD": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0x57Ab1ec28D129707052df4dF418D58a2D46d5f51"),
			L1Bridge:         common.HexToAddress("0x36443fC70E073fe9D50425f82a3eE19feF697d62"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x4Ef4C1208F7374d0252767E3992546d61dCf9848"),
			L2CanonicalBridge:  common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:   common.HexToAddress("0x8c6f28f2F1A3C87F0f938b96d27520d9751ec8d9"),
			L2Bridge:           common.HexToAddress("0x33Fe5bB8DA466dA55a8A32D6ADE2BB104E2C5201"),
			L2HopBridgeToken:   common.HexToAddress("0x6F03052743CD99ce1b29265E377e320CD24Eb632"),
			L2AmmWrapper:       common.HexToAddress("0x29Fba7d2A6C95DB162ee09C6250e912D6893DCa6"),
			L2SaddleSwap:       common.HexToAddress("0x8d4063E82A4Db8CdAed46932E1c71e03CA69Bede"),
			L2SaddleLpToken:    common.HexToAddress("0xBD08972Cef7C9a5A046C9Ef13C9c3CE13739B8d6"),
		},
	},
	"rETH": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0xae78736Cd615f374D3085123A210448E74Fc6393"),
			L1Bridge:         common.HexToAddress("0x87269B23e73305117D0404557bAdc459CEd0dbEc"),
		},
		walletCommon.OptimismMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0xae26bbD1FA3083E1dae3AEaA2050b97c55886f5d"),
			L2CanonicalBridge:  common.HexToAddress("0x4200000000000000000000000000000000000010"),
			L2CanonicalToken:   common.HexToAddress("0x9Bcef72be871e61ED4fBbc7630889beE758eb81D"),
			L2Bridge:           common.HexToAddress("0xA0075E8cE43dcB9970cB7709b9526c1232cc39c2"),
			L2HopBridgeToken:   common.HexToAddress("0x755569159598f3702bdD7DFF6233A317C156d3Dd"),
			L2AmmWrapper:       common.HexToAddress("0x19B2162CA4C2C6F08C6942bFB846ce5C396aCB75"),
			L2SaddleSwap:       common.HexToAddress("0x9Dd8685463285aD5a94D2c128bda3c5e8a6173c8"),
			L2SaddleLpToken:    common.HexToAddress("0x0699BC1Ca03761110929b2B56BcCBeb691fa9ca6"),
		},
		walletCommon.ArbitrumMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0x7fEb7af8d5B277e249868aCF7644e7BB4A5937f8"),
			L2CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2CanonicalToken:   common.HexToAddress("0xEC70Dcb4A1EFa46b8F2D97C310C9c4790ba5ffA8"),
			L2Bridge:           common.HexToAddress("0xc315239cFb05F1E130E7E28E603CEa4C014c57f0"),
			L2HopBridgeToken:   common.HexToAddress("0x588Bae9C85a605a7F14E551d144279984469423B"),
			L2AmmWrapper:       common.HexToAddress("0x16e08C02e4B78B0a5b3A917FF5FeaeDd349a5a95"),
			L2SaddleSwap:       common.HexToAddress("0x0Ded0d521AC7B0d312871D18EA4FDE79f03Ee7CA"),
			L2SaddleLpToken:    common.HexToAddress("0xbBA837dFFB3eCf4638D200F11B8c691eA641AdCb"),
		},
	},
	"MAGIC": {
		walletCommon.EthereumMainnet: {
			L1CanonicalToken: common.HexToAddress("0xB0c7a3Ba49C7a6EaBa6cD4a96C55a1391070Ac9A"),
			L1Bridge:         common.HexToAddress("0xf074540eb83c86211F305E145eB31743E228E57d"),
		},
		walletCommon.ArbitrumMainnet: {
			L1CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L1MessengerWrapper: common.HexToAddress("0xa0c37738582E63B383E609624423d052BFA4b316"),
			L2CanonicalBridge:  common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2CanonicalToken:   common.HexToAddress("0x539bdE0d7Dbd336b79148AA742883198BBF60342"),
			L2Bridge:           common.HexToAddress("0xEa5abf2C909169823d939de377Ef2Bf897A6CE98"),
			L2HopBridgeToken:   common.HexToAddress("0xB76e673EBC922b1E8f10303D0d513a9E710f5c4c"),
			L2AmmWrapper:       common.HexToAddress("0x50a3a623d00fd8b8a4F3CbC5aa53D0Bc6FA912DD"),
			L2SaddleSwap:       common.HexToAddress("0xFFe42d3Ba79Ee5Ee74a999CAd0c60EF1153F0b82"),
			L2SaddleLpToken:    common.HexToAddress("0x163A9E12787dBFa2836caa549aE02ed67F73e7C2"),
		},
	},
}

func isHTokenSend() bool {
	// isHTokenSend is false in Status app for now
	return false
}

func shouldUseCctpBridge(symbol string) bool {
	if symbol == "USDC" {
		return true
	}
	return symbol == "USDC.e" && isHTokenSend()
}

func shouldUseAmm(symbol string) bool {
	return symbol != "HOP"
}

func GetContractAddress(chainID uint64, symbol string) (addr common.Address, contractType string, err error) {
	err = errorNotAvailableOnChainID

	if chainID == walletCommon.EthereumMainnet ||
		chainID == walletCommon.EthereumSepolia {
		if shouldUseCctpBridge(symbol) {
			if addr, ok := hopBridgeContractAddresses[symbol][chainID][CctpL1Bridge]; ok {
				return addr, CctpL1Bridge, nil
			}
			return
		}

		if addr, ok := hopBridgeContractAddresses[symbol][chainID][L1Bridge]; ok {
			return addr, L1Bridge, nil
		}
		return
	}

	if shouldUseCctpBridge(symbol) {
		if addr, ok := hopBridgeContractAddresses[symbol][chainID][CctpL2Bridge]; ok {
			return addr, CctpL2Bridge, nil
		}
		return
	}

	if isHTokenSend() || !shouldUseAmm(symbol) {
		if addr, ok := hopBridgeContractAddresses[symbol][chainID][L2Bridge]; ok {
			return addr, L2Bridge, nil
		}
		return
	}

	if addr, ok := hopBridgeContractAddresses[symbol][chainID][L2AmmWrapper]; ok {
		return addr, L2AmmWrapper, nil
	}
	return
}
