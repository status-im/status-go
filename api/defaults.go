package api

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

const pathWalletRoot = "m/44'/60'/0'/0"
const pathEIP1581 = "m/43'/60'/1581'"
const pathDefaultChat = pathEIP1581 + "/0'/0"
const pathDefaultWallet = pathWalletRoot + "/0"
const defaultMnemonicLength = 12
const shardsTestClusterID = 16
const walletAccountDefaultName = "Account 1"
const keystoreRelativePath = "keystore"
const DefaultKeycardPairingDataFile = "/ethereum/mainnet_rpc/keycard/pairings.json"

const DefaultArchivesRelativePath = "data/archivedata"
const DefaultTorrentTorrentsRelativePath = "data/torrents"

const DefaultDataDir = "/ethereum/mainnet_rpc"
const DefaultNodeName = "StatusIM"
const DefaultLogFile = "geth.log"
const DefaultLogLevel = "ERROR"
const DefaultMaxPeers = 20
const DefaultMaxPendingPeers = 20
const DefaultListenAddr = ":0"
const DefaultMaxMessageDeliveryAttempts = 6
const DefaultVerifyTransactionChainID = 1

var paths = []string{pathWalletRoot, pathEIP1581, pathDefaultChat, pathDefaultWallet}

var DefaultFleet = params.FleetShardsTest

func defaultSettings(generatedAccountInfo generator.GeneratedAccountInfo, derivedAddresses map[string]generator.AccountInfo, mnemonic *string) (*settings.Settings, error) {
	chatKeyString := derivedAddresses[pathDefaultChat].PublicKey

	s := &settings.Settings{}
	s.Mnemonic = &generatedAccountInfo.Mnemonic
	s.BackupEnabled = true
	logLevel := "INFO"
	s.LogLevel = &logLevel
	s.ProfilePicturesShowTo = settings.ProfilePicturesShowToEveryone
	s.ProfilePicturesVisibility = settings.ProfilePicturesVisibilityEveryone
	s.KeyUID = generatedAccountInfo.KeyUID
	s.Address = types.HexToAddress(generatedAccountInfo.Address)
	s.WalletRootAddress = types.HexToAddress(derivedAddresses[pathWalletRoot].Address)
	s.URLUnfurlingMode = settings.URLUnfurlingAlwaysAsk

	// Set chat key & name
	name, err := alias.GenerateFromPublicKeyString(chatKeyString)
	if err != nil {
		return nil, err
	}
	s.Name = name
	s.PublicKey = chatKeyString

	s.DappsAddress = types.HexToAddress(derivedAddresses[pathDefaultWallet].Address)
	s.EIP1581Address = types.HexToAddress(derivedAddresses[pathEIP1581].Address)
	s.Mnemonic = mnemonic

	signingPhrase, err := buildSigningPhrase()
	if err != nil {
		return nil, err
	}
	s.SigningPhrase = signingPhrase

	s.SendPushNotifications = true
	s.InstallationID = uuid.New().String()
	s.UseMailservers = true

	s.PreviewPrivacy = true
	s.PeerSyncingEnabled = false
	s.Currency = "usd"
	s.LinkPreviewRequestEnabled = true

	visibleTokens := make(map[string][]string)
	visibleTokens["mainnet"] = []string{"SNT"}
	visibleTokensJSON, err := json.Marshal(visibleTokens)
	if err != nil {
		return nil, err
	}
	visibleTokenJSONRaw := json.RawMessage(visibleTokensJSON)
	s.WalletVisibleTokens = &visibleTokenJSONRaw

	// TODO: fix this
	networks := make([]map[string]string, 0)
	networksJSON, err := json.Marshal(networks)
	if err != nil {
		return nil, err
	}
	networkRawMessage := json.RawMessage(networksJSON)
	s.Networks = &networkRawMessage
	s.CurrentNetwork = "mainnet_rpc"

	s.TokenGroupByCommunity = false
	s.ShowCommunityAssetWhenSendingTokens = true
	s.DisplayAssetsBelowBalance = false

	s.TestNetworksEnabled = false

	// Default user status
	currentUserStatus, err := json.Marshal(protocol.UserStatus{
		PublicKey:  chatKeyString,
		StatusType: int(protobuf.StatusUpdate_AUTOMATIC),
		Clock:      0,
		CustomText: "",
	})
	if err != nil {
		return nil, err
	}
	userRawMessage := json.RawMessage(currentUserStatus)
	s.CurrentUserStatus = &userRawMessage

	return s, nil
}

