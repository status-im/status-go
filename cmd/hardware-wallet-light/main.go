package main

import (
	"flag"
	"fmt"
	stdlog "log"
	"os"

	"github.com/ebfe/scard"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/smartcard/lightwallet"
)

type commandFunc func(*lightwallet.Installer) error

var (
	logger = log.New("package", "status-go/cmd/hardware-wallet-light")

	commands map[string]commandFunc

	flagCapFile   = flag.String("file", "", "cap file path")
	flagOverwrite = flag.Bool("force", false, "overwrite applet if already installed")
	logLevel      = flag.String("log", "", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
)

func init() {
	if err := logutils.OverrideRootLog(true, "ERROR", "", true); err != nil {
		stdlog.Fatalf("Error initializing logger: %v", err)
	}

	commands = map[string]commandFunc{
		"install": commandInstall,
		"status":  commandStatus,
		"delete":  commandDelete,
	}
}

func usage() {
	fmt.Println("\nUsage: hardware-wallet-light COMMAND [FLAGS]\n\nValid commands:\n")
	for name, _ := range commands {
		fmt.Printf("- %s\n", name)
	}
	fmt.Print("\nFlags:\n\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func fail(msg string, ctx ...interface{}) {
	logger.Error(msg, ctx...)
	os.Exit(1)
}

func main() {
	flag.Parse()
	cmd := flag.Arg(0)
	if cmd == "" {
		logger.Error("you must specify a command (install, status, delete).")
		usage()
	}

	ctx, err := scard.EstablishContext()
	if err != nil {
		fail("error establishing card context", err)
	}
	defer ctx.Release()

	readers, err := ctx.ListReaders()
	if err != nil {
		fail("error getting readers", err)
	}

	if len(readers) == 0 {
		fail("couldn't find any reader")
	}

	if len(readers) > 1 {
		fail("too many readers found")
	}

	reader := readers[0]
	logger.Debug("using reader %s:\n", reader)
	logger.Debug("connecting to card in %s\n", reader)
	card, err := ctx.Connect(reader, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		fail("error connecting to card", err)
	}
	defer card.Disconnect(scard.ResetCard)

	status, err := card.Status()
	if err != nil {
		fail("error getting card status", err)
	}

	switch status.ActiveProtocol {
	case scard.ProtocolT0:
		logger.Debug("Protocol T0\n")
	case scard.ProtocolT1:
		logger.Debug("Protocol T1\n")
	default:
		logger.Debug("Unknown protocol\n")
	}

	i := lightwallet.NewInstaller(card)
	if f, ok := commands[cmd]; ok {
		err = f(i)
		os.Exit(0)
	}

	fail("unknown command %s\n", cmd)
	usage()
}

func commandInstall(i *lightwallet.Installer) error {
	if *flagCapFile == "" {
		logger.Error("you must specify a cap file path with the -f flag\n")
		usage()
	}

	f, err := os.Open(*flagCapFile)
	if err != nil {
		fail("error opening cap file", err)
	}
	defer f.Close()

	secrets, err := i.Install(f, *flagOverwrite)
	if err != nil {
		fail("installation error: ", err)
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
