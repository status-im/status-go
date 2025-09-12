package plugin_esc

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"go.uber.org/zap"
)

type EscPlugin struct {
	plugins.Plugin
}

func (p *EscPlugin) Name() string {
	return "esc"
}

func (p *EscPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "esc" ||
		opts.GoPackage == "github.com/mjibson/esc"
}

func (p *EscPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {

	conf := parseArgs(opts.SanitizedArgs)
	if conf == nil {
		return nil
	}

	var err error

	ioFiles := plugins.InputOutputFiles{}

	escFiles := []string{}
	var ignoreRegexp *regexp.Regexp
	if conf.Ignore != "" {
		ignoreRegexp, err = regexp.Compile(conf.Ignore)
		if err != nil {
			zap.S().Warn("esc parsing error: %s", err)
			return nil
		}
	}
	var includeRegexp *regexp.Regexp
	if conf.Include != "" {
		includeRegexp, err = regexp.Compile(conf.Include)
		if err != nil {
			zap.S().Warn("esc parsing error: %s", err)
			return nil
		}
	}

	directories := []string{}
	for _, base := range conf.Files {

		files := []string{base}
		for len(files) > 0 {
			fname := files[0]
			files = files[1:]
			if ignoreRegexp != nil && ignoreRegexp.MatchString(fname) {
				continue
			}
			f, err := os.Open(fname)
			if err != nil {
				zap.S().Warn("esc parsing error: %s", err)
				return nil
			}
			fi, err := f.Stat()
			if err != nil {
				zap.S().Warn("esc parsing error: %s", err)
				return nil
			}
			fpath := filepath.ToSlash(fname)
			if fi.IsDir() {
				fis, err := f.Readdir(0)
				if err != nil {
					zap.S().Warn("esc parsing error: %s", err)
					return nil
				}
				for _, fi := range fis {
					childFName := filepath.Join(fname, fi.Name())
					files = append(files, childFName)
					if ignoreRegexp != nil && ignoreRegexp.MatchString(childFName) {
						continue
					}
				}
				directories = append(directories, fpath)
			} else if includeRegexp == nil || includeRegexp.MatchString(fname) {
				escFiles = append(escFiles, fpath)
			}
			f.Close()
		}
	}

	ioFiles.InputFiles = append(ioFiles.InputFiles, escFiles...)

	fakeOutFileName := "static.go"
	if conf.OutputFile != "" {
		fakeOutFileName = conf.OutputFile
	}
	ioFiles.OutputFiles = append(ioFiles.OutputFiles, fakeOutFileName)
	ioFiles.Extra = directories

	return &ioFiles
}

func init() {
	plugins.RegisterPlugin(&EscPlugin{})
}

func New() {
	plugins.RegisterPlugin(&EscPlugin{})
}
