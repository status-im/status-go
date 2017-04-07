// Copyright 2014 The go-ethereum Authors
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

// evm executes EVM code snippets.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	goruntime "runtime"
	"time"

	"github.com/teslapatrick/go-ethereum/cmd/utils"
	"github.com/teslapatrick/go-ethereum/common"
	"github.com/teslapatrick/go-ethereum/core/state"
	"github.com/teslapatrick/go-ethereum/core/vm"
	"github.com/teslapatrick/go-ethereum/core/vm/runtime"
	"github.com/teslapatrick/go-ethereum/crypto"
	"github.com/teslapatrick/go-ethereum/ethdb"
	"github.com/teslapatrick/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
)

var gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)

var (
	app = utils.NewApp(gitCommit, "the evm command line interface")

	DebugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "output full trace logs",
	}
	CodeFlag = cli.StringFlag{
		Name:  "code",
		Usage: "EVM code",
	}
	CodeFileFlag = cli.StringFlag{
		Name:  "codefile",
		Usage: "file containing EVM code",
	}
	GasFlag = cli.StringFlag{
		Name:  "gas",
		Usage: "gas limit for the evm",
		Value: "10000000000",
	}
	PriceFlag = cli.StringFlag{
		Name:  "price",
		Usage: "price set for the evm",
		Value: "0",
	}
	ValueFlag = cli.StringFlag{
		Name:  "value",
		Usage: "value set for the evm",
		Value: "0",
	}
	DumpFlag = cli.BoolFlag{
		Name:  "dump",
		Usage: "dumps the state after the run",
	}
	InputFlag = cli.StringFlag{
		Name:  "input",
		Usage: "input for the EVM",
	}
	SysStatFlag = cli.BoolFlag{
		Name:  "sysstat",
		Usage: "display system stats",
	}
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "sets the verbosity level",
	}
	CreateFlag = cli.BoolFlag{
		Name:  "create",
		Usage: "indicates the action should be create rather than call",
	}
	DisableGasMeteringFlag = cli.BoolFlag{
		Name:  "nogasmetering",
		Usage: "disable gas metering",
	}
)

func init() {
	app.Flags = []cli.Flag{
		CreateFlag,
		DebugFlag,
		VerbosityFlag,
		SysStatFlag,
		CodeFlag,
		CodeFileFlag,
		GasFlag,
		PriceFlag,
		ValueFlag,
		DumpFlag,
		InputFlag,
		DisableGasMeteringFlag,
	}
	app.Action = run
}

func run(ctx *cli.Context) error {
	glog.SetToStderr(true)
	glog.SetV(ctx.GlobalInt(VerbosityFlag.Name))

	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, db)
	sender := statedb.CreateAccount(common.StringToAddress("sender"))

	logger := vm.NewStructLogger(nil)

	tstart := time.Now()

	var (
		code []byte
		ret  []byte
		err  error
	)

	if ctx.GlobalString(CodeFlag.Name) != "" {
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
		code = common.Hex2Bytes(string(hexcode[:]))
	}

	if ctx.GlobalBool(CreateFlag.Name) {
		input := append(code, common.Hex2Bytes(ctx.GlobalString(InputFlag.Name))...)
		ret, _, err = runtime.Create(input, &runtime.Config{
			Origin:   sender.Address(),
			State:    statedb,
			GasLimit: common.Big(ctx.GlobalString(GasFlag.Name)),
			GasPrice: common.Big(ctx.GlobalString(PriceFlag.Name)),
			Value:    common.Big(ctx.GlobalString(ValueFlag.Name)),
			EVMConfig: vm.Config{
				Tracer:             logger,
				Debug:              ctx.GlobalBool(DebugFlag.Name),
				DisableGasMetering: ctx.GlobalBool(DisableGasMeteringFlag.Name),
			},
		})
	} else {
		receiver := statedb.CreateAccount(common.StringToAddress("receiver"))
		receiver.SetCode(crypto.Keccak256Hash(code), code)

		ret, err = runtime.Call(receiver.Address(), common.Hex2Bytes(ctx.GlobalString(InputFlag.Name)), &runtime.Config{
			Origin:   sender.Address(),
			State:    statedb,
			GasLimit: common.Big(ctx.GlobalString(GasFlag.Name)),
			GasPrice: common.Big(ctx.GlobalString(PriceFlag.Name)),
			Value:    common.Big(ctx.GlobalString(ValueFlag.Name)),
			EVMConfig: vm.Config{
				Tracer:             logger,
				Debug:              ctx.GlobalBool(DebugFlag.Name),
				DisableGasMetering: ctx.GlobalBool(DisableGasMeteringFlag.Name),
			},
		})
	}
	vmdone := time.Since(tstart)

	if ctx.GlobalBool(DumpFlag.Name) {
		statedb.Commit(true)
		fmt.Println(string(statedb.Dump()))
	}
	vm.StdErrFormat(logger.StructLogs())

	if ctx.GlobalBool(SysStatFlag.Name) {
		var mem goruntime.MemStats
		goruntime.ReadMemStats(&mem)
		fmt.Printf("vm took %v\n", vmdone)
		fmt.Printf(`alloc:      %d
tot alloc:  %d
no. malloc: %d
heap alloc: %d
heap objs:  %d
num gc:     %d
`, mem.Alloc, mem.TotalAlloc, mem.Mallocs, mem.HeapAlloc, mem.HeapObjects, mem.NumGC)
	}

	fmt.Printf("OUT: 0x%x", ret)
	if err != nil {
		fmt.Printf(" error: %v", err)
	}
	fmt.Println()
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
