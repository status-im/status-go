// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/teslapatrick/go-ethereum/cmd/utils"
	"github.com/teslapatrick/go-ethereum/common"
	"github.com/teslapatrick/go-ethereum/console"
	"github.com/teslapatrick/go-ethereum/core"
	"github.com/teslapatrick/go-ethereum/core/state"
	"github.com/teslapatrick/go-ethereum/core/types"
	"github.com/teslapatrick/go-ethereum/ethdb"
	"github.com/teslapatrick/go-ethereum/logger"
	"github.com/teslapatrick/go-ethereum/logger/glog"
	"github.com/teslapatrick/go-ethereum/trie"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gopkg.in/urfave/cli.v1"
)

var (
	initCommand = cli.Command{
		Action:    initGenesis,
		Name:      "init",
		Usage:     "Bootstrap and initialize a new genesis block",
		ArgsUsage: "<genesisPath>",
		Category:  "BLOCKCHAIN COMMANDS",
		Description: `
The init command initializes a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.
`,
	}
	importCommand = cli.Command{
		Action:    importChain,
		Name:      "import",
		Usage:     "Import a blockchain file",
		ArgsUsage: "<filename>",
		Category:  "BLOCKCHAIN COMMANDS",
		Description: `
TODO: Please write this
`,
	}
	exportCommand = cli.Command{
		Action:    exportChain,
		Name:      "export",
		Usage:     "Export blockchain into file",
		ArgsUsage: "<filename> [<blockNumFirst> <blockNumLast>]",
		Category:  "BLOCKCHAIN COMMANDS",
		Description: `
Requires a first argument of the file to write to.
Optional second and third arguments control the first and
last block to write. In this mode, the file will be appended
if already existing.
`,
	}
	upgradedbCommand = cli.Command{
		Action:    upgradeDB,
		Name:      "upgradedb",
		Usage:     "Upgrade chainblock database",
		ArgsUsage: " ",
		Category:  "BLOCKCHAIN COMMANDS",
		Description: `
TODO: Please write this
`,
	}
	removedbCommand = cli.Command{
		Action:    removeDB,
		Name:      "removedb",
		Usage:     "Remove blockchain and state databases",
		ArgsUsage: " ",
		Category:  "BLOCKCHAIN COMMANDS",
		Description: `
TODO: Please write this
`,
	}
	dumpCommand = cli.Command{
		Action:    dump,
		Name:      "dump",
		Usage:     "Dump a specific block from storage",
		ArgsUsage: "[<blockHash> | <blockNum>]...",
		Category:  "BLOCKCHAIN COMMANDS",
		Description: `
The arguments are interpreted as block numbers or hashes.
Use "ethereum dump 0" to dump the genesis block.
`,
	}
)

// initGenesis will initialise the given JSON format genesis file and writes it as
// the zero'd block (i.e. genesis) or will fail hard if it can't succeed.
func initGenesis(ctx *cli.Context) error {
	genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		utils.Fatalf("must supply path to genesis JSON file")
	}

	stack := makeFullNode(ctx)
	chaindb := utils.MakeChainDatabase(ctx, stack)

	genesisFile, err := os.Open(genesisPath)
	if err != nil {
		utils.Fatalf("failed to read genesis file: %v", err)
	}
	defer genesisFile.Close()

	block, err := core.WriteGenesisBlock(chaindb, genesisFile)
	if err != nil {
		utils.Fatalf("failed to write genesis block: %v", err)
	}
	glog.V(logger.Info).Infof("successfully wrote genesis block and/or chain rule set: %x", block.Hash())
	return nil
}

func importChain(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	stack := makeFullNode(ctx)
	chain, chainDb := utils.MakeChain(ctx, stack)
	defer chainDb.Close()

	// Start periodically gathering memory profiles
	var peakMemAlloc, peakMemSys uint64
	go func() {
		stats := new(runtime.MemStats)
		for {
			runtime.ReadMemStats(stats)
			if atomic.LoadUint64(&peakMemAlloc) < stats.Alloc {
				atomic.StoreUint64(&peakMemAlloc, stats.Alloc)
			}
			if atomic.LoadUint64(&peakMemSys) < stats.Sys {
				atomic.StoreUint64(&peakMemSys, stats.Sys)
			}
			time.Sleep(5 * time.Second)
		}
	}()
	// Import the chain
	start := time.Now()
	if err := utils.ImportChain(chain, ctx.Args().First()); err != nil {
		utils.Fatalf("Import error: %v", err)
	}
	fmt.Printf("Import done in %v.\n\n", time.Since(start))

	// Output pre-compaction stats mostly to see the import trashing
	db := chainDb.(*ethdb.LDBDatabase)

	stats, err := db.LDB().GetProperty("leveldb.stats")
	if err != nil {
		utils.Fatalf("Failed to read database stats: %v", err)
	}
	fmt.Println(stats)
	fmt.Printf("Trie cache misses:  %d\n", trie.CacheMisses())
	fmt.Printf("Trie cache unloads: %d\n\n", trie.CacheUnloads())

	// Print the memory statistics used by the importing
	mem := new(runtime.MemStats)
	runtime.ReadMemStats(mem)

	fmt.Printf("Object memory: %.3f MB current, %.3f MB peak\n", float64(mem.Alloc)/1024/1024, float64(atomic.LoadUint64(&peakMemAlloc))/1024/1024)
	fmt.Printf("System memory: %.3f MB current, %.3f MB peak\n", float64(mem.Sys)/1024/1024, float64(atomic.LoadUint64(&peakMemSys))/1024/1024)
	fmt.Printf("Allocations:   %.3f million\n", float64(mem.Mallocs)/1000000)
	fmt.Printf("GC pause:      %v\n\n", time.Duration(mem.PauseTotalNs))

	// Compact the entire database to more accurately measure disk io and print the stats
	start = time.Now()
	fmt.Println("Compacting entire database...")
	if err = db.LDB().CompactRange(util.Range{}); err != nil {
		utils.Fatalf("Compaction failed: %v", err)
	}
	fmt.Printf("Compaction done in %v.\n\n", time.Since(start))

	stats, err = db.LDB().GetProperty("leveldb.stats")
	if err != nil {
		utils.Fatalf("Failed to read database stats: %v", err)
	}
	fmt.Println(stats)

	return nil
}

