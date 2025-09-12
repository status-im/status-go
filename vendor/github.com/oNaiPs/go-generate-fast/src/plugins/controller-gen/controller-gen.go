package plugin_ControllerGen

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/oNaiPs/go-generate-fast/src/core/golist"
	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/deepcopy"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/markers"
	"sigs.k8s.io/controller-tools/pkg/rbac"
	"sigs.k8s.io/controller-tools/pkg/schemapatcher"
	"sigs.k8s.io/controller-tools/pkg/webhook"
)

type ControllerGenPlugin struct {
	plugins.Plugin
}

func (p *ControllerGenPlugin) Name() string {
	return "controller-gen"
}

func (p *ControllerGenPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "controller-gen" ||
		opts.GoPackage == "sigs.k8s.io/controller-tools/cmd/controller-gen"
}

// got input options from https://github.com/kubernetes-sigs/controller-tools/blob/master/cmd/controller-gen/main.go
var (
	allGenerators = map[string]genall.Generator{
		"crd":         crd.Generator{},
		"rbac":        rbac.Generator{},
		"object":      deepcopy.Generator{},
		"webhook":     webhook.Generator{},
		"schemapatch": schemapatcher.Generator{},
	}
	allOutputRules = map[string]genall.OutputRule{
		"dir":       genall.OutputToDirectory(""),
		"none":      genall.OutputToNothing,
		"stdout":    genall.OutputToStdout,
		"artifacts": genall.OutputArtifacts{},
	}
	optionsRegistry = &markers.Registry{}
)

func init() {
	for genName, gen := range allGenerators {
		// make the generator options marker itself
		defn := markers.Must(markers.MakeDefinition(genName, markers.DescribesPackage, gen))
		if err := optionsRegistry.Register(defn); err != nil {
			panic(err)
		}
		// make per-generation output rule markers
		for ruleName, rule := range allOutputRules {
			ruleMarker := markers.Must(markers.MakeDefinition(fmt.Sprintf("output:%s:%s", genName, ruleName), markers.DescribesPackage, rule))
			if err := optionsRegistry.Register(ruleMarker); err != nil {
				panic(err)
			}
		}
	}

	// make "default output" output rule markers
	for ruleName, rule := range allOutputRules {
		ruleMarker := markers.Must(markers.MakeDefinition("output:"+ruleName, markers.DescribesPackage, rule))
		if err := optionsRegistry.Register(ruleMarker); err != nil {
			panic(err)
		}
	}
	// add in the common options markers
	if err := genall.RegisterOptionsMarkers(optionsRegistry); err != nil {
		panic(err)
	}
}

func (p *ControllerGenPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {
	ioFiles := plugins.InputOutputFiles{}

	inputPaths := []string{"./..."}

	for _, rawOpt := range opts.SanitizedArgs {
		if rawOpt[0] != '+' {
			rawOpt = "+" + rawOpt // add a `+` to make it acceptable for usage with the registry
		}
		defn := optionsRegistry.Lookup(rawOpt, markers.DescribesPackage)
		if defn == nil {
			zap.S().Errorf("unknown option %q", rawOpt[1:])
			return nil
		}

		val, err := defn.Parse(rawOpt)
		if err != nil {
			zap.S().Errorf("unable to parse option %q: %w", rawOpt[1:], err)
			return nil
		}

		switch val := val.(type) {
		case crd.Generator:
			if val.HeaderFile != "" {
				ioFiles.InputFiles = append(ioFiles.InputFiles, val.HeaderFile)
			}
		case rbac.Generator:
			if val.HeaderFile != "" {
				ioFiles.InputFiles = append(ioFiles.InputFiles, val.HeaderFile)
			}
		case deepcopy.Generator:
			if val.HeaderFile != "" {
				ioFiles.InputFiles = append(ioFiles.InputFiles, val.HeaderFile)
			}
		case webhook.Generator:
			if val.HeaderFile != "" {
				ioFiles.InputFiles = append(ioFiles.InputFiles, val.HeaderFile)
			}
		case schemapatcher.Generator:
			dirEntries, err := os.ReadDir(val.ManifestsPath)
			if err != nil {
				zap.S().Errorw("cannot ready manifests path: %w", err)
				return nil
			}
			for _, fileInfo := range dirEntries {
				if fileInfo.IsDir() || filepath.Ext(fileInfo.Name()) != ".yaml" {
					continue
				}
				ioFiles.InputFiles = append(ioFiles.InputFiles, path.Join(val.ManifestsPath, fileInfo.Name()))
			}

		case genall.OutputToDirectory:
			ioFiles.OutputPatterns = append(ioFiles.OutputPatterns, path.Join(string(val), "**"))
		case genall.OutputArtifacts:
			ioFiles.OutputPatterns = append(ioFiles.OutputPatterns, path.Join(string(val.Config), "**"))
			if val.Code != "" {
				ioFiles.OutputPatterns = append(ioFiles.OutputPatterns, path.Join(string(val.Code), "**"))
			}
		case genall.InputPaths:
			inputPaths = val
		default:
			zap.S().Errorf("unknown option marker %q", defn.Name)
			return nil
		}
	}

	for _, pkg := range golist.ModulesAndErrors(inputPaths) {
		if pkg.Error != nil {
			zap.S().Errorf("cannot get input path: ", pkg.Error)
			continue
		}

		ioFiles.InputFiles = append(ioFiles.InputFiles, pkg.Package)
	}

	return &ioFiles
}

func init() {
	plugins.RegisterPlugin(&ControllerGenPlugin{})
}
