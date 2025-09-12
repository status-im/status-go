package plugin_moq

import (
	"flag"

	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"github.com/oNaiPs/go-generate-fast/src/utils/pkg"
	"go.uber.org/zap"
)

type MoqPlugin struct {
	plugins.Plugin
}

func (p *MoqPlugin) Name() string {
	return "moq"
}

func (p *MoqPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "moq" ||
		opts.GoPackage == "github.com/matryer/moq"
}

type userFlags struct {
	outFile    string
	pkgName    string
	formatter  string
	stubImpl   bool
	skipEnsure bool
	withResets bool
	remove     bool
	args       []string
}

func (p *MoqPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {
	flag := flag.NewFlagSet("mockgen", flag.ContinueOnError)

	var flags userFlags
	flag.StringVar(&flags.outFile, "out", "", "output file (default stdout)")
	flag.StringVar(&flags.pkgName, "pkg", "", "package name (default will infer)")
	flag.StringVar(&flags.formatter, "fmt", "", "go pretty-printer: gofmt, goimports or noop (default gofmt)")
	flag.BoolVar(&flags.stubImpl, "stub", false,
		"return zero values when no mock implementation is provided, do not panic")
	printVersion := flag.Bool("version", false, "show the version for moq")
	flag.BoolVar(&flags.skipEnsure, "skip-ensure", false,
		"suppress mock implementation check, avoid import cycle if mocks generated outside of the tested package")
	flag.BoolVar(&flags.remove, "rm", false, "first remove output file, if it exists")
	flag.BoolVar(&flags.withResets, "with-resets", false,
		"generate functions to facilitate resetting calls made to a mock")

	err := flag.Parse(opts.SanitizedArgs)
	if err != nil {
		zap.S().Debug("cannot parse args")
		return nil
	}
	flags.args = flag.Args()

	if *printVersion {
		return nil
	}

	if flags.outFile == "" {
		zap.S().Error("unsupported stdout mode")
		return nil
	}

	ioFiles := plugins.InputOutputFiles{}

	pkg := pkg.LoadPackages(flags.args[0], []string{}, []string{})
	if pkg == nil {
		//did not find package matches
		return nil
	} else if len(pkg.Errors) > 0 {
		zap.S().Debug("Got errors parsing packages: %s", pkg.Errors)
		return nil
	}

	ioFiles.InputFiles = append(ioFiles.InputFiles, pkg.CompiledGoFiles...)

	ioFiles.OutputFiles = append(ioFiles.OutputFiles, flags.outFile)

	return &ioFiles
}

func init() {
	plugins.RegisterPlugin(&MoqPlugin{})
}
