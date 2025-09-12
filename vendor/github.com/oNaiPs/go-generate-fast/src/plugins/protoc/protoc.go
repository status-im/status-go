package plugin_protoc

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"go.uber.org/zap"
)

type ProtocPlugin struct {
	plugins.Plugin
}

func (p *ProtocPlugin) Name() string {
	return "protoc"
}

func (p *ProtocPlugin) Matches(opts plugins.GenerateOpts) bool {
	return opts.ExecutableName == "protoc"
}

type ProtocParsedFlags struct {
	Include []string `short:"I" long:"proto_path"`
	GoOut   string   `long:"go_out"`
	GoOpt   []string `long:"go_opt"`
	Plugin  []string `long:"plugin"`
}

func (p *ProtocPlugin) ComputeInputOutputFiles(opts plugins.GenerateOpts) *plugins.InputOutputFiles {
	parsedFlags := ProtocParsedFlags{}
	args, err := flags.ParseArgs(&parsedFlags, opts.SanitizedArgs)
	if len(parsedFlags.Include) == 0 {
		// default search path when no include paths are specified is current dir
		parsedFlags.Include = append(parsedFlags.Include, opts.Dir())
	}

	if err != nil {
		panic(err)
	}

	ioFiles := plugins.InputOutputFiles{}

	pathsMode := "import"
	for _, opt := range parsedFlags.GoOpt {
		if strings.HasPrefix(opt, "paths") {
			pathsMode = strings.TrimPrefix(opt, "paths=")
		}
	}

	for index, inputFile := range args {
		if !strings.HasSuffix(inputFile, ".proto") {
			zap.S().Warn("Input file doesn't end with .proto: ", inputFile, ". Skipping")
			continue
		}

		var inputFilePath string
		if !filepath.IsAbs(inputFile) {
			inputFilePath = searchFile(inputFile, parsedFlags.Include, opts.Dir())
			if inputFilePath == "" {
				zap.S().Warn("Cannot find ", inputFile, " in include dirs. Skipping")
				continue
			}
			inputFile = filepath.Base(inputFilePath)
		} else {
			inputFilePath = inputFile
			inputFile = filepath.Base(inputFile)
		}
		inputFileDir := filepath.Dir(inputFilePath)

		ioFiles.InputFiles = append(ioFiles.InputFiles, inputFilePath)

		protoFile, err := parseProtoFile(inputFilePath)
		if err != nil {
			zap.S().Warn("Cannot parse proto file: ", inputFilePath, ". Skipping")
			continue
		}
		outputDir := ""
		switch pathsMode {
		// uses go_package specified in the .proto file
		case "import":
			goPackage := protoFile.GoPackage

			//parse M go_opt (see https://protobuf.dev/reference/go/go-generated/#package)
			for _, opt := range parsedFlags.GoOpt {
				if !strings.HasPrefix(opt, "M") {
					continue
				}

				spl := strings.Split(strings.TrimPrefix(opt, "M"), "=")

				if spl[0] == args[index] {
					goPackage = spl[1]
				}
			}

			//remove possible package_name specification (not supported atm)
			goPackage = strings.Split(goPackage, ";")[0]

			outputDir = filepath.Join(opts.Dir(), goPackage)

		case "source_relative":
			outputDir = inputFileDir
		default:
			zap.S().Fatal("Unknown paths mode ", pathsMode)
		}

		outputFile := path.Join(outputDir, strings.TrimSuffix(inputFile, ".proto")+".pb.go")
		ioFiles.OutputFiles = append(ioFiles.OutputFiles, outputFile)

		for _, importFile := range protoFile.Imports {
			importFilePath := searchFile(importFile, parsedFlags.Include, inputFileDir)
			if importFilePath != "" {
				ioFiles.InputFiles = append(ioFiles.InputFiles, importFilePath)
			} else {
				zap.S().Warn("Cannot find import ", importFile, " in include dirs. Skipping")
			}
		}
	}

	// TODO add @<filename>

	return &ioFiles
}

// searchFile searches for a given file within a list of include directories.
// It returns the full path of the file if found in any of the directories.
// If the file is not found, an empty string is returned.
func searchFile(filePath string, includeDirs []string, baseDir string) string {
	for _, includeDir := range includeDirs {
		if !filepath.IsAbs(includeDir) {
			includeDir = filepath.Join(baseDir, includeDir)
		}
		fullPath := filepath.Join(includeDir, filePath)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}
	return ""
}

type ProtoFile struct {
	Imports   []string
	GoPackage string
}

func parseProtoFile(filePath string) (*ProtoFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	goFile := &ProtoFile{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		importMatch := regexp.MustCompile(`^\s*import\s+"(.*)"`).FindStringSubmatch(line)
		if len(importMatch) > 0 {
			goFile.Imports = append(goFile.Imports, importMatch[1])
		}

		goPackageMatch := regexp.MustCompile(`^\s*option\s+go_package\s*=\s*"(.*)"`).FindStringSubmatch(line)
		if len(goPackageMatch) > 0 {
			goFile.GoPackage = goPackageMatch[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return goFile, nil
}

func init() {
	plugins.RegisterPlugin(&ProtocPlugin{})
}
