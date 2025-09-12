package plugin_mockgen

import (
	"flag"
	"strings"

	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"github.com/oNaiPs/go-generate-fast/src/utils/pkg"
	"go.uber.org/zap"
)

type MockgenPlugin struct {
	plugins.Plugin
}

func (p *MockgenPlugin) Name() string {
	return "mockgen"
}

func (p *MockgenPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "mockgen" ||
		opts.GoPackage == "go.uber.org/mock/mockgen" ||
		opts.GoPackage == "github.com/golang/mock/mockgen"
}

type MockgenFlags struct {
	Source                 string
	Destination            string
	MockNames              string
	PackageOut             string
	SelfPackage            string
	WritePkgComment        bool
	WriteSourceComment     bool
	WriteGenerateDirective bool
	CopyrightFile          string
	Typed                  bool
	Imports                string
	AuxFiles               string
	ExcludeInterfaces      string

	DebugParser bool
	ShowVersion bool
}

func (p *MockgenPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {
	flagSet := flag.NewFlagSet("mockgen", flag.ContinueOnError)

	flags := MockgenFlags{}
	flagSet.StringVar(&flags.Source, "source", "", "(source mode) Input Go source file; enables source mode.")
	flagSet.StringVar(&flags.Destination, "destination", "", "Output file; defaults to stdout.")
	flagSet.StringVar(&flags.MockNames, "mock_names", "", "Comma-separated interfaceName=mockName pairs of explicit mock names to use. Mock names default to 'Mock'+ interfaceName suffix.")
	flagSet.StringVar(&flags.PackageOut, "package", "", "Package of the generated code; defaults to the package of the input with a 'mock_' prefix.")
	flagSet.StringVar(&flags.SelfPackage, "self_package", "", "The full package import path for the generated code. The purpose of this flag is to prevent import cycles in the generated code by trying to include its own package. This can happen if the mock's package is set to one of its inputs (usually the main one) and the output is stdio so mockgen cannot detect the final output package. Setting this flag will then tell mockgen which import to exclude.")
	flagSet.BoolVar(&flags.WritePkgComment, "write_package_comment", true, "Writes package documentation comment (godoc) if true.")
	flagSet.BoolVar(&flags.WriteSourceComment, "write_source_comment", true, "Writes original file (source mode) or interface names (reflect mode) comment if true.")
	flagSet.BoolVar(&flags.WriteGenerateDirective, "write_generate_directive", false, "Add //go:generate directive to regenerate the mock")
	flagSet.StringVar(&flags.CopyrightFile, "copyright_file", "", "Copyright file used to add copyright header")
	flagSet.BoolVar(&flags.Typed, "typed", false, "Generate Type-safe 'Return', 'Do', 'DoAndReturn' function")
	flagSet.StringVar(&flags.Imports, "imports", "", "(source mode) Comma-separated name=path pairs of explicit imports to use.")
	flagSet.StringVar(&flags.AuxFiles, "aux_files", "", "(source mode) Comma-separated pkg=path pairs of auxiliary Go source files.")
	flagSet.StringVar(&flags.ExcludeInterfaces, "exclude_interfaces", "", "Comma-separated names of interfaces to be excluded")

	flagSet.BoolVar(&flags.DebugParser, "debug_parser", false, "Print out parser results only.")
	flagSet.BoolVar(&flags.ShowVersion, "version", false, "Print version.")

	err := flagSet.Parse(opts.SanitizedArgs)
	if err != nil {
		zap.S().Warn("Cannot parse Mockgen arguments: ", err.Error())
		return nil
	}

	ioFiles := plugins.InputOutputFiles{}

	auxFiles := strings.Split(flags.AuxFiles, ",")
	for _, aux := range auxFiles {
		pair := strings.Split(aux, "=")
		if len(pair) != 2 {
			continue
		}
		ioFiles.InputFiles = append(ioFiles.InputFiles, pair[1])
	}

	if flags.CopyrightFile != "" {
		ioFiles.InputFiles = append(ioFiles.InputFiles, flags.CopyrightFile)
	}

	if flags.Destination != "" {
		ioFiles.OutputFiles = append(ioFiles.OutputFiles, flags.Destination)
	}

	imports := strings.Split(flags.Imports, ",")
	for _, imp := range imports {
		pair := strings.Split(imp, "=")
		if len(pair) != 2 {
			continue
		}
		ioFiles.InputFiles = append(ioFiles.InputFiles, pair[1])
	}

	if flags.Source != "" {
		ioFiles.InputFiles = append(ioFiles.InputFiles, flags.Source)
	}

	if len(flagSet.Args()) == 2 {

		pkg := pkg.LoadPackages(opts.Dir(), []string{flagSet.Args()[0]}, []string{})
		if pkg == nil {
			//did not find package matches
			return nil
		} else if len(pkg.Errors) > 0 {
			zap.S().Debug("Got errors parsing packages: %s", pkg.Errors)
			return nil
		}

		ioFiles.InputFiles = append(ioFiles.InputFiles, pkg.CompiledGoFiles...)
	}

	return &ioFiles
}

func init() {
	plugins.RegisterPlugin(&MockgenPlugin{})
}
