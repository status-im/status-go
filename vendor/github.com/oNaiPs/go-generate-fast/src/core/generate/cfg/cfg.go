package cfg

import (
	"flag"
	"os"
)

// These are general "build flags" used by build and other commands.
var (
	BuildN bool // -n flag
	BuildV bool // -v flag
	BuildX bool // -x flag
)

func init() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flags.BoolVar(&BuildN, "n", false, "")
	flags.BoolVar(&BuildV, "v", false, "")
	flags.BoolVar(&BuildX, "x", false, "")

	err := flags.Parse(os.Args[1:])
	if err != nil {
		println("ERROR!", err)
	}
	// CommandLine.
}
