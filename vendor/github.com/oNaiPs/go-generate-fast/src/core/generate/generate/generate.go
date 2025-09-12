// oNaiPs 2023 - file inspired from
// https://github.com/golang/go/blob/master/src/cmd/go/internal/generate/generate.go

// Package generate implements the “go generate” command.
package generate

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/oNaiPs/go-generate-fast/src/core/cache"
	"github.com/oNaiPs/go-generate-fast/src/core/config"
	"github.com/oNaiPs/go-generate-fast/src/core/generate/base"
	"github.com/oNaiPs/go-generate-fast/src/core/generate/cfg"
	"github.com/oNaiPs/go-generate-fast/src/core/golist"
	"github.com/oNaiPs/go-generate-fast/src/plugins"
	"github.com/oNaiPs/go-generate-fast/src/utils/fs"
	"github.com/oNaiPs/go-generate-fast/src/utils/str"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var (
	generateRunFlag string         // generate -run flag
	generateRunRE   *regexp.Regexp // compiled expression for -run

	generateSkipFlag string         // generate -skip flag
	generateSkipRE   *regexp.Regexp // compiled expression for -skip
)

func RunGenerate(args []string) {
	if generateRunFlag != "" {
		var err error
		generateRunRE, err = regexp.Compile(generateRunFlag)
		if err != nil {
			log.Fatalf("generate: %s", err)
		}
	}
	if generateSkipFlag != "" {
		var err error
		generateSkipRE, err = regexp.Compile(generateSkipFlag)
		if err != nil {
			log.Fatalf("generate: %s", err)
		}
	}

	for _, pkg := range golist.ModulesAndErrors(args) {

		if pkg.Error != nil {
			fmt.Println(*pkg.Error)
			continue
		}

		if !generate(pkg.Package) {
			break
		}
	}
}

// generate runs the generation directives for a single file.
func generate(absFile string) bool {
	src, err := os.ReadFile(absFile)
	if err != nil {
		log.Fatalf("generate: %s", err)
	}

	// Parse package clause
	filePkg, err := parser.ParseFile(token.NewFileSet(), "", src, parser.PackageClauseOnly)
	if err != nil {
		// Invalid package clause - ignore file.
		return true
	}

	g := &Generator{
		r:        bytes.NewReader(src),
		path:     absFile,
		pkg:      filePkg.Name.String(),
		commands: make(map[string][]string),
	}

	return g.run()
}

// A Generator represents the state of a single Go source file
// being scanned for generator commands.
type Generator struct {
	r        io.Reader
	path     string
	pkg      string
	commands map[string][]string
	lineNum  int // current line number.
	env      []string
}

