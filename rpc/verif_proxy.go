//go:build nimbus_light_client
// +build nimbus_light_client

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/nodecfg"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	proxy "github.com/vitvly/lc-proxy-wrapper"
	proxytypes "github.com/vitvly/lc-proxy-wrapper/types"
)

// TODO Figure out a better way of setting this
const defaultBlockRoot = "0x307397cd6a44e038e6da207ae7e969681b848eb2654fde1f26e8e72c66ecfd08"

func init() {
	verifProxyInitFn = func(c *Client) {
		defer c.wg.Done()

		blockRoot, err := nodecfg.GetNimbusTrustedBlockRoot(c.db)
		if err != nil {
			fmt.Println("verif_proxy GetNimbusTrustedBlockRoot error", err)
		}
		if len(blockRoot) == 0 {
			blockRoot = defaultBlockRoot
		}

		cfg := proxy.Config{
			Eth2Network:      "mainnet",
			TrustedBlockRoot: blockRoot,
			Web3Url:          c.upstreamURL,
			RpcAddress:       "127.0.0.1",
			RpcPort:          8545,
			LogLevel:         "INFO",
		}

		proxyCh, err := proxy.StartVerifProxy(&cfg)
		if err != nil {
			return
		}
		ticker := time.NewTicker(5 * time.Second)
		proxyInitialized := false
		var proxyClient *gethrpc.Client

		for {
			select {
			case <-c.ctx.Done():
				// Client's Close() method has been invoked
				c.log.Info("Context done")
				proxy.StopVerifProxy()
			case ev := <-proxyCh:
				if ev.EventType == proxytypes.Stopped || ev.EventType == proxytypes.Error {
					return
				}

				if !proxyInitialized {
					proxyInitialized = true
					// Create RPC client using verification proxy endpoint
					endpoint := "http://" + cfg.RpcAddress + ":" + fmt.Sprint(cfg.RpcPort)
					proxyClient, err = gethrpc.DialHTTP(endpoint)
					if err != nil {
						log.Error("Error when creating VerifProxy client", err)
						return
					}

				}
				if proxyInitialized && ev.EventType == proxytypes.FinalizedHeader {
					err = storeUpdatedBlockRoot(c, ev.Msg)
					if err != nil {
						fmt.Println("verif_proxy storeUpdatedBlockRoot", err)
					}
				}
			case <-ticker.C:
				if proxyInitialized {
					// Invoke a simple RPC method in order to ascertain that proxy is up and running
					ctx, _ := context.WithTimeout(c.ctx, 5*time.Second)
					_, err := blockNumber(ctx, proxyClient)
					fmt.Println("verif_proxy blockNumber result", err)
					if err == nil {
						ticker.Stop()
						installAPIHandlers(c, proxyClient)
					}
				}

			}
		}

	}
}

func installAPIHandlers(c *Client, proxyClient *gethrpc.Client) {
	c.log.Info("### installAPIHandlers()")
	// Install API handlers
	c.RegisterHandler(
		"eth_chainId",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			return chainId(ctx, proxyClient)
		},
	)

	c.RegisterHandler(
		"eth_blockNumber",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			return blockNumber(ctx, proxyClient)
		},
	)

	c.RegisterHandler(
		"eth_getBalance",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			addr := params[0].(common.Address)
			block := params[1].(string)
			return getBalance(ctx, proxyClient, addr, block)
		},
	)

	c.RegisterHandler(
		"eth_getStorageAt",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			addr := params[0].(common.Address)
			slot := params[1].(string)
			block := params[2].(string)
			return getStorageAt(ctx, proxyClient, addr, slot, block)
		},
	)

	c.RegisterHandler(
		"eth_getTransactionCount",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			addr := params[0].(common.Address)
			block := params[1].(string)
			return getTransactionCount(ctx, proxyClient, addr, block)
		},
	)

	c.RegisterHandler(
		"eth_getCode",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			addr := params[0].(common.Address)
			block := params[1].(string)
			return getCode(ctx, proxyClient, addr, block)
		},
	)

	c.RegisterHandler(
		"eth_getBlockByNumber",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			block := params[0].(string)
			fullTransactions := params[1].(bool)
			return getBlockByNumber(ctx, proxyClient, block, fullTransactions)
		},
	)

	c.RegisterHandler(
		"eth_getBlockByHash",
		func(ctx context.Context, v uint64, params ...interface{}) (interface{}, error) {
			blockHash := params[0].(string)
			fullTransactions := params[1].(bool)
			return getBlockByHash(ctx, proxyClient, blockHash, fullTransactions)
		},
	)
}

func storeUpdatedBlockRoot(c *Client, msg string) error {

	// Store updated trusted block root into DB
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(msg), &jsonData)
	if err != nil {
		return errors.New("could not unmarshal finalized header")
	}
	beacon, exists := jsonData["beacon"]
	if !exists {
		return errors.New("could not unmarshal beacon json")
	}
	stateRoot, exists := (beacon.(map[string]interface{}))["state_root"]
	if !exists {
		return errors.New("could not find state_root")
	}
	// TODO store stateRoot into DB
	err = nodecfg.SetNimbusTrustedBlockRoot(c.db, "0x"+stateRoot.(string))
	if err != nil {
		return err
	}

	return nil
}

func chainId(ctx context.Context, proxyClient *gethrpc.Client) (interface{}, error) {
	var result hexutil.Big
	err := proxyClient.CallContext(ctx, &result, "eth_chainId")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func blockNumber(ctx context.Context, proxyClient *gethrpc.Client) (interface{}, error) {
	var result hexutil.Big
	err := proxyClient.CallContext(ctx, &result, "eth_blockNumber")
	if err != nil {
		return nil, err
	}
	return result, nil

}

func getBalance(ctx context.Context, proxyClient *gethrpc.Client, address common.Address, block string) (interface{}, error) {

	var result hexutil.Big
	err := proxyClient.CallContext(ctx, &result, "eth_getBalance", address, block)
	if err != nil {
		return nil, err
	}
	return result, nil

}

func getStorageAt(ctx context.Context, proxyClient *gethrpc.Client, address common.Address, slot string, block string) (interface{}, error) {
	var result string
	err := proxyClient.CallContext(ctx, &result, "eth_getStorageAt", address, slot, block)
	if err != nil {
		return nil, err
	}
	return result, nil

}

func getTransactionCount(ctx context.Context, proxyClient *gethrpc.Client, address common.Address, block string) (interface{}, error) {
	var result hexutil.Big
	err := proxyClient.CallContext(ctx, &result, "eth_getTransactionCount", address, block)
	if err != nil {
		return nil, err
	}
	return result, nil

}

func getCode(ctx context.Context, proxyClient *gethrpc.Client, address common.Address, block string) (interface{}, error) {
	var result string
	err := proxyClient.CallContext(ctx, &result, "eth_getCode", address, block)
	if err != nil {
		return nil, err
	}
	return result, nil

}

func getBlockByNumber(ctx context.Context, proxyClient *gethrpc.Client, block string, fullTransactions bool) (interface{}, error) {
	var result types.Block
	err := proxyClient.CallContext(ctx, &result, "eth_getBlockByNumber", block, fullTransactions)
	if err != nil {
		return nil, err
	}
	return result, nil

}

func getBlockByHash(ctx context.Context, proxyClient *gethrpc.Client, blockHash string, fullTransactions bool) (interface{}, error) {
	var result types.Block
	err := proxyClient.CallContext(ctx, &result, "eth_getBlockByHash", blockHash, fullTransactions)
	if err != nil {
		return nil, err
	}
	return result, nil

}
