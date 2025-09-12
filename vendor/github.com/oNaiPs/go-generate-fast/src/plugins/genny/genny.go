package plugin_genny

import (
	"flag"
	"strings"

	"github.com/cheekybits/genny/parse"
	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"go.uber.org/zap"
)

type GennyPlugin struct {
	plugins.Plugin
}

func (p *GennyPlugin) Name() string {
	return "genny"
}

func (p *GennyPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "genny" ||
		opts.GoPackage == "github.com/cheekybits/genny"
}

func (p *GennyPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {
	ioFiles := plugins.InputOutputFiles{}

	flag := flag.NewFlagSet("Genny", flag.ContinueOnError)

	var (
		in  = flag.String("in", "", "file to parse instead of stdin")
		out = flag.String("out", "", "file to save output to instead of stdout")
		_   = flag.String("pkg", "", "package name for generated files")
		_   = flag.String("tag", "", "build tag that is stripped from output")
		_   = "https://github.com/metabition/gennylib/raw/master/"
	)
	err := flag.Parse(opts.SanitizedArgs)
	if err != nil {
		zap.S().Debug("cannot parse args")
		return nil
	}
	args := flag.Args()

	if len(args) < 2 {
		zap.S().Debug("invalid args")
		return nil
	}

	if strings.ToLower(args[0]) != "gen" && strings.ToLower(args[0]) != "get" {
		zap.S().Debug("invalid args")
		return nil
	}

	// parse the typesets
	var setsArg = args[1]
	if strings.ToLower(args[0]) == "get" {
		setsArg = args[2]
	}
	_, err = parse.TypeSet(setsArg)
	if err != nil {
		zap.S().Debug("invalid type set")
		return nil
	}

	if *out == "" {
		zap.S().Debug("invalid output filename")
		return nil
	}
	ioFiles.OutputFiles = append(ioFiles.OutputFiles, *out)

	if strings.ToLower(args[0]) == "get" {
		zap.S().Debug("get command is not supported")
		return nil
	} else if len(*in) > 0 {
		ioFiles.InputFiles = append(ioFiles.InputFiles, *in)
	} else {
		zap.S().Debug("unsupported read from stdin")
		return nil
	}

	return &ioFiles
}

func init() {
	plugins.RegisterPlugin(&GennyPlugin{})
}