// run runs the generators in the current file.
func (g *Generator) run() (ok bool) {
	// Processing below here calls g.errorf on failure, which does panic(stop).
	// If we encounter an error, we abort the package.
	defer func() {
		e := recover()
		if e != nil {
			ok = false
			if e != stop {
				panic(e)
			}
			base.SetExitStatus(1)
		}
	}()
	if cfg.BuildV {
		zap.S().Debug(g.path)
		// fmt.Fprintf(os.Stderr, "%s\n", g.path)
	}

	// Scan for lines that start "//go:generate".
	// Can't use bufio.Scanner because it can't handle long lines,
	// which are likely to appear when using generate.
	input := bufio.NewReader(g.r)

	ExtraInputPatterns := []string{}
	ExtraOutputPatterns := []string{}
	var err error
	// One line per loop.
	for {
		g.lineNum++ // 1-indexed.
		var buf []byte
		buf, err = input.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			// Line too long - consume and ignore.
			if isGoGenerate(buf) {
				g.errorf("directive too long")
			}

			for err == bufio.ErrBufferFull {
				_, err = input.ReadSlice('\n')
			}
			if err != nil {
				break
			}
			continue
		}

		if err != nil {
			// Check for marker at EOF without final \n.
			if err == io.EOF && isGoGenerate(buf) {
				err = io.ErrUnexpectedEOF
			}
			break
		}

		if !isGoGenerate(buf) {
			continue
		} else if isGoGenerateExtraInput(buf) {
			words := g.split(string(buf), len("//go:generate_input "))
			ExtraInputPatterns = append(ExtraInputPatterns, words...)
			continue
		} else if isGoGenerateExtraOutput(buf) {
			words := g.split(string(buf), len("//go:generate_output "))
			ExtraOutputPatterns = append(ExtraOutputPatterns, words...)
			continue
		} else if !isGoGenerateCommand(buf) {
			continue
		}

		words := g.split(string(buf), len("//go:generate "))
		opts := plugins.GenerateOpts{
			Path:                g.path,
			Words:               words,
			ExtraInputPatterns:  ExtraInputPatterns,
			ExtraOutputPatterns: ExtraOutputPatterns,
		}

		ExtraInputPatterns = []string{}
		ExtraOutputPatterns = []string{}

		if generateRunFlag != "" && !generateRunRE.Match(bytes.TrimSpace(buf)) {
			continue
		}
		if generateSkipFlag != "" && generateSkipRE.Match(bytes.TrimSpace(buf)) {
			continue
		}

		g.setEnv()
		if len(words) == 0 {
			g.errorf("no arguments to directive")
		}
		if words[0] == "-command" {
			g.setShorthand(words)
			continue
		}
		// Run the command line.
		if cfg.BuildN || cfg.BuildX {
			zap.S().Info(opts.Command())
		}
		if cfg.BuildN {
			continue
		}

		var cacheResult cache.VerifyResult

		var runGenerate bool
		var runSave bool
		var runRestore bool
		cachedInfo := []string{}

		start := time.Now()

		executablePath, err := fs.FindExecutablePath(words[0])
		if err == nil {
			opts.ExecutablePath = executablePath
		} else {
			zap.S().Debugf("Cannot find executable path for %s", words[0])
			opts.ExecutablePath = words[0]
		}

		if !maybeParseGoRunCommand(&opts) {
			opts.ExecutableName = filepath.Base(words[0])
			opts.SanitizedArgs = words[1:]
		}

		cwd, err := os.Getwd()
		if err != nil {
			zap.S().Fatal("cannot get working directory: %s", err)
		}

		// change to the command being run directory, so plugins can work with relative paths to it
		err = os.Chdir(opts.Dir())
		if err != nil {
			zap.S().Fatal("cannot chdir to command directory: %s", err)
		}

		if config.Get().Disable {
			runGenerate = true
		} else {

			cacheResult, err = cache.Verify(opts)

			if err != nil {
				zap.S().Debugf("cannot verify cache: %s", err)
				// cannot verify cache, cannot use save/restore functions
				runGenerate = true
			} else if config.Get().ReCache {
				runGenerate = true
				runSave = true
			} else {
				runRestore = cacheResult.CacheHit
				runGenerate = !cacheResult.CacheHit
				runSave = !cacheResult.CacheHit
			}
		}

		if runRestore {
			err := cache.Restore(cacheResult)
			if err != nil {
				zap.S().Errorf("cannot restore cache: %s", err)
				runGenerate = true
				runSave = true
			} else {
				cachedInfo = append(cachedInfo, "cached")
			}
		}

		if runGenerate {
			if config.Get().ForceUseCache {
				cachedInfo = append(cachedInfo, "force_use_cache")
				base.SetExitStatus(1)
			} else {
				if g.exec(opts) {
					cachedInfo = append(cachedInfo, "generated")
				} else {
					runSave = false
					cachedInfo = append(cachedInfo, "error")
					base.SetExitStatus(1)
				}
			}
		}

		if runSave &&
			cacheResult.CanSave &&
			!config.Get().ReadOnly &&
			!config.Get().ForceUseCache {
			err := cache.Save(cacheResult)
			if err != nil {
				zap.S().Errorf("cannot save cache: %s", err)
			} else {
				cachedInfo = append(cachedInfo, "saved")
			}
		}

		if config.Get().Disable {
			cachedInfo = append(cachedInfo, "disabled")
		} else if cacheResult.PluginMatch == nil {
			cachedInfo = append(cachedInfo, "noplugin")
		}

		cachedInfo = append(cachedInfo, fmt.Sprintf("%dms", time.Since(start).Milliseconds()))

		err = os.Chdir(cwd)
		if err != nil {
			zap.S().Fatal("cannot restore working directory: %s", err)
		}

		relPath, err := filepath.Rel(cwd, g.path)
		if err != nil {
			zap.S().Fatal("cannot compute relative dir: %s", err)
		}
		zap.S().Infof("%s: %s (%s)", relPath, opts.Command(), strings.Join(cachedInfo, ", "))

	}

	if err != nil && err != io.EOF {
		g.errorf("error reading %s: %s", g.path, err)
	}
	return true
}

func isGoGenerate(buf []byte) bool {
	return bytes.HasPrefix(buf, []byte("//go:generate"))
}

func isGoGenerateCommand(buf []byte) bool {
	return bytes.HasPrefix(buf, []byte("//go:generate ")) || bytes.HasPrefix(buf, []byte("//go:generate\t"))
}

func isGoGenerateExtraInput(buf []byte) bool {
	return bytes.HasPrefix(buf, []byte("//go:generate_input "))
}

func isGoGenerateExtraOutput(buf []byte) bool {
	return bytes.HasPrefix(buf, []byte("//go:generate_output "))
}

