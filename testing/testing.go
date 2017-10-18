package integration

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

var (
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

	// DefaultNodeSyncTimeout will mostly be used as timeout
	// for contexts of EnsureNodeSync(). Still individual ones
	// can be set.
	DefaultNodeSyncTimeout = 5 * time.Minute
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
// with tests afterwards.
func EnsureNodeSync(ctx context.Context, nodeManager common.NodeManager) error {
	les, err := nodeManager.LightEthereumService()
	if err != nil {
		return err
	}
	if les == nil {
		return errors.New("LightEthereumService is nil")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			downloader := les.Downloader()

			if downloader != nil {
				isSyncing := downloader.Synchronising()
				progress := downloader.Progress()

				if !isSyncing && progress.HighestBlock > 0 && progress.CurrentBlock >= progress.HighestBlock {
					return nil
				}
			}

		}
	}
}