func exportChain(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		utils.Fatalf("This command requires an argument.")
	}
	stack := makeFullNode(ctx)
	chain, _ := utils.MakeChain(ctx, stack)
	start := time.Now()

	var err error
	fp := ctx.Args().First()
	if len(ctx.Args()) < 3 {
		err = utils.ExportChain(chain, fp)
	} else {
		// This can be improved to allow for numbers larger than 9223372036854775807
		first, ferr := strconv.ParseInt(ctx.Args().Get(1), 10, 64)
		last, lerr := strconv.ParseInt(ctx.Args().Get(2), 10, 64)
		if ferr != nil || lerr != nil {
			utils.Fatalf("Export error in parsing parameters: block number not an integer\n")
		}
		if first < 0 || last < 0 {
			utils.Fatalf("Export error: block number must be greater than 0\n")
		}
		err = utils.ExportAppendChain(chain, fp, uint64(first), uint64(last))
	}

	if err != nil {
		utils.Fatalf("Export error: %v\n", err)
	}
	fmt.Printf("Export done in %v", time.Since(start))
	return nil
}

func removeDB(ctx *cli.Context) error {
	stack := utils.MakeNode(ctx, clientIdentifier, gitCommit)
	dbdir := stack.ResolvePath(utils.ChainDbName(ctx))
	if !common.FileExist(dbdir) {
		fmt.Println(dbdir, "does not exist")
		return nil
	}

	fmt.Println(dbdir)
	confirm, err := console.Stdin.PromptConfirm("Remove this database?")
	switch {
	case err != nil:
		utils.Fatalf("%v", err)
	case !confirm:
		fmt.Println("Operation aborted")
	default:
		fmt.Println("Removing...")
		start := time.Now()
		os.RemoveAll(dbdir)
		fmt.Printf("Removed in %v\n", time.Since(start))
	}
	return nil
}

func upgradeDB(ctx *cli.Context) error {
	glog.Infoln("Upgrading blockchain database")

	stack := utils.MakeNode(ctx, clientIdentifier, gitCommit)
	chain, chainDb := utils.MakeChain(ctx, stack)
	bcVersion := core.GetBlockChainVersion(chainDb)
	if bcVersion == 0 {
		bcVersion = core.BlockChainVersion
	}

	// Export the current chain.
	filename := fmt.Sprintf("blockchain_%d_%s.chain", bcVersion, time.Now().Format("20060102_150405"))
	exportFile := filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), filename)
	if err := utils.ExportChain(chain, exportFile); err != nil {
		utils.Fatalf("Unable to export chain for reimport %s", err)
	}
	chainDb.Close()
	if dir := dbDirectory(chainDb); dir != "" {
		os.RemoveAll(dir)
	}

	// Import the chain file.
	chain, chainDb = utils.MakeChain(ctx, stack)
	core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	err := utils.ImportChain(chain, exportFile)
	chainDb.Close()
	if err != nil {
		utils.Fatalf("Import error %v (a backup is made in %s, use the import command to import it)", err, exportFile)
	} else {
		os.Remove(exportFile)
		glog.Infoln("Import finished")
	}
	return nil
}

func dbDirectory(db ethdb.Database) string {
	ldb, ok := db.(*ethdb.LDBDatabase)
	if !ok {
		return ""
	}
	return ldb.Path()
}

func dump(ctx *cli.Context) error {
	stack := makeFullNode(ctx)
	chain, chainDb := utils.MakeChain(ctx, stack)
	for _, arg := range ctx.Args() {
		var block *types.Block
		if hashish(arg) {
			block = chain.GetBlockByHash(common.HexToHash(arg))
		} else {
			num, _ := strconv.Atoi(arg)
			block = chain.GetBlockByNumber(uint64(num))
		}
		if block == nil {
			fmt.Println("{}")
			utils.Fatalf("block not found")
		} else {
			state, err := state.New(block.Root(), chainDb)
			if err != nil {
				utils.Fatalf("could not create new state: %v", err)
			}
			fmt.Printf("%s\n", state.Dump())
		}
	}
	chainDb.Close()
	return nil
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}

func closeAll(dbs ...ethdb.Database) {
	for _, db := range dbs {
		db.Close()
	}
}
