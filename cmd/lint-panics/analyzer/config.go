package analyzer

import (
	"flag"
	"io"
	"os"
	"path"
	"strings"
)

type Config struct {
	RootDir string
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
	flags := flag.NewFlagSet("lint-panics", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Otherwise errors are printed to stderr
	flags.StringVar(&c.RootDir, "root", workdir, "root directory to run gopls")
	flags.StringVar(&c.SkipDir, "skip", "", "skip paths with this suffix")

	// We parse the flags here to have `rootDir` before the call to `singlechecker.Main(analyzer)`
	// For same reasons we discard the output and skip the undefined flag error.
	err := flags.Parse(os.Args[1:])
	if err != nil && strings.Contains(err.Error(), "flag provided but not defined") {
		err = nil
	}

	return *flags, err
}

func (c *Config) WithAbsolutePaths() *Config {
	out := *c

	if !path.IsAbs(out.RootDir) {
		out.RootDir = path.Join(workdir, out.RootDir)
	}

	if out.SkipDir != "" && !path.IsAbs(out.SkipDir) {
		out.SkipDir = path.Join(out.RootDir, out.SkipDir)
	}

	return &out
}
