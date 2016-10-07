package geth

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent( const char *jsonEvent );
*/
import "C"
import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/whisper"
	"gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "Geth"     // Client identifier to advertise over the network
	versionMajor     = 1          // Major version component of the current release
	versionMinor     = 5          // Minor version component of the current release
	versionPatch     = 0          // Patch version component of the current release
	versionMeta      = "unstable" // Version metadata to append to the version string

	RPCPort = 8545 // RPC port (replaced in unit tests)

	EventNodeStarted = "node.started"
)

var (
	ErrDataDirPreprocessingFailed  = errors.New("failed to pre-process data directory")
	ErrInvalidGethNode             = errors.New("no running geth node detected")
	ErrInvalidAccountManager       = errors.New("could not retrieve account manager")
	ErrInvalidWhisperService       = errors.New("whisper service is unavailable")
	ErrInvalidLightEthereumService = errors.New("can not retrieve LES service")
	ErrInvalidClient               = errors.New("RPC client is not properly initialized")
	ErrInvalidJailedRequestQueue   = errors.New("Jailed request queue is not properly initialized")
	ErrNodeStartFailure            = errors.New("could not create the in-memory node object")
)

type SelectedExtKey struct {
	Address     common.Address
	AccountKey  *accounts.Key
	SubAccounts []accounts.Account
}

type NodeManager struct {
	currentNode        *node.Node                // currently running geth node
	ctx                *cli.Context              // the CLI context used to start the geth node
	lightEthereum      *les.LightEthereum        // LES service
	accountManager     *accounts.Manager         // the account manager attached to the currentNode
	jailedRequestQueue *JailedRequestQueue       // bridge via which jail notifies node of incoming requests
	SelectedAccount    *SelectedExtKey           // account that was processed during the last call to SelectAccount()
	whisperService     *whisper.Whisper          // Whisper service
	client             *rpc.ClientRestartWrapper // RPC client
	nodeStarted        chan struct{}             // channel to wait for node to start
}

var (
	nodeManagerInstance *NodeManager
	createOnce          sync.Once
)

func NewNodeManager(datadir string, rpcport int) *NodeManager {
	createOnce.Do(func() {
		nodeManagerInstance = &NodeManager{
			jailedRequestQueue: NewJailedRequestsQueue(),
		}
		nodeManagerInstance.MakeNode(datadir, rpcport)
	})

	return nodeManagerInstance
}

func GetNodeManager() *NodeManager {
	return nodeManagerInstance
}

// createAndStartNode creates a node entity and starts the
// node running locally exposing given RPC port
func CreateAndRunNode(datadir string, rpcport int) error {
	nodeManager := NewNodeManager(datadir, rpcport)

	if nodeManager.HasNode() {
		nodeManager.RunNode()

		<-nodeManager.nodeStarted // block until node is ready
		return nil
	}

	return ErrNodeStartFailure
}

// MakeNode create a geth node entity
func (m *NodeManager) MakeNode(datadir string, rpcport int) *node.Node {
	// TODO remove admin rpcapi flag
	set := flag.NewFlagSet("test", 0)
	set.Bool("lightkdf", true, "Reduce key-derivation RAM & CPU usage at some expense of KDF strength")
	set.Bool("shh", true, "whisper")
	set.Bool("light", true, "disable eth")
	set.Bool("testnet", true, "light test network")
	set.Bool("rpc", true, "enable rpc")
	set.String("rpcaddr", "localhost", "host for RPC")
	set.Int("rpcport", rpcport, "rpc port")
	set.String("rpccorsdomain", "*", "allow all domains")
	set.String("verbosity", "3", "verbosity level")
	set.String("rpcapi", "db,eth,net,web3,shh,personal,admin", "rpc api(s)")
	set.String("datadir", datadir, "data directory for geth")
	set.String("logdir", datadir, "log dir for glog")
	m.ctx = cli.NewContext(nil, set, nil)

	utils.DebugSetup(m.ctx)

	// create node and start requested protocols
	m.currentNode = utils.MakeNode(m.ctx, clientIdentifier, "")
	utils.RegisterEthService(m.ctx, m.currentNode, makeDefaultExtra())

	// Whisper must be explicitly enabled, but is auto-enabled in --dev mode.
	shhEnabled := m.ctx.GlobalBool(utils.WhisperEnabledFlag.Name)
	shhAutoEnabled := !m.ctx.GlobalIsSet(utils.WhisperEnabledFlag.Name) && m.ctx.GlobalIsSet(utils.DevModeFlag.Name)
	if shhEnabled || shhAutoEnabled {
		utils.RegisterShhService(m.currentNode)
	}

	m.accountManager = m.currentNode.AccountManager()
	m.nodeStarted = make(chan struct{})

	return m.currentNode
}

