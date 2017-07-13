package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/whisper/mailserver"
	"github.com/ethereum/go-ethereum/whisper/notifications"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

// node-related errors
var (
	ErrEthServiceRegistrationFailure     = errors.New("failed to register the Ethereum service")
	ErrWhisperServiceRegistrationFailure = errors.New("failed to register the Whisper service")
	ErrLightEthRegistrationFailure       = errors.New("failed to register the LES service")
	ErrNodeMakeFailure                   = errors.New("error creating p2p node")
	ErrNodeRunFailure                    = errors.New("error running p2p node")
	ErrNodeStartFailure                  = errors.New("error starting p2p node")
)

// MakeNode create a geth node entity
func MakeNode(config *params.NodeConfig) (*node.Node, error) {
	// make sure data directory exists
	if err := os.MkdirAll(filepath.Join(config.DataDir), os.ModePerm); err != nil {
		return nil, err
	}

	// make sure keys directory exists
	if err := os.MkdirAll(filepath.Join(config.KeyStoreDir), os.ModePerm); err != nil {
		return nil, err
	}

	// setup logging
	if _, err := common.SetupLogger(config); err != nil {
		return nil, err
	}

	// configure required node (should you need to update node's config, e.g. add bootstrap nodes, see node.Config)
	stackConfig := defaultEmbeddedNodeConfig(config)

	if len(config.NodeKeyFile) > 0 {
		log.Info("Loading private key file", "file", config.NodeKeyFile)
		pk, err := crypto.LoadECDSA(config.NodeKeyFile)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed loading private key file '%s': %v", config.NodeKeyFile, err))
		}

		// override node's private key
		stackConfig.P2P.PrivateKey = pk
	}

	if len(config.NodeKeyFile) > 0 {
		log.Info("Loading private key file", "file", config.NodeKeyFile)
		pk, err := crypto.LoadECDSA(config.NodeKeyFile)
		if err != nil {
			log.Info("Failed loading private key file", "file", config.NodeKeyFile, "err", err)
		}

		// override node's private key
		stackConfig.P2P.PrivateKey = pk
	}

	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, ErrNodeMakeFailure
	}

	// start Ethereum service
	if err := activateEthService(stack, config); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrEthServiceRegistrationFailure, err)
	}

	// start Whisper service
	if err := activateShhService(stack, config); err != nil {
		return nil, fmt.Errorf("%v: %v", ErrWhisperServiceRegistrationFailure, err)
	}

	return stack, nil
}

// defaultEmbeddedNodeConfig returns default stack configuration for mobile client node
func defaultEmbeddedNodeConfig(config *params.NodeConfig) *node.Config {
	nc := &node.Config{
		DataDir:           config.DataDir,
		KeyStoreDir:       config.KeyStoreDir,
		UseLightweightKDF: true,
		NoUSB:             true,
		Name:              config.Name,
		Version:           config.Version,
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			DiscoveryV5Addr:  ":0",
			BootstrapNodes:   makeBootstrapNodes(),
			BootstrapNodesV5: makeBootstrapNodesV5(),
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
			MaxPendingPeers:  config.MaxPendingPeers,
		},
		IPCPath:     makeIPCPath(config),
		HTTPCors:    []string{"*"},
		HTTPModules: strings.Split(config.APIModules, ","),
		WSHost:      makeWSHost(config),
		WSPort:      config.WSPort,
		WSOrigins:   []string{"*"},
		WSModules:   strings.Split(config.APIModules, ","),
	}

	if config.RPCEnabled {
		nc.HTTPHost = config.HTTPHost
		nc.HTTPPort = config.HTTPPort
	}

	return nc
}

