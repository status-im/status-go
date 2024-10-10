package processor

import (
	"os"
	"go.uber.org/zap"
	"fmt"
	"go/ast"
	gotoken "go/token"
	goparser "go/parser"
	"github.com/pkg/errors"
	"github.com/status-im/status-go/cmd/lint-panics/utils"
)

const LogOnPanic = "LogOnPanic"

type Processor struct {
	logger *zap.Logger
	fset   *gotoken.FileSet
	lsp    LSP
}

type LSP interface {
	Definition(string, int, int) (string, int, error)
}

func NewProcessor(logger *zap.Logger, lsp LSP) *Processor {
	return &Processor{
		logger: logger.Named("parser"),
		fset:   gotoken.NewFileSet(),
		lsp:    lsp,
	}
}

func (p *Processor) Run(path string) ([]string, error) {
	logger := p.logger.With(zap.String("file", path))
	//logger.Debug("scanning file")

	file, err := p.parseFile(path)
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
		logger := p.logger.With(
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		logger.Debug("found anonymous goroutine")
		if err := p.сheckGoroutine(fun.Body); err != nil {
			logger.Warn("missing LogOnPanic()", zap.Error(err))
		}

	case *ast.SelectorExpr:
		// method call
		pos := p.fset.Position(fun.Sel.Pos())
		p.logger.Info("found method call as goroutine",
			zap.String("methodName", fun.Sel.Name),
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		defFilePath, defLineNumber, err := p.checkGoroutineDefinition(pos)
		if err != nil {
			logger := p.logger.With(
				utils.ZapURI(defFilePath, defLineNumber),
				zap.Int("column", pos.Column),
			)
			logger.Warn("missing LogOnPanic()", zap.Error(err))
		}

	case *ast.Ident:
		// function call
		pos := p.fset.Position(fun.Pos())
		p.logger.Info("found function call as goroutine",
			zap.String("functionName", fun.Name),
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		defFilePath, defLineNumber, err := p.checkGoroutineDefinition(pos)
		if err != nil {
			logger := p.logger.With(
				utils.ZapURI(defFilePath, defLineNumber),
				zap.Int("column", pos.Column),
			)
			logger.Warn("missing LogOnPanic()", zap.Error(err))
		}

	default:
		p.logger.Error("unexpected goroutine type",
			zap.String("type", fmt.Sprintf("%T", fun)),
		)
		return true
	}

	return true
}

func (p *Processor) parseFile(path string) (*ast.File, error) {
	logger := p.logger.With(zap.String("path", path))

	src, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to open file", zap.Error(err))
	}

	file, err := goparser.ParseFile(p.fset, path, src, 0)
	if err != nil {
		logger.Error("failed to parse file", zap.Error(err))
		return nil, err
	}

	return file, nil
}

func (p *Processor) сheckGoroutine(body *ast.BlockStmt) error {
	if body == nil {
		p.logger.Warn("missing function body")
		return nil
	}
	if len(body.List) == 0 {
		return nil
	}

	deferStatement, ok := body.List[0].(*ast.DeferStmt)
	if !ok {
		return errors.New("first statement is not defer")
	}

	selectorExpr, ok := deferStatement.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		return errors.New("first statement call is not a selector")
	}

	firstLineFunName := selectorExpr.Sel.Name
	if firstLineFunName != LogOnPanic {
		return errors.New("first statement is not LogOnPanic")
	}

	return nil
}

func (p *Processor) GetFunctionBody(node ast.Node, lineNumber int) (body *ast.BlockStmt) {
	// Traverse the AST to find the function declaration at the specified position
	ast.Inspect(node, func(n ast.Node) bool {
		// Get the start position of the function

		// Check if the node is a function declaration
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		startPos := p.fset.Position(n.Pos())

		if startPos.Line != lineNumber {
			return true
		}

		//// Check if the function matches the given line and column
		//if startPos.Line != line && startPos.Column <= column {
		//	// Get the function body as a string
		//	fmt.Printf("Found function %s at line %d, column %d\n", funcDecl.Name.Name, startPos.Line, startPos.Column)
		//
		//	// Get the body of the function
		//	bodyPos := p.fset.Position(.Pos())
		//
		//}

		body = funcDecl.Body

		return false
	})

	return body
}

func (p *Processor) checkGoroutineDefinition(pos gotoken.Position) (string, int, error) {
	defFilePath, defLineNumber, err := p.lsp.Definition(pos.Filename, pos.Line, pos.Column)
	if err != nil {
		p.logger.Error("failed to find function definition", zap.Error(err))
		return "", 0, err
	}

	file, err := p.parseFile(defFilePath)
	if err != nil {
		p.logger.Error("failed to parse file", zap.Error(err))
		return "", 0, err
	}

	body := p.GetFunctionBody(file, defLineNumber)
	err = p.сheckGoroutine(body)

	return defFilePath, defLineNumber, err
}
