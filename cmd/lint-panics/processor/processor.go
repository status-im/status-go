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
	"context"
	"golang.org/x/tools/go/analysis"
)

const Pattern = "LogOnPanic"

type Processor struct {
	logger *zap.Logger
	lsp    LSP
	pass   *analysis.Pass
}

type LSP interface {
	Definition(context.Context, string, int, int) (string, int, error)
}

func NewProcessor(logger *zap.Logger, pass *analysis.Pass, lsp LSP) *Processor {
	return &Processor{
		logger: logger.Named("processor"),
		pass:   pass,
		lsp:    lsp,
	}
}

func (p *Processor) Run(ctx context.Context, path string) error {
	logger := p.logger.With(zap.String("file", path))

	file, err := p.parseFile(path)
	if err != nil {
		logger.Error("failed to parse file", zap.Error(err))
		return err
	}

	// Traverse the AST to find goroutines
	ast.Inspect(file, func(n ast.Node) bool {
		p.ProcessNode(ctx, n)
		return true
	})

	return nil
}

func (p *Processor) ProcessNode(ctx context.Context, n ast.Node) {
	// Check if the node is a GoStmt (which represents a 'go' statement)
	goStmt, ok := n.(*ast.GoStmt)
	if !ok {
		return
	}

	switch fun := goStmt.Call.Fun.(type) {
	case *ast.FuncLit:
		// anonymous function
		pos := p.pass.Fset.Position(fun.Pos())
		logger := p.logger.With(
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		logger.Debug("found anonymous goroutine")
		if err := p.checkGoroutine(fun.Body); err != nil {
			p.logLinterError(fun.Pos(), err)
			p.pass.Reportf(fun.Pos(), "missing %s()", Pattern)
		}

	case *ast.SelectorExpr:
		// method call
		pos := p.pass.Fset.Position(fun.Sel.Pos())
		p.logger.Info("found method call as goroutine",
			zap.String("methodName", fun.Sel.Name),
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		defPos, err := p.checkGoroutineDefinition(ctx, pos)
		if err != nil {
			p.logLinterError(defPos, err)
			p.pass.Reportf(defPos, "missing %s(), goroutine at %s", Pattern, utils.PositionURI(pos))
		}

	case *ast.Ident:
		// function call
		pos := p.pass.Fset.Position(fun.Pos())
		p.logger.Info("found function call as goroutine",
			zap.String("functionName", fun.Name),
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		defPos, err := p.checkGoroutineDefinition(ctx, pos)
		if err != nil {
			p.logLinterError(defPos, err)
			p.pass.Reportf(defPos, "missing %s(), called as goroutine at %s", Pattern, utils.PositionURI(pos))
		}

	default:
		p.logger.Error("unexpected goroutine type",
			zap.String("type", fmt.Sprintf("%T", fun)),
		)
	}

	return
}

func (p *Processor) parseFile(path string) (*ast.File, error) {
	logger := p.logger.With(zap.String("path", path))

	src, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to open file", zap.Error(err))
	}

	file, err := goparser.ParseFile(p.pass.Fset, path, src, 0)
	if err != nil {
		logger.Error("failed to parse file", zap.Error(err))
		return nil, err
	}

	return file, nil
}

func (p *Processor) checkGoroutine(body *ast.BlockStmt) error {
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
	if firstLineFunName != Pattern {
		return errors.Errorf("first statement is not %s", Pattern)
	}

	return nil
}

func (p *Processor) getFunctionBody(node ast.Node, lineNumber int) (body *ast.BlockStmt, pos gotoken.Pos) {
	ast.Inspect(node, func(n ast.Node) bool {
		// Check if the node is a function declaration
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if p.pass.Fset.Position(n.Pos()).Line != lineNumber {
			return true
		}

		body = funcDecl.Body
		pos = n.Pos()
		return false
	})

	return body, pos

}

func (p *Processor) checkGoroutineDefinition(ctx context.Context, pos gotoken.Position) (gotoken.Pos, error) {
	defFilePath, defLineNumber, err := p.lsp.Definition(ctx, pos.Filename, pos.Line, pos.Column)
	if err != nil {
		p.logger.Error("failed to find function definition", zap.Error(err))
		return 0, err
	}

	file, err := p.parseFile(defFilePath)
	if err != nil {
		p.logger.Error("failed to parse file", zap.Error(err))
		return 0, err
	}

	body, defPosition := p.getFunctionBody(file, defLineNumber)
	return defPosition, p.checkGoroutine(body)
}

func (p *Processor) logLinterError(pos gotoken.Pos, err error) {
	position := p.pass.Fset.Position(pos)
	message := fmt.Sprintf("missing %s()", Pattern)
	p.logger.Warn(message,
		utils.ZapURI(position.Filename, position.Line),
		zap.String("details", err.Error()))
}