// StartNode starts a geth node entity
func (m *NodeManager) RunNode() {
	go func() {
		utils.StartNode(m.currentNode)

		if m.currentNode.AccountManager() == nil {
			glog.V(logger.Warn).Infoln("cannot get account manager")
		}
		if err := m.currentNode.Service(&m.whisperService); err != nil {
			glog.V(logger.Warn).Infoln("cannot get whisper service:", err)
		}
		if err := m.currentNode.Service(&m.lightEthereum); err != nil {
			glog.V(logger.Warn).Infoln("cannot get light ethereum service:", err)
		}

		// setup handlers
		m.lightEthereum.StatusBackend.SetTransactionQueueHandler(onSendTransactionRequest)
		m.lightEthereum.StatusBackend.SetAccountsFilterHandler(onAccountsListRequest)

		m.client = rpc.NewClientRestartWrapper(func() *rpc.Client {
			client, err := m.currentNode.Attach()
			if err != nil {
				return nil
			}
			return client
		})

		// @TODO Remove after LES supports discover out of box
		m.populateStaticPeers()

		m.onNodeStarted() // node started, notify listeners
		m.currentNode.Wait()
	}()
}

func (m *NodeManager) onNodeStarted() {
	// notify local listener
	m.nodeStarted <- struct{}{}
	close(m.nodeStarted)

	// send signal up to native app
	event := GethEvent{
		Type:  EventNodeStarted,
		Event: struct{}{},
	}

	body, _ := json.Marshal(&event)
	C.StatusServiceSignalEvent(C.CString(string(body)))
}

func (m *NodeManager) AddPeer(url string) (bool, error) {
	if m == nil || !m.HasNode() {
		return false, ErrInvalidGethNode
	}

	server := m.currentNode.Server()
	if server == nil {
		return false, errors.New("node not started")
	}
	// Try to add the url as a static peer and return
	parsedNode, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.AddPeer(parsedNode)

	return true, nil
}

func (m *NodeManager) HasNode() bool {
	return m != nil && m.currentNode != nil
}

func (m *NodeManager) HasAccountManager() bool {
	return m.accountManager != nil
}

func (m *NodeManager) AccountManager() (*accounts.Manager, error) {
	if m == nil || !m.HasNode() {
		return nil, ErrInvalidGethNode
	}

	if !m.HasAccountManager() {
		return nil, ErrInvalidAccountManager
	}

	return m.accountManager, nil
}

func (m *NodeManager) HasWhisperService() bool {
	return m.whisperService != nil
}

func (m *NodeManager) WhisperService() (*whisper.Whisper, error) {
	if m == nil || !m.HasNode() {
		return nil, ErrInvalidGethNode
	}

	if !m.HasWhisperService() {
		return nil, ErrInvalidWhisperService
	}

	return m.whisperService, nil
}

func (m *NodeManager) HasLightEthereumService() bool {
	return m.lightEthereum != nil
}

func (m *NodeManager) LightEthereumService() (*les.LightEthereum, error) {
	if m == nil || !m.HasNode() {
		return nil, ErrInvalidGethNode
	}

	if !m.HasLightEthereumService() {
		return nil, ErrInvalidLightEthereumService
	}

	return m.lightEthereum, nil
}

func (m *NodeManager) HasClientRestartWrapper() bool {
	return m.client != nil
}

func (m *NodeManager) ClientRestartWrapper() (*rpc.ClientRestartWrapper, error) {
	if m == nil || !m.HasNode() {
		return nil, ErrInvalidGethNode
	}

	if !m.HasClientRestartWrapper() {
		return nil, ErrInvalidClient
	}

	return m.client, nil
}

func (m *NodeManager) HasJailedRequestQueue() bool {
	return m.jailedRequestQueue != nil
}

func (m *NodeManager) JailedRequestQueue() (*JailedRequestQueue, error) {
	if m == nil || !m.HasNode() {
		return nil, ErrInvalidGethNode
	}

	if !m.HasJailedRequestQueue() {
		return nil, ErrInvalidJailedRequestQueue
	}

	return m.jailedRequestQueue, nil
}

func makeDefaultExtra() []byte {
	var clientInfo = struct {
		Version   uint
		Name      string
		GoVersion string
		Os        string
	}{uint(versionMajor<<16 | versionMinor<<8 | versionPatch), clientIdentifier, runtime.Version(), runtime.GOOS}
	extra, err := rlp.EncodeToBytes(clientInfo)
	if err != nil {
		glog.V(logger.Warn).Infoln("error setting canonical miner information:", err)
	}

	if uint64(len(extra)) > params.MaximumExtraDataSize.Uint64() {
		glog.V(logger.Warn).Infoln("error setting canonical miner information: extra exceeds", params.MaximumExtraDataSize)
		glog.V(logger.Debug).Infof("extra: %x\n", extra)
		return nil
	}

	return extra
}

func (m *NodeManager) populateStaticPeers() {
	// manually add static nodes (LES auto-discovery is not stable yet)
	configFile, err := ioutil.ReadFile(filepath.Join("../data", "static-nodes.json"))
	if err != nil {
		return
	}
	var enodes []string
	if err = json.Unmarshal(configFile, &enodes); err != nil {
		return
	}

	for _, enode := range enodes {
		m.AddPeer(enode)
	}
}