// updateCHT changes trusted canonical hash trie root
func updateCHT(eth *les.LightEthereum, config *params.NodeConfig) {
	bc := eth.BlockChain()

	// TODO: Remove this thing as this is an ugly hack.
	// Once CHT sync sub-protocol is working in LES, we will rely on it, as it provides
	// decentralized solution. For now, in order to avoid forcing users to long sync times
	// we use central static resource
	type MsgCHTRoot struct {
		GenesisHash string `json:"net"`
		Number      uint64 `json:"number"`
		Prod        string `json:"prod"`
		Dev         string `json:"dev"`
	}
	loadCHTLists := func() ([]MsgCHTRoot, error) {
		url := config.LightEthConfig.CHTRootConfigURL + "?u=" + strconv.Itoa(int(time.Now().Unix()))
		client := &http.Client{Timeout: 5 * time.Second}
		r, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		var roots []MsgCHTRoot
		err = json.NewDecoder(r.Body).Decode(&roots)
		if err != nil {
			return nil, err
		}

		return roots, nil
	}
	if roots, err := loadCHTLists(); err == nil {
		for _, root := range roots {
			if bc.Genesis().Hash().Hex() == root.GenesisHash {
				log.Info("Loaded root", "root", root)
				if root.Number == 0 {
					continue
				}

				chtRoot := root.Prod
				if config.DevMode {
					chtRoot = root.Dev
				}
				eth.WriteTrustedCht(light.TrustedCht{
					Number: root.Number,
					Root:   gethcommon.HexToHash(chtRoot),
				})
				log.Info("Loaded CHT from net", "CHT", chtRoot, "number", root.Number, "dev", config.DevMode)
				return
			}
		}
	}

	// resort to manually updated
	log.Info("Loading CHT from net failed, setting manually")
	if bc.Genesis().Hash() == params.MainNetGenesisHash {
		eth.WriteTrustedCht(light.TrustedCht{
			Number: 805,
			Root:   gethcommon.HexToHash("85e4286fe0a730390245c49de8476977afdae0eb5530b277f62a52b12313d50f"),
		})
		log.Info("Added trusted CHT for mainnet")
	}

	if bc.Genesis().Hash() == params.RopstenNetGenesisHash {
		root := "fa851b5252cc48ab55f375833b0344cc5c7cacea69be7e2a57976c38d3bb3aef"
		if config.DevMode {
			root = "f2f862314509b22a773eedaaa7fa6452474eb71a3b72525a97dbf5060cbea88f"
		}
		eth.WriteTrustedCht(light.TrustedCht{
			Number: 239,
			Root:   gethcommon.HexToHash(root),
		})
		log.Info("Added trusted CHT for Ropsten", "CHT", root)
	}

	//if bc.Genesis().Hash() == params.RinkebyNetGenesisHash {
	//	root := "0xb100882d00a09292f15e712707649b24d019a47a509e83a00530ac542425c3bd"
	//	if config.DevMode {
	//		root = "0xb100882d00a09292f15e712707649b24d019a47a509e83a00530ac542425c3bd"
	//	}
	//	eth.WriteTrustedCht(light.TrustedCht{
	//		Number: 55,
	//		Root:   gethcommon.HexToHash(root),
	//	})
	//	log.Info("Added trusted CHT for Rinkeby", "CHT", root)
	//}
}

// activateEthService configures and registers the eth.Ethereum service with a given node.
func activateEthService(stack *node.Node, config *params.NodeConfig) error {
	if !config.LightEthConfig.Enabled {
		log.Info("LES protocol is disabled")
		return nil
	}

	var genesis *core.Genesis
	if config.LightEthConfig.Genesis != "" {
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.LightEthConfig.Genesis), genesis); err != nil {
			return fmt.Errorf("invalid genesis spec: %v", err)
		}
	}

	ethConf := eth.DefaultConfig
	ethConf.Genesis = genesis
	ethConf.SyncMode = downloader.LightSync
	ethConf.NetworkId = config.NetworkID
	ethConf.DatabaseCache = config.LightEthConfig.DatabaseCache
	ethConf.MaxPeers = config.MaxPeers
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		lightEth, err := les.New(ctx, &ethConf)
		if err == nil {
			updateCHT(lightEth, config)
		}
		return lightEth, err
	}); err != nil {
		return fmt.Errorf("%v: %v", ErrLightEthRegistrationFailure, err)
	}

	return nil
}

// activateShhService configures Whisper and adds it to the given node.
func activateShhService(stack *node.Node, config *params.NodeConfig) error {
	if !config.WhisperConfig.Enabled {
		log.Info("SHH protocol is disabled")
		return nil
	}

	serviceConstructor := func(*node.ServiceContext) (node.Service, error) {
		whisperConfig := config.WhisperConfig
		whisperService := whisper.New()

		// enable mail service
		if whisperConfig.MailServerNode {
			password, err := whisperConfig.ReadPasswordFile()
			if err != nil {
				return nil, err
			}

			var mailServer mailserver.WMailServer
			whisperService.RegisterServer(&mailServer)
			mailServer.Init(whisperService, whisperConfig.DataDir, string(password), whisperConfig.MinimumPoW)
		}

		// enable notification service
		if whisperConfig.NotificationServerNode {
			var notificationServer notifications.NotificationServer
			whisperService.RegisterNotificationServer(&notificationServer)

			notificationServer.Init(whisperService, whisperConfig)
		}

		return whisperService, nil
	}

	return stack.Register(serviceConstructor)
}

// makeIPCPath returns IPC-RPC filename
func makeIPCPath(config *params.NodeConfig) string {
	if !config.IPCEnabled {
		return ""
	}

	return path.Join(config.DataDir, config.IPCFile)
}

// makeWSHost returns WS-RPC Server host, given enabled/disabled flag
func makeWSHost(config *params.NodeConfig) string {
	if !config.WSEnabled {
		return ""
	}

	return config.WSHost
}

// makeBootstrapNodes returns default (hence bootstrap) list of peers
func makeBootstrapNodes() []*discover.Node {
	// on desktops params.TestnetBootnodes and params.MainBootnodes,
	// on mobile client we deliberately keep this list empty
	enodes := []string{}

	var bootstapNodes []*discover.Node
	for _, enode := range enodes {
		bootstapNodes = append(bootstapNodes, discover.MustParseNode(enode))
	}

	return bootstapNodes
}

// makeBootstrapNodesV5 returns default (hence bootstrap) list of peers
func makeBootstrapNodesV5() []*discv5.Node {
	enodes := gethparams.DiscoveryV5Bootnodes

	var bootstapNodes []*discv5.Node
	for _, enode := range enodes {
		bootstapNodes = append(bootstapNodes, discv5.MustParseNode(enode))
	}

	return bootstapNodes
}
