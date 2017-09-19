package rpc

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"syscall"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/log"
)

const (
	encodedBlockNumberLen = 8
)

// SyncPreRequisites verifies if the local node has enough free disk space to perform
// the light client sync operation
func SyncPreRequisites(client *Client) (bool, error) {
	nbytes, err := getSpaceRequisite(client)
	if err != nil {
		return false, err
	}
	return hasFreeSpace(nbytes)
}

// getSpaceRequisite returns an estimate of the number of bytes required to perform the sync operation
func getSpaceRequisite(client *Client) (uint64, error) {
	var nbytes uint64

	lheader, err := getLatestHeader(client.local)
	if err != nil {
		return 0, fmt.Errorf("get the latest local header: %s", err)
	}
	log.Info("Loaded most recent local header", "number", lheader.Number)

	uheader, err := getLatestHeader(client.upstream)
	if err != nil {
		return 0, fmt.Errorf("get the latest upstream header: %s", err)
	}
	log.Info("Loaded most recent upstream header", "number", uheader.Number)

	// number of headers left to sync
	hdiff := uheader.Number.Uint64() - lheader.Number.Uint64()
	if hdiff > 0 {
		// get the space requirement for each header based
		lraw, err := rlp.EncodeToBytes(lheader)
		if err != nil {
			return 0, err
		}
		uraw, err := rlp.EncodeToBytes(uheader)
		if err != nil {
			return 0, err
		}
		// average size for headers between the local and upstream one
		// there are 5 scalar variables in the header that will always increase (Ex: block number)
		// reference - database_utils.WriteHeader (encoded blocknumber + encoded header) + goleveldb metadata
		encodedHeaderLen := uint64((len(lraw) + len(uraw)) / 2)
		nbytes = encodedHeaderLen + encodedBlockNumberLen
		nbytes *= hdiff
	}
	log.Info("Updated sync requirements", "header count", hdiff, "space needed(bytes)", nbytes)
	return nbytes, nil
}

// getLatestHeader returns the latest header for a specific eth client
func getLatestHeader(client *gethrpc.Client) (*types.Header, error) {
	ethClient := ethclient.NewClient(client)
	header, err := ethClient.HeaderByNumber(context.Background(), big.NewInt(int64(gethrpc.LatestBlockNumber)))
	if err != nil {
		return nil, err
	}
	return header, nil
}

// hasFreeSpace verifies if there's a certain value of free disk space.
func hasFreeSpace(nbytes uint64) (bool, error) {
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	if err := syscall.Statfs(wd, &stat); err != nil {
		return false, err
	}
	free := stat.Bavail * uint64(stat.Bsize)
	if nbytes > free {
		return false, nil
	}
	log.Info("Finished disk analysis operation", "free space", free, "required space", nbytes)
	return true, nil
}
