// Copyright 2017 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	goruntime "runtime"

	"github.com/ethereum/go-ethereum/cmd/evm/internal/compiler"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

var runCommand = cli.Command{
	Action:      runCmd,
	Name:        "run",
	Usage:       "run arbitrary evm binary",
	ArgsUsage:   "<code>",
	Description: `The run command runs arbitrary EVM code.`,
}

func runCmd(ctx *cli.Context) error {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(ctx.GlobalInt(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)

	var (
		db, _      = ethdb.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, db)
		sender     = common.StringToAddress("sender")
		logger     = vm.NewStructLogger(nil)
	)
	statedb.CreateAccount(sender)

	var (
		code []byte
		ret  []byte
		err  error
	)
	if fn := ctx.Args().First(); len(fn) > 0 {
		src, err := ioutil.ReadFile(fn)
		if err != nil {
			return err
		}

		bin, err := compiler.Compile(fn, src, false)
		if err != nil {
			return err
		}
		code = common.Hex2Bytes(bin)
	} else if ctx.GlobalString(CodeFlag.Name) != "" {
		code = common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name))
	} else {
		var hexcode []byte
		if ctx.GlobalString(CodeFileFlag.Name) != "" {
			var err error
			hexcode, err = ioutil.ReadFile(ctx.GlobalString(CodeFileFlag.Name))
			if err != nil {
				fmt.Printf("Could not load code from file: %v\n", err)
				os.Exit(1)
			}
		} else {
			var err error
			hexcode, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				fmt.Printf("Could not load code from stdin: %v\n", err)
				os.Exit(1)
			}
		}
		code = common.Hex2Bytes(string(bytes.TrimRight(hexcode, "\n")))
	}

	runtimeConfig := runtime.Config{
		Origin:   sender,
		State:    statedb,
		GasLimit: ctx.GlobalUint64(GasFlag.Name),
		GasPrice: utils.GlobalBig(ctx, PriceFlag.Name),
		Value:    utils.GlobalBig(ctx, ValueFlag.Name),
		EVMConfig: vm.Config{
			Tracer:             logger,
			Debug:              ctx.GlobalBool(DebugFlag.Name),
			DisableGasMetering: ctx.GlobalBool(DisableGasMeteringFlag.Name),
		},
	}

	tstart := time.Now()
	if ctx.GlobalBool(CreateFlag.Name) {
		input := append(code, common.Hex2Bytes(ctx.GlobalString(InputFlag.Name))...)
		ret, _, err = runtime.Create(input, &runtimeConfig)
	} else {
		receiver := common.StringToAddress("receiver")
		statedb.SetCode(receiver, code)

		ret, err = runtime.Call(receiver, common.Hex2Bytes(ctx.GlobalString(InputFlag.Name)), &runtimeConfig)
	}
	execTime := time.Since(tstart)

	if ctx.GlobalBool(DumpFlag.Name) {
		statedb.Commit(true)
		fmt.Println(string(statedb.Dump()))
	}

	if ctx.GlobalBool(DebugFlag.Name) {
		fmt.Fprintln(os.Stderr, "#### TRACE ####")
		vm.WriteTrace(os.Stderr, logger.StructLogs())
		fmt.Fprintln(os.Stderr, "#### LOGS ####")
		vm.WriteLogs(os.Stderr, statedb.Logs())

		var mem goruntime.MemStats
		goruntime.ReadMemStats(&mem)
		fmt.Fprintf(os.Stderr, `evm execution time: %v
heap objects:       %d
allocations:        %d
total allocations:  %d
GC calls:           %d

`, execTime, mem.HeapObjects, mem.Alloc, mem.TotalAlloc, mem.NumGC)
	}

	fmt.Printf("0x%x", ret)
	if err != nil {
		fmt.Printf(" error: %v", err)
	}
	fmt.Println()
	return nil
}
