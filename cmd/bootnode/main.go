// bootnode runs a bootstrap node for the Ethereum Discovery Protocol.
package main

import (
	"flag"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
)

func main() {
	var (
		listenAddr  = flag.String("addr", ":30301", "listen address")
		nodeKeyFile = flag.String("nodekey", "", "private key filename")
		verbosity   = flag.Int("verbosity", int(log.LvlInfo), "log verbosity (0-9)")
		vmodule     = flag.String("vmodule", "", "log verbosity pattern")
	)
	flag.Parse()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*verbosity))
	glogger.Vmodule(*vmodule)
	log.Root().SetHandler(glogger)

	nodeKey, err := crypto.LoadECDSA(*nodeKeyFile)
	if err != nil {
		log.Crit("Failed to load ecdsa key from", "file", *nodeKeyFile, "error", err)
	}

	addr, err := net.ResolveUDPAddr("udp", *listenAddr)
	if err != nil {
		log.Crit("Unable to resolve UDP", "address", *listenAddr, "error", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Crit("Unable to listen on udp", "address", addr, "error", err)
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	tab, err := discv5.ListenUDP(nodeKey, conn, realaddr, "", nil)
	if err != nil {
		log.Crit("Failed to create discovery v5 table:", "error", err)
	}
	defer tab.Close()
	select {}
}
