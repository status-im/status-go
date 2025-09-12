package plugin_go_bindata

import (
	"github.com/go-bindata/go-bindata"
	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"go.uber.org/zap"
)

type GobindataPlugin struct {
	plugins.Plugin
}

func (p *GobindataPlugin) Name() string {
	return "go-bindata"
}

func (p *GobindataPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "go-bindata" ||
		opts.GoPackage == "github.com/go-bindata/go-bindata/..."
}

func (p *GobindataPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {
	ioFiles := plugins.InputOutputFiles{}

	cfg := parseArgs(opts.SanitizedArgs)
	if cfg == nil {
		// there's no config returned, so most likely the original command will fail
		return nil
	}

	var toc []bindata.Asset
	var knownFuncs = make(map[string]int)
	var visitedPaths = make(map[string]bool)
	for _, input := range cfg.Input {
		err := findFiles(input.Path, cfg.Prefix, input.Recursive, &toc, cfg.Ignore, knownFuncs, visitedPaths)
		if err != nil {
			zap.S().Error("go-bindata: cannot find files: %s", err)
			return nil
		}
	}

	for _, asset := range toc {
		ioFiles.InputFiles = append(ioFiles.InputFiles, asset.Path)
	}

	ioFiles.OutputFiles = append(ioFiles.OutputFiles, cfg.Output)

	return &ioFiles
}
func init() {
	plugins.RegisterPlugin(&GobindataPlugin{})
}
