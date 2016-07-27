package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/release"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/whisper"
	"gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "Geth"     // Client identifier to advertise over the network
	versionMajor     = 1          // Major version component of the current release
	versionMinor     = 5          // Minor version component of the current release
	versionPatch     = 0          // Patch version component of the current release
	versionMeta      = "unstable" // Version metadata to append to the version string

	versionOracle = "0xfa7b9770ca4cb04296cac84f37736d4041251cdf" // Ethereum address of the Geth release oracle
)

var (
	vString        string             // Combined textual representation of the version
	rConfig        release.Config     // Structured version information and release oracle config
	currentNode    *node.Node         // currently running geth node
	c              *cli.Context       // the CLI context used to start the geth node
	accountSync    *[]node.Service    // the object used to sync accounts between geth services
	lightEthereum  *les.LightEthereum // LES service
	accountManager *accounts.Manager  // the account manager attached to the currentNode
	whisperService *whisper.Whisper   // whisper service
	datadir        string             // data directory for geth
)

func main() {

	// Placeholder for anything we want to run by default
	fmt.Println("You are running statusgo!")

}

// MakeNode create a geth node entity
func MakeNode(inputDir string) *node.Node {

	datadir = inputDir

	// TODO remove admin rpcapi flag
	set := flag.NewFlagSet("test", 0)
	set.Bool("lightkdf", true, "Reduce key-derivation RAM & CPU usage at some expense of KDF strength")
	set.Bool("shh", true, "whisper")
	set.Bool("light", true, "disable eth")
	set.Bool("testnet", true, "light test network")
	set.Bool("rpc", true, "enable rpc")
	set.String("rpcaddr", "localhost", "host for RPC")
	set.String("rpcport", "8545", "rpc port")
	set.String("verbosity", "3", "verbosity level")
	set.String("rpcapi", "db,eth,net,web3,shh,personal,admin", "rpc api(s)")
	set.String("datadir", datadir, "data directory for geth")
	set.String("logdir", datadir, "log dir for glog")
	c = cli.NewContext(nil, set, nil)

	// Construct the textual version string from the individual components
	vString = fmt.Sprintf("%d.%d.%d", versionMajor, versionMinor, versionPatch)

	// Construct the version release oracle configuration
	rConfig.Oracle = common.HexToAddress(versionOracle)

	rConfig.Major = uint32(versionMajor)
	rConfig.Minor = uint32(versionMinor)
	rConfig.Patch = uint32(versionPatch)

	utils.DebugSetup(c)
	currentNode, accountSync = utils.MakeSystemNode(clientIdentifier, vString, rConfig, makeDefaultExtra(), c)

	return currentNode

}

// StartNode starts a geth node entity
func RunNode(nodeIn *node.Node) {
	utils.StartNode(nodeIn)

	if err := nodeIn.Service(&accountManager); err != nil {
		glog.V(logger.Warn).Infoln("cannot get account manager:", err)
	}
	if err := nodeIn.Service(&whisperService); err != nil {
		glog.V(logger.Warn).Infoln("cannot get whisper service:", err)
	}
	if err := nodeIn.Service(&lightEthereum); err != nil {
		glog.V(logger.Warn).Infoln("cannot get light ethereum service:", err)
	}
	lightEthereum.StatusBackend.SetTransactionQueueHandler(onSendTransactionRequest)

	nodeIn.Wait()
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
