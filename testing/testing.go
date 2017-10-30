package integration

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

var (
	networkSelected = flag.String("network", "statuschain", "-network=NETWORKID or -network=NETWORKNAME to select network used for tests")

	// ErrNoRemoteURL is returned when network id has no associated url.
	ErrNoRemoteURL = errors.New("network id requires a remote URL")

	// TestConfig defines the default config usable at package-level.
	TestConfig *common.TestConfig

	// RootDir is the main application directory
	RootDir string

	// TestDataDir is data directory used for tests
	TestDataDir string

	// TestNetworkNames network ID to name mapping
	TestNetworkNames = map[int]string{
		params.MainNetworkID:        "Mainnet",
		params.RopstenNetworkID:     "Ropsten",
		params.RinkebyNetworkID:     "Rinkeby",
		params.StatusChainNetworkID: "StatusChain",
	}
)

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// setup root directory
	RootDir = filepath.Dir(pwd)
	if strings.HasSuffix(RootDir, "geth") || strings.HasSuffix(RootDir, "cmd") { // we need to hop one more level
		RootDir = filepath.Join(RootDir, "..")
	}

	// setup auxiliary directories
	TestDataDir = filepath.Join(RootDir, ".ethereumtest")

	TestConfig, err = common.LoadTestConfig()
	if err != nil {
		panic(err)
	}
}

// LoadFromFile is useful for loading test data, from testdata/filename into a variable
// nolint: errcheck
func LoadFromFile(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)
	f.Close()

	return string(buf.Bytes())
}

// EnsureNodeSync waits until node synchronzation is done to continue
// with tests afterwards. Panics in case of an error or a timeout.
func EnsureNodeSync(nodeManager common.NodeManager) {
	nc, err := nodeManager.NodeConfig()
	if err != nil {
		panic("can't retrieve NodeConfig")
	}
	// Don't wait for any blockchain sync for the local private chain as blocks are never mined.
	if nc.NetworkID == params.StatusChainNetworkID {
		return
	}

	les, err := nodeManager.LightEthereumService()
	if err != nil {
		panic(err)
	}
	if les == nil {
		panic("LightEthereumService is nil")
	}

	timeouter := time.NewTimer(20 * time.Minute)
	defer timeouter.Stop()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeouter.C:
			panic("timout during node synchronization")
		case <-ticker.C:
			downloader := les.Downloader()

			if downloader != nil {
				isSyncing := downloader.Synchronising()
				progress := downloader.Progress()

				if !isSyncing && progress.HighestBlock > 0 && progress.CurrentBlock >= progress.HighestBlock {
					return
				}
			}
		}
	}
}

// GetRemoteURLFromNetworkID returns associated network url for giving network id.
func GetRemoteURLFromNetworkID(id int) (url string, err error) {
	switch id {
	case params.MainNetworkID:
		url = params.MainnetEthereumNetworkURL
	case params.RinkebyNetworkID:
		url = params.RinkebyEthereumNetworkURL
	case params.RopstenNetworkID:
		url = params.RopstenEthereumNetworkURL
	default:
		err = ErrNoRemoteURL
	}

	return
}

// GetHeadHashFromNetworkID returns the hash associated with a given network id.
func GetHeadHashFromNetworkID(id int) string {
	switch id {
	case params.RinkebyNetworkID:
		return "0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177"
	case params.RopstenNetworkID:
		return "0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d"
	case params.StatusChainNetworkID:
		return "0x50e6edb4e90d9616ac8bf7119ee37c4048c41ad09328676e1b39dc68a0ecfb3d"
	}

	return ""
}

// GetRemoteURL returns the url associated with a given network id.
func GetRemoteURL() (string, error) {
	return GetRemoteURLFromNetworkID(GetNetworkID())
}

// GetHeadHash returns the hash associated with a given network id.
func GetHeadHash() string {
	return GetHeadHashFromNetworkID(GetNetworkID())
}

// GetNetworkID returns appropriate network id for test based on
// default or provided -network flag.
func GetNetworkID() int {
	switch strings.ToLower(*networkSelected) {
	case fmt.Sprintf("%d", params.RinkebyNetworkID), "rinkeby":
		return params.RinkebyNetworkID
	case fmt.Sprintf("%d", params.RopstenNetworkID), "ropsten", "testnet":
		return params.RopstenNetworkID
	case fmt.Sprintf("%d", params.StatusChainNetworkID), "statuschain":
		return params.StatusChainNetworkID
	}

	return params.StatusChainNetworkID
}