// setEnv sets the extra environment variables used when executing a
// single go:generate command.
func (g *Generator) setEnv() {
	env := []string{
		"GOROOT=" + runtime.GOROOT(),
		"GOARCH=" + runtime.GOARCH,
		"GOOS=" + runtime.GOOS,
		"GOFILE=" + filepath.Base(g.path),
		"GOLINE=" + strconv.Itoa(g.lineNum),
		"GOPACKAGE=" + g.pkg,
		"DOLLAR=" + "$",
	}
	env = base.AppendPATH(env)
	env = base.AppendPWD(env, filepath.Dir(g.path))
	g.env = env
}

// split breaks the line into words, evaluating quoted
// strings and evaluating environment variables.
// The initial //go:generate element is present in line.
func (g *Generator) split(line string, stripPrefixLen int) []string {
	// Parse line, obeying quoted strings.
	var words []string
	line = line[stripPrefixLen : len(line)-1] // Drop preamble and final newline.
	// There may still be a carriage return.
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	// One (possibly quoted) word per iteration.
Words:
	for {
		line = strings.TrimLeft(line, " \t")
		if len(line) == 0 {
			break
		}
		if line[0] == '"' {
			for i := 1; i < len(line); i++ {
				c := line[i] // Only looking for ASCII so this is OK.
				switch c {
				case '\\':
					if i+1 == len(line) {
						g.errorf("bad backslash")
					}
					i++ // Absorb next byte (If it's a multibyte we'll get an error in Unquote).
				case '"':
					word, err := strconv.Unquote(line[0 : i+1])
					if err != nil {
						g.errorf("bad quoted string")
					}
					words = append(words, word)
					line = line[i+1:]
					// Check the next character is space or end of line.
					if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
						g.errorf("expect space after quoted argument")
					}
					continue Words
				}
			}
			g.errorf("mismatched quoted string")
		}
		i := strings.IndexAny(line, " \t")
		if i < 0 {
			i = len(line)
		}
		words = append(words, line[0:i])
		line = line[i:]
	}
	// Substitute command if required.
	if len(words) > 0 && g.commands[words[0]] != nil {
		// Replace 0th word by command substitution.
		//
		// Force a copy of the command definition to
		// ensure words doesn't end up as a reference
		// to the g.commands content.
		tmpCmdWords := append([]string(nil), (g.commands[words[0]])...)
		words = append(tmpCmdWords, words[1:]...)
	}
	// Substitute environment variables.
	for i, word := range words {
		words[i] = os.Expand(word, g.expandVar)
	}
	return words
}

var stop = fmt.Errorf("error in generation")

// errorf logs an error message prefixed with the file and line number.
// It then exits the program (with exit status 1) because generation stops
// at the first error.
func (g *Generator) errorf(format string, args ...any) {
	zap.S().Errorf("%s:%d: %s\n", g.path, g.lineNum, fmt.Sprintf(format, args...))
	panic(stop)
}

// expandVar expands the $XXX invocation in word. It is called
// by os.Expand.
func (g *Generator) expandVar(word string) string {
	w := word + "="
	for _, e := range g.env {
		if strings.HasPrefix(e, w) {
			return e[len(w):]
		}
	}
	return os.Getenv(word)
}

// setShorthand installs a new shorthand as defined by a -command directive.
func (g *Generator) setShorthand(words []string) {
	// Create command shorthand.
	if len(words) == 1 {
		g.errorf("no command specified for -command")
	}
	command := words[1]
	if g.commands[command] != nil {
		g.errorf("command %q multiply defined", command)
	}
	g.commands[command] = slices.Clip(words[2:])
}

// exec runs the command specified by the argument. The first word is
// the command name itself.
func (g *Generator) exec(opts plugins.GenerateOpts) bool {
	cmd := exec.Command(opts.ExecutablePath, opts.Words[1:]...)
	cmd.Args[0] = opts.ExecutablePath // Overwrite with the original in case it was rewritten above.

	// Standard in and out of generator should be the usual.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Run the command in the package directory.
	cmd.Dir = opts.Dir()
	cmd.Env = str.StringList(os.Environ(), g.env)
	err := cmd.Run()
	if err != nil {
		zap.S().Errorf("running %q: %s", opts.Command(), err)
		return false
	}
	return true
}

func maybeParseGoRunCommand(opts *plugins.GenerateOpts) bool {

	if opts.Words[0] != "go" || opts.Words[1] != "run" {
		return false
	}
	flagSet := flag.NewFlagSet("go run", flag.ContinueOnError)

	err := flagSet.Parse(opts.Words)
	if err != nil {
		zap.S().Warn("Cannot parse go run arguments: ", err.Error())
		return false
	}

	if len(flagSet.Args()) < 3 {
		zap.S().Debugf("Cannot parse go run command with <2 args")
		return false
	}

	packageAndVersion := strings.Split(flagSet.Args()[2], "@")

	opts.GoPackage = packageAndVersion[0]
	if len(packageAndVersion) > 1 {
		opts.GoPackageVersion = packageAndVersion[1]
	}
	opts.SanitizedArgs = flagSet.Args()[3:]

	return true
}
