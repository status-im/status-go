package plugins

import (
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

type GenerateOpts struct {
	// full path name of the file where the command is being run.
	Path string
	// all the words being passed on the generate macro
	Words []string
	// name of the executable being run. if it is a go package, it will be the name of the cmd tool
	ExecutableName string
	// full path of the command being run. if it is a go package, this path will be an empty string
	ExecutablePath string
	// when this command is a "go run [pkg]" command, the name of the package being run
	GoPackage string
	// when this command is a "go run [pkg]@version" command, the version specified on it (e.g. 1.2.3, latest). Empty string when not specified.
	GoPackageVersion string
	// arguments being passed to the target executable.
	// examples:
	// [exec] -a -b arg -> ["-a", "-b", "arg"]
	// go run [pkg] -a -b arg -> ["-a", "-b", "cmd"]
	SanitizedArgs []string
	// optionally added input files before the command
	ExtraInputPatterns []string
	// optionally added output files before the command
	ExtraOutputPatterns []string
}

// base name of the file containing the command.
func (g *GenerateOpts) File() string {
	return filepath.Base(g.Path)
}

// full dir where the command is being run on. this is the os.Getwd() when the plugins are running
func (g *GenerateOpts) Dir() string {
	return filepath.Dir(g.Path)
}

func (g *GenerateOpts) Command() string {
	return strings.Join(g.Words, " ")
}

type Plugin interface {
	Name() string
	Matches(opts GenerateOpts) bool
	ComputeInputOutputFiles(opts GenerateOpts) *InputOutputFiles
}

type InputOutputFiles struct {
	InputFiles  []string
	OutputFiles []string
	// output file patterns - will be computed after command is run
	OutputPatterns []string
	// extra data that is used in the hash calculations
	Extra []string
}

var PluginsMap = make(map[string]Plugin)

func RegisterPlugin(plugin Plugin) {
	if PluginsMap[plugin.Name()] != nil {
		zap.S().Panic("Already registered plugin with name: ", plugin.Name())
	}

	PluginsMap[plugin.Name()] = plugin
}

func ClearPlugins() {
	PluginsMap = make(map[string]Plugin)
}

func MatchPlugin(opts GenerateOpts) Plugin {
	zap.S().Debugf("Matching %d plugins for command \"%s\"", len(PluginsMap), strings.Join(opts.Words, " "))

	for _, plugin := range PluginsMap {
		if plugin.Matches(opts) {
			zap.S().Debug("Matched ", plugin.Name())
			return plugin
		}
	}
	return nil
}