func SetDefaultFleet(nodeConfig *params.NodeConfig) error {
	return SetFleet(DefaultFleet, nodeConfig)
}

func SetFleet(fleet string, nodeConfig *params.NodeConfig) error {
	specifiedWakuV2Config := nodeConfig.WakuV2Config
	nodeConfig.WakuV2Config = params.WakuV2Config{
		Enabled:        true,
		EnableDiscV5:   true,
		DiscoveryLimit: 20,
		Host:           "0.0.0.0",
		AutoUpdate:     true,
		// mobile may need override following options
		LightClient: specifiedWakuV2Config.LightClient,
		Nameserver:  specifiedWakuV2Config.Nameserver,
	}

	clusterConfig, err := params.LoadClusterConfigFromFleet(fleet)
	if err != nil {
		return err
	}
	nodeConfig.ClusterConfig = *clusterConfig
	nodeConfig.ClusterConfig.Fleet = fleet
	nodeConfig.ClusterConfig.WakuNodes = params.DefaultWakuNodes(fleet)
	nodeConfig.ClusterConfig.DiscV5BootstrapNodes = params.DefaultDiscV5Nodes(fleet)

	if fleet == params.FleetShardsTest {
		nodeConfig.ClusterConfig.ClusterID = shardsTestClusterID
		nodeConfig.WakuV2Config.UseShardAsDefaultTopic = true
	}

	return nil
}

func buildWalletConfig(request *requests.WalletSecretsConfig) params.WalletConfig {
	walletConfig := params.WalletConfig{
		Enabled:        true,
		AlchemyAPIKeys: make(map[uint64]string),
	}

	if request.OpenseaAPIKey != "" {
		walletConfig.OpenseaAPIKey = request.OpenseaAPIKey
	}

	if request.RaribleMainnetAPIKey != "" {
		walletConfig.RaribleMainnetAPIKey = request.RaribleMainnetAPIKey
	}

	if request.RaribleTestnetAPIKey != "" {
		walletConfig.RaribleTestnetAPIKey = request.RaribleTestnetAPIKey
	}

	if request.InfuraToken != "" {
		walletConfig.InfuraAPIKey = request.InfuraToken
	}

	if request.InfuraSecret != "" {
		walletConfig.InfuraAPIKeySecret = request.InfuraSecret
	}

	if request.AlchemyEthereumMainnetToken != "" {
		walletConfig.AlchemyAPIKeys[mainnetChainID] = request.AlchemyEthereumMainnetToken
	}
	if request.AlchemyEthereumGoerliToken != "" {
		walletConfig.AlchemyAPIKeys[goerliChainID] = request.AlchemyEthereumGoerliToken
	}
	if request.AlchemyEthereumSepoliaToken != "" {
		walletConfig.AlchemyAPIKeys[sepoliaChainID] = request.AlchemyEthereumSepoliaToken
	}
	if request.AlchemyArbitrumMainnetToken != "" {
		walletConfig.AlchemyAPIKeys[arbitrumChainID] = request.AlchemyArbitrumMainnetToken
	}
	if request.AlchemyArbitrumGoerliToken != "" {
		walletConfig.AlchemyAPIKeys[arbitrumGoerliChainID] = request.AlchemyArbitrumGoerliToken
	}
	if request.AlchemyArbitrumSepoliaToken != "" {
		walletConfig.AlchemyAPIKeys[arbitrumSepoliaChainID] = request.AlchemyArbitrumSepoliaToken
	}
	if request.AlchemyOptimismMainnetToken != "" {
		walletConfig.AlchemyAPIKeys[optimismChainID] = request.AlchemyOptimismMainnetToken
	}
	if request.AlchemyOptimismGoerliToken != "" {
		walletConfig.AlchemyAPIKeys[optimismGoerliChainID] = request.AlchemyOptimismGoerliToken
	}
	if request.AlchemyOptimismSepoliaToken != "" {
		walletConfig.AlchemyAPIKeys[optimismSepoliaChainID] = request.AlchemyOptimismSepoliaToken
	}

	return walletConfig
}

