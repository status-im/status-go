package plugin_stringer

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"github.com/oNaiPs/go-generate-fast/src/utils/fs"
	"github.com/oNaiPs/go-generate-fast/src/utils/pkg"
	"go.uber.org/zap"
)

type StringerPlugin struct {
	plugins.Plugin
}

func (p *StringerPlugin) Name() string {
	return "stringer"
}

func (p *StringerPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "stringer" ||
		opts.GoPackage == "golang.org/x/tools/cmd/stringer"
}

type Flags struct {
	TypeNames   string
	Output      string
	TrimPrefix  string
	LineComment bool
	BuildTags   string
}

func (p *StringerPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {
	// parsing code inspired by https://github.com/golang/tools/blob/master/cmd/stringer/stringer.go
	flagSet := flag.NewFlagSet("stringer", flag.ContinueOnError)

	flags := Flags{}
	flagSet.StringVar(&flags.TypeNames, "type", "", "comma-separated list of type names; must be set")
	flagSet.StringVar(&flags.Output, "output", "", "output file name; default srcdir/<type>_string.go")
	flagSet.StringVar(&flags.TrimPrefix, "trimprefix", "", "trim the `prefix` from the generated constant names")
	flagSet.BoolVar(&flags.LineComment, "linecomment", false, "use line comment text as printed text when present")
	flagSet.StringVar(&flags.BuildTags, "tags", "", "comma-separated list of build tags to apply")

	err := flagSet.Parse(opts.SanitizedArgs)
	if err != nil {
		zap.S().Warn("Cannot parse stringer arguments: ", err.Error())
		return nil
	}

	ioFiles := plugins.InputOutputFiles{}

	args := flagSet.Args()
	tags := []string{}
	if len(flags.BuildTags) > 0 {
		tags = strings.Split(flags.BuildTags, ",")
	}

	// We accept either one directory or a list of files. Which do we have?
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	// Parse the package once.
	var dir string
	if len(args) == 0 {
		dir = opts.Dir()
	} else if len(args) == 1 && fs.IsDir(args[0]) {
		dir = args[0]
	} else {
		if len(tags) != 0 {
			zap.S().Fatal("-tags option applies only to directories, not when files are specified")
		}
		dir = filepath.Dir(args[0])
	}

	pkg := pkg.LoadPackages(dir, args, tags)
	if pkg == nil {
		//did not find package matches
		return nil
	} else if len(pkg.Errors) > 0 {
		zap.S().Debug("Got errors parsing packages: %s", pkg.Errors)
		return nil
	}

	ioFiles.InputFiles = append(ioFiles.InputFiles, pkg.CompiledGoFiles...)

	types := strings.Split(flags.TypeNames, ",")
	outputName := flags.Output

	if outputName == "" {
		outputName = strings.ToLower(fmt.Sprintf("%s_string.go", types[0]))
	}
	ioFiles.OutputFiles = append(ioFiles.OutputFiles, outputName)

	return &ioFiles
}

func init() {
	plugins.RegisterPlugin(&StringerPlugin{})
}
