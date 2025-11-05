package analyzer

import (
	"flag"
	"io"
	"os"
	"path"
	"strings"
)

type Config struct {
	SkipDir string
}

var workdir string

func init() {
	var err error
	workdir, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

func (c *Config) ParseFlags() (flag.FlagSet, error) {
	flags := flag.NewFlagSet("goroutine-defer-guard", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Otherwise errors are printed to stderr
	flags.StringVar(&c.SkipDir, "skip", "", "skip paths with this prefix")

	// We parse the flags here to have the config before the call to `singlechecker.Main(analyzer)`
	// For same reasons we discard the output and skip the undefined flag error.
	err := flags.Parse(os.Args[1:])
	if err == nil {
		return *flags, nil
	}

	if strings.Contains(err.Error(), "flag provided but not defined") {
		err = nil
	} else if strings.Contains(err.Error(), "help requested") {
		err = nil
	}

	return *flags, err
}

func (c *Config) WithAbsolutePaths() *Config {
	out := *c

	if out.SkipDir != "" && !path.IsAbs(out.SkipDir) {
		out.SkipDir = path.Join(workdir, out.SkipDir)
	}

	return &out
}