func defaultNodeConfig(installationID string, request *requests.CreateAccount, opts ...params.Option) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.LogEnabled = request.LogEnabled
	nodeConfig.LogFile = DefaultLogFile
	nodeConfig.LogDir = request.LogFilePath
	nodeConfig.LogLevel = DefaultLogLevel
	nodeConfig.DataDir = DefaultDataDir
	nodeConfig.KeycardPairingDataFile = DefaultKeycardPairingDataFile
	nodeConfig.ProcessBackedupMessages = false

	if request.LogLevel != nil {
		nodeConfig.LogLevel = *request.LogLevel
		nodeConfig.LogEnabled = true
	} else {
		nodeConfig.LogEnabled = false
	}

	if request.TestOverrideNetworks != nil {
		nodeConfig.Networks = request.TestOverrideNetworks
	} else {
		nodeConfig.Networks = BuildDefaultNetworks(&request.WalletSecretsConfig)
	}

	if request.NetworkID != nil {
		nodeConfig.NetworkID = *request.NetworkID
	} else {
		nodeConfig.NetworkID = nodeConfig.Networks[0].ChainID
	}

	if request.UpstreamConfig != "" {
		nodeConfig.UpstreamConfig = params.UpstreamRPCConfig{
			Enabled: true,
			URL:     request.UpstreamConfig,
		}
	} else {
		nodeConfig.UpstreamConfig.URL = defaultNetworks[0].RPCURL
		nodeConfig.UpstreamConfig.Enabled = true
	}

	nodeConfig.Name = DefaultNodeName
	nodeConfig.Rendezvous = false
	nodeConfig.NoDiscovery = true
	nodeConfig.MaxPeers = DefaultMaxPeers
	nodeConfig.MaxPendingPeers = DefaultMaxPendingPeers

	nodeConfig.WalletConfig = buildWalletConfig(&request.WalletSecretsConfig)

	nodeConfig.LocalNotificationsConfig = params.LocalNotificationsConfig{Enabled: true}
	nodeConfig.BrowsersConfig = params.BrowsersConfig{Enabled: true}
	nodeConfig.PermissionsConfig = params.PermissionsConfig{Enabled: true}
	nodeConfig.MailserversConfig = params.MailserversConfig{Enabled: true}

	nodeConfig.ListenAddr = DefaultListenAddr

	err := SetDefaultFleet(nodeConfig)
	if err != nil {
		return nil, err
	}

	if request.WakuV2LightClient {
		nodeConfig.WakuV2Config.LightClient = true
	}

	if request.WakuV2Nameserver != nil {
		nodeConfig.WakuV2Config.Nameserver = *request.WakuV2Nameserver
	}

	if request.TelemetryServerURL != "" {
		nodeConfig.WakuV2Config.TelemetryServerURL = request.TelemetryServerURL
	}

	nodeConfig.ShhextConfig = params.ShhextConfig{
		BackupDisabledDataDir:      "./",
		InstallationID:             installationID,
		MaxMessageDeliveryAttempts: DefaultMaxMessageDeliveryAttempts,
		MailServerConfirmations:    true,
		VerifyTransactionChainID:   DefaultVerifyTransactionChainID,
		DataSyncEnabled:            true,
		PFSEnabled:                 true,
	}

	if request.VerifyTransactionURL != nil {
		nodeConfig.ShhextConfig.VerifyTransactionURL = *request.VerifyTransactionURL
	} else {
		nodeConfig.ShhextConfig.VerifyTransactionURL = defaultNetworks[0].FallbackURL
	}

	if request.VerifyENSURL != nil {
		nodeConfig.ShhextConfig.VerifyENSURL = *request.VerifyENSURL
	} else {
		nodeConfig.ShhextConfig.VerifyENSURL = defaultNetworks[0].FallbackURL
	}

	if request.VerifyTransactionChainID != nil {
		nodeConfig.ShhextConfig.VerifyTransactionChainID = *request.VerifyTransactionChainID
	}

	if request.VerifyENSContractAddress != nil {
		nodeConfig.ShhextConfig.VerifyENSContractAddress = *request.VerifyENSContractAddress
	}

	if request.NetworkID != nil {
		nodeConfig.NetworkID = *request.NetworkID
	}

	nodeConfig.TorrentConfig = params.TorrentConfig{
		Enabled:    false,
		Port:       0,
		DataDir:    filepath.Join(nodeConfig.RootDataDir, DefaultArchivesRelativePath),
		TorrentDir: filepath.Join(nodeConfig.RootDataDir, DefaultTorrentTorrentsRelativePath),
	}

	if request.TorrentConfigEnabled != nil {
		nodeConfig.TorrentConfig.Enabled = *request.TorrentConfigEnabled

	}
	if request.TorrentConfigPort != nil {
		nodeConfig.TorrentConfig.Port = *request.TorrentConfigPort
	}

	if request.APIConfig != nil {
		nodeConfig.HTTPEnabled = true
		nodeConfig.HTTPHost = request.APIConfig.HTTPHost
		nodeConfig.HTTPPort = request.APIConfig.HTTPPort
		nodeConfig.APIModules = request.APIConfig.APIModules
	}

	for _, opt := range opts {
		if err := opt(nodeConfig); err != nil {
			return nil, err
		}
	}

	return nodeConfig, nil
}

