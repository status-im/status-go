package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ebfe/scard"
	"github.com/status-im/status-go/smartcard/lightwallet"
)

type commandFunc func(*lightwallet.Installer) error

var (
	commands map[string]commandFunc

	flagCapFile   = flag.String("f", "", "cap file path")
	flagOverwrite = flag.Bool("o", false, "overwrite applet if already installed")
)

func init() {
	commands = map[string]commandFunc{
		"install": commandInstall,
		"status":  commandStatus,
		"delete":  commandDelete,
	}
}

func usage() {
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()
	cmd := flag.Arg(0)
	if cmd == "" {
		log.Printf("you must specify a command (install, status, delete).\n")
		usage()
	}

	ctx, err := scard.EstablishContext()
	if err != nil {
		log.Fatal(err)
	}
	defer ctx.Release()

	readers, err := ctx.ListReaders()
	if err != nil {
		log.Fatal(err)
	}

	if len(readers) == 0 {
		log.Fatal("couldn't find any reader")
	}

	if len(readers) > 1 {
		log.Fatal("too many readers found")
	}

	reader := readers[0]
	fmt.Printf("using reader %s:\n", reader)
	fmt.Printf("connecting to card in %s\n", reader)
	card, err := ctx.Connect(reader, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		log.Fatal(err)
	}
	defer card.Disconnect(scard.ResetCard)

	status, err := card.Status()
	if err != nil {
		log.Fatal(err)
	}

	switch status.ActiveProtocol {
	case scard.ProtocolT0:
		fmt.Printf("Protocol T0\n")
	case scard.ProtocolT1:
		fmt.Printf("Protocol T1\n")
	default:
		fmt.Printf("Unknown protocol\n")
	}

	i := lightwallet.NewInstaller(card)
	if f, ok := commands[cmd]; ok {
		err = f(i)
		os.Exit(0)
	}

	fmt.Printf("unknown command %s\n", cmd)
	usage()
}

func commandInstall(i *lightwallet.Installer) error {
	if *flagCapFile == "" {
		log.Printf("you must specify a cap file path with the -f flag\n")
	}

	f, err := os.Open(*flagCapFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	secrets, err := i.Install(f, *flagOverwrite)
	if err != nil {
		log.Fatal("installation error: ", err)
	}

	fmt.Printf("PUK %s\n", secrets.Puk())
	fmt.Printf("Pairing password: %s\n", secrets.PairingPass())

	return nil
}

func commandStatus(i *lightwallet.Installer) error {
	installed, err := i.Info()
	if err != nil {
		return err
	}

	if installed {
		fmt.Printf("applet already installed\n")
	} else {
		fmt.Printf("applet not installed\n")
	}

	return nil
}

func commandDelete(i *lightwallet.Installer) error {
	err := i.Delete()
	if err != nil {
		return err
	}

	fmt.Printf("applet deleted\n")

	return nil
}
