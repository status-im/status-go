package main

import (
	"os"
	"bufio"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"fmt"
	"go/ast"
	gotoken "go/token"
	goparser "go/parser"
)

type Processor struct {
	logger   *zap.Logger
	fset     *gotoken.FileSet
	language LanguageInterface
}

type LanguageInterface interface {
	Definition(string, int, int) (string, int, error)
}

func NewParser(logger *zap.Logger, language LanguageInterface) *Processor {
	return &Processor{
		logger:   logger.Named("parser"),
		fset:     gotoken.NewFileSet(),
		language: language,
	}
}

func (p *Processor) Run(path string) ([]string, error) {
	logger := p.logger.With(zap.String("file", path))
	//logger.Debug("scanning file")

	src, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to open file", zap.Error(err))
	}

	file, err := goparser.ParseFile(p.fset, path, src, 0)
	if err != nil {
		logger.Error("failed to parse file", zap.Error(err))
		return nil, err
	}

	// Traverse the AST to find goroutines
	ast.Inspect(file, p.processNode)

	return nil, nil
}

func (p *Processor) processNode(n ast.Node) bool {
	// Check if the node is a GoStmt (which represents a 'go' statement)
	goStmt, ok := n.(*ast.GoStmt)
	if !ok {
		return true
	}

	switch fun := goStmt.Call.Fun.(type) {
	case *ast.FuncLit:
		// anonymous function
		pos := p.fset.Position(fun.Pos())
		if len(fun.Body.List) == 0 {
			return true
		}
		body := fun.Body.List[0]
		exprStmt, ok := body.(*ast.ExprStmt)
		if !ok {
			return true
		}
		callStmt, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		selectorExpr, ok := callStmt.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		firstLineFunName := selectorExpr.Sel.Name
		p.logger.Debug("found anonymous goroutine",
			ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
			zap.String("firstLineFunName", firstLineFunName),
		)
		if firstLineFunName != "LogOnPanic" {
			p.logger.Warn("missing LogOnPanic()",
				ZapURI(pos.Filename, pos.Line),
				zap.Int("column", pos.Column),
			)
		}
	case *ast.SelectorExpr:
		// method call
		pos := p.fset.Position(fun.Pos())
		p.logger.Info("found method call as goroutine",
			zap.String("methodName", fun.Sel.Name),
			ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)
		// TODO: Find function definition and check first line
		defFilePath, defLineNumber, err := p.language.Definition(pos.Filename, pos.Line, pos.Column)
		if err != nil {
			p.logger.Error("failed to find function definition", zap.Error(err))
			return false
		}

		p.checkFirstLineInFunctionBody(defFilePath, defLineNumber)
	case *ast.Ident:
		// function call
		pos := p.fset.Position(fun.Pos())
		p.logger.Info("found function call as goroutine",
			zap.String("functionName", fun.Name),
			ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)
		// TODO: Find function definition and check first line
		defFilePath, defLineNumber, err := p.language.Definition(pos.Filename, pos.Line, pos.Column)
		if err != nil {
			p.logger.Error("failed to find function definition", zap.Error(err))
			return false
		}

		p.checkFirstLineInFunctionBody(defFilePath, defLineNumber)
	default:
		p.logger.Error("unexpected goroutine type",
			zap.String("type", fmt.Sprintf("%T", fun)),
		)
		return false
	}

	return true
}

// checkFileForGoroutines scans a Go file for any `go` statements (goroutines)
func (p *Processor) checkFileForGoroutines(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		p.logger.Error("Error opening file", zap.String("file", filePath), zap.Error(err))
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lineNumber int
	// Regex for non-anonymous function/method calls: `go functionName()`
	regex := regexp.MustCompile(`go\s+(\.|\w)+\(\)$`)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text() // Do not trim spaces here

		lineLogger := p.logger.With(
			zap.String("url", fmt.Sprintf("%s:%d", filePath, lineNumber)),
		)

		// Detect anonymous goroutines
		if strings.Contains(line, "go func") {
			lineLogger.Debug("Found anonymous goroutine", zap.String("lineContent", line))
			p.checkFirstLineInFunctionBody(filePath, lineNumber)
			continue
		}

		// Detect non-anonymous goroutines using regex
		if !regex.MatchString(line) {
			continue
		}

		// Find the position of the first occurrence of "()"
		cursorPos := strings.Index(line, "()")
		if cursorPos == -1 {
			lineLogger.Error("failed to find function call")
			continue
		}

		lineLogger.Debug("Found non-anonymous goroutine call",
			zap.Int("cursor", cursorPos),
			zap.String("lineContent", line),
		)

		defFilePath, defLineNumber, err := p.language.Definition(filePath, lineNumber, cursorPos)
		if err != nil {
			lineLogger.Error("failed to find function", zap.Error(err))
			continue
		}

		p.checkFirstLineInFunctionBody(defFilePath, defLineNumber)
	}

	if err := scanner.Err(); err != nil {
		p.logger.Error("failed to read file", zap.Error(err))
	}
}

// checkFirstLineInFunctionBody checks the first line inside a function body for `defer gocommon.utils.LogOnPanic()`
func (p *Processor) checkFirstLineInFunctionBody(filePath string, startLine int) {
	logger := p.logger.With(
		zap.String("url", fmt.Sprintf("%s:%d", filePath, startLine)),
	)

	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Error opening file", zap.Error(err))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentLine int
	for scanner.Scan() {
		currentLine++
		if currentLine <= startLine {
			continue
		}

		line := scanner.Text()
		url := fmt.Sprintf("%s:%d", filePath, startLine)

		if strings.Contains(line, "LogOnPanic()") {
			p.logger.Info("found LogOnPanic()", zap.String("url", url))
		} else {
			p.logger.Warn("missing LogOnPanic()", zap.String("url", url))
		}

		return
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading file", zap.Error(err))
	}
}