func buildSigningPhrase() (string, error) {
	length := big.NewInt(int64(len(dictionary)))
	a, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}
	b, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}
	c, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}

	return dictionary[a.Int64()] + " " + dictionary[b.Int64()] + " " + dictionary[c.Int64()], nil

}

var dictionary = []string{
	"acid",
	"alto",
	"apse",
	"arch",
	"area",
	"army",
	"atom",
	"aunt",
	"babe",
	"baby",
	"back",
	"bail",
	"bait",
	"bake",
	"ball",
	"band",
	"bank",
	"barn",
	"base",
	"bass",
	"bath",
	"bead",
	"beak",
	"beam",
	"bean",
	"bear",
	"beat",
	"beef",
	"beer",
	"beet",
	"bell",
	"belt",
	"bend",
	"bike",
	"bill",
	"bird",
	"bite",
	"blow",
	"blue",
	"boar",
	"boat",
	"body",
	"bolt",
	"bomb",
	"bone",
	"book",
	"boot",
	"bore",
	"boss",
	"bowl",
	"brow",
	"bulb",
	"bull",
	"burn",
	"bush",
	"bust",
	"cafe",
	"cake",
	"calf",
	"call",
	"calm",
	"camp",
	"cane",
	"cape",
	"card",
	"care",
	"carp",
	"cart",
	"case",
	"cash",
	"cast",
	"cave",
	"cell",
	"cent",
	"chap",
	"chef",
	"chin",
	"chip",
	"chop",
	"chub",
	"chug",
	"city",
	"clam",
	"clef",
	"clip",
	"club",
	"clue",
	"coal",
	"coat",
	"code",
	"coil",
	"coin",
	"coke",
	"cold",
	"colt",
	"comb",
	"cone",
	"cook",
	"cope",
	"copy",
	"cord",
	"cork",
	"corn",
	"cost",
	"crab",
	"craw",
	"crew",
	"crib",
	"crop",
	"crow",
	"curl",
	"cyst",
	"dame",
	"dare",
	"dark",
	"dart",
	"dash",
	"data",
	"date",
	"dead",
	"deal",
	"dear",
	"debt",
	"deck",
	"deep",
	"deer",
	"desk",
	"dhow",
	"diet",
	"dill",
	"dime",
	"dirt",
	"dish",
	"disk",
	"dock",
	"doll",
	"door",
	"dory",
	"drag",
	"draw",
	"drop",
	"drug",
	"drum",
	"duck",
	"dump",
	"dust",
	"duty",
	"ease",
	"east",
	"eave",
	"eddy",
	"edge",
	"envy",
	"epee",
	"exam",
	"exit",
	"face",
	"fact",
	"fail",
	"fall",
	"fame",
	"fang",
	"farm",
	"fawn",
	"fear",
	"feed",
	"feel",
	"feet",
	"file",
	"fill",
	"film",
	"find",
	"fine",
	"fire",
	"fish",
	"flag",
	"flat",
	"flax",
	"flow",
	"foam",
	"fold",
	"font",
	"food",
	"foot",
	"fork",
	"form",
	"fort",
	"fowl",
	"frog",
	"fuel",
	"full",
	"gain",
	"gale",
	"galn",
	"game",
	"garb",
	"gate",
	"gear",
	"gene",
	"gift",
	"girl",
	"give",
	"glad",
	"glen",
	"glue",
	"glut",
	"goal",
	"goat",
	"gold",
	"golf",
	"gong",
	"good",
	"gown",
	"grab",
	"gram",
	"gray",
	"grey",
	"grip",
	"grit",
	"gyro",
	"hail",
	"hair",
	"half",
	"hall",
	"hand",
	"hang",
	"harm",
	"harp",
	"hate",
	"hawk",
	"head",
	"heat",
	"heel",
	"hell",
	"helo",
	"help",
	"hemp",
	"herb",
	"hide",
	"high",
	"hill",
	"hire",
	"hive",
	"hold",
	"hole",
	"home",
	"hood",
	"hoof",
	"hook",
	"hope",
	"hops",
	"horn",
	"hose",
	"host",
	"hour",
	"hunt",
	"hurt",
	"icon",
	"idea",
	"inch",
	"iris",
	"iron",
	"item",
	"jail",
	"jeep",
	"jeff",
	"joey",
	"join",
	"joke",
	"judo",
	"jump",
	"junk",
	"jury",
	"jute",
	"kale",
	"keep",
	"kick",
	"kill",
	"kilt",
	"kind",
	"king",
	"kiss",
	"kite",
	"knee",
	"knot",
	"lace",
	"lack",
	"lady",
	"lake",
	"lamb",
	"lamp",
	"land",
	"lark",
	"lava",
	"lawn",
	"lead",
	"leaf",
	"leek",
	"lier",
	"life",
	"lift",
	"lily",
	"limo",
	"line",
	"link",
	"lion",
	"lisa",
	"list",
	"load",
	"loaf",
	"loan",
	"lock",
	"loft",
	"long",
	"look",
	"loss",
	"lout",
	"love",
	"luck",
	"lung",
	"lute",
	"lynx",
	"lyre",
	"maid",
	"mail",
	"main",
	"make",
	"male",
	"mall",
	"manx",
	"many",
	"mare",
	"mark",
	"mask",
	"mass",
	"mate",
	"math",
	"meal",
	"meat",
	"meet",
	"menu",
	"mess",
	"mice",
	"midi",
	"mile",
	"milk",
	"mime",
	"mind",
	"mine",
	"mini",
	"mint",
	"miss",
	"mist",
	"moat",
	"mode",
	"mole",
	"mood",
	"moon",
	"most",
	"moth",
	"move",
	"mule",
	"mutt",
	"nail",
	"name",
	"neat",
	"neck",
	"need",
	"neon",
	"nest",
	"news",
	"node",
	"nose",
	"note",
	"oboe",
	"okra",
	"open",
	"oval",
	"oven",
	"oxen",
	"pace",
	"pack",
	"page",
	"pail",
	"pain",
	"pair",
	"palm",
	"pard",
	"park",
	"part",
	"pass",
	"past",
	"path",
	"peak",
	"pear",
	"peen",
	"peer",
	"pelt",
	"perp",
	"pest",
	"pick",
	"pier",
	"pike",
	"pile",
	"pimp",
	"pine",
	"ping",
	"pink",
	"pint",
	"pipe",
	"piss",
	"pith",
	"plan",
	"play",
	"plot",
	"plow",
	"poem",
	"poet",
	"pole",
	"polo",
	"pond",
	"pony",
	"poof",
	"pool",
	"port",
	"post",
	"prow",
	"pull",
	"puma",
	"pump",
	"pupa",
	"push",
	"quit",
	"race",
	"rack",
	"raft",
	"rage",
	"rail",
	"rain",
	"rake",
	"rank",
	"rate",
	"read",
	"rear",
	"reef",
	"rent",
	"rest",
	"rice",
	"rich",
	"ride",
	"ring",
	"rise",
	"risk",
	"road",
	"robe",
	"rock",
	"role",
	"roll",
	"roof",
	"room",
	"root",
	"rope",
	"rose",
	"ruin",
	"rule",
	"rush",
	"ruth",
	"sack",
	"safe",
	"sage",
	"sail",
	"sale",
	"salt",
	"sand",
	"sari",
	"sash",
	"save",
	"scow",
	"seal",
	"seat",
	"seed",
	"self",
	"sell",
	"shed",
	"shin",
	"ship",
	"shoe",
	"shop",
	"shot",
	"show",
	"sick",
	"side",
	"sign",
	"silk",
	"sill",
	"silo",
	"sing",
	"sink",
	"site",
	"size",
	"skin",
	"sled",
	"slip",
	"smog",
	"snob",
	"snow",
	"soap",
	"sock",
	"soda",
	"sofa",
	"soft",
	"soil",
	"song",
	"soot",
	"sort",
	"soup",
	"spot",
	"spur",
	"stag",
	"star",
	"stay",
	"stem",
	"step",
	"stew",
	"stop",
	"stud",
	"suck",
	"suit",
	"swan",
	"swim",
	"tail",
	"tale",
	"talk",
	"tank",
	"tard",
	"task",
	"taxi",
	"team",
	"tear",
	"teen",
	"tell",
	"temp",
	"tent",
	"term",
	"test",
	"text",
	"thaw",
	"tile",
	"till",
	"time",
	"tire",
	"toad",
	"toga",
	"togs",
	"tone",
	"tool",
	"toot",
	"tote",
	"tour",
	"town",
	"tram",
	"tray",
	"tree",
	"trim",
	"trip",
	"tuba",
	"tube",
	"tuna",
	"tune",
	"turn",
	"tutu",
	"twig",
	"type",
	"unit",
	"user",
	"vane",
	"vase",
	"vast",
	"veal",
	"veil",
	"vein",
	"vest",
	"vibe",
	"view",
	"vise",
	"wait",
	"wake",
	"walk",
	"wall",
	"wash",
	"wasp",
	"wave",
	"wear",
	"weed",
	"week",
	"well",
	"west",
	"whip",
	"wife",
	"will",
	"wind",
	"wine",
	"wing",
	"wire",
	"wish",
	"wolf",
	"wood",
	"wool",
	"word",
	"work",
	"worm",
	"wrap",
	"wren",
	"yard",
	"yarn",
	"yawl",
	"year",
	"yoga",
	"yoke",
	"yurt",
	"zinc",
	"zone",
}
