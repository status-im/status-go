package analyzer

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"time"

	"go.uber.org/zap"

	goparser "go/parser"
	gotoken "go/token"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	gopls2 "github.com/status-im/status-go/cmd/lint-panics/gopls"
	"github.com/status-im/status-go/cmd/lint-panics/utils"
)

const Pattern = "LogOnPanic"

type Analyzer struct {
	logger *zap.Logger
	lsp    LSP
	cfg    *Config
}

type LSP interface {
	Definition(context.Context, string, int, int) (string, int, error)
}

func New(ctx context.Context, logger *zap.Logger) (*analysis.Analyzer, error) {
	cfg := Config{}
	flags, err := cfg.ParseFlags()
	if err != nil {
		return nil, err
	}

	logger.Info("creating analyzer", zap.String("root", cfg.RootDir))

	gopls := gopls2.NewGoplsClient(ctx, logger, cfg.RootDir)
	processor := newAnalyzer(logger, gopls, &cfg)

	analyzer := &analysis.Analyzer{
		Name:     "logpanics",
		Doc:      fmt.Sprintf("reports missing defer call to %s", Pattern),
		Run:      processor.Run,
		Flags:    flags,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	return analyzer, nil
}

func newAnalyzer(logger *zap.Logger, lsp LSP, cfg *Config) *Analyzer {
	return &Analyzer{
		logger: logger.Named("processor"),
		lsp:    lsp,
		cfg:    cfg.WithAbsolutePaths(),
	}
}

func (p *Analyzer) Run(pass *analysis.Pass) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	inspected, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	// Check if the node is a GoStmt (which represents a 'go' statement)
	nodeFilter := []ast.Node{
		(*ast.GoStmt)(nil),
	}

	// Traverse the AST to find goroutines
	inspected.Preorder(nodeFilter, func(n ast.Node) {
		p.ProcessNode(ctx, pass, n)
	})

	return nil, nil
}

func (p *Analyzer) ProcessNode(ctx context.Context, pass *analysis.Pass, n ast.Node) {
	goStmt, ok := n.(*ast.GoStmt)
	if !ok {
		panic("unexpected node type")
	}

	switch fun := goStmt.Call.Fun.(type) {
	case *ast.FuncLit: // anonymous function
		pos := pass.Fset.Position(fun.Pos())
		logger := p.logger.With(
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		logger.Debug("found anonymous goroutine")
		if err := p.checkGoroutine(fun.Body); err != nil {
			p.logLinterError(pass, fun.Pos(), fun.Pos(), err)
		}

	case *ast.SelectorExpr: // method call
		pos := pass.Fset.Position(fun.Sel.Pos())
		p.logger.Info("found method call as goroutine",
			zap.String("methodName", fun.Sel.Name),
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		defPos, err := p.checkGoroutineDefinition(ctx, pos, pass)
		if err != nil {
			p.logLinterError(pass, defPos, fun.Sel.Pos(), err)
		}

	case *ast.Ident: // function call
		pos := pass.Fset.Position(fun.Pos())
		p.logger.Info("found function call as goroutine",
			zap.String("functionName", fun.Name),
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		defPos, err := p.checkGoroutineDefinition(ctx, pos, pass)
		if err != nil {
			p.logLinterError(pass, defPos, fun.Pos(), err)
		}

	default:
		p.logger.Error("unexpected goroutine type",
			zap.String("type", fmt.Sprintf("%T", fun)),
		)
	}
}

func (p *Analyzer) parseFile(path string, pass *analysis.Pass) (*ast.File, error) {
	logger := p.logger.With(zap.String("path", path))

	src, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to open file", zap.Error(err))
	}

	file, err := goparser.ParseFile(pass.Fset, path, src, 0)
	if err != nil {
		logger.Error("failed to parse file", zap.Error(err))
		return nil, err
	}

	return file, nil
}

func (p *Analyzer) checkGoroutine(body *ast.BlockStmt) error {
	if body == nil {
		p.logger.Warn("missing function body")
		return nil
	}
	if len(body.List) == 0 {
		// empty goroutine is weird, but it never panics, so not a linter error
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

func (p *Analyzer) getFunctionBody(node ast.Node, lineNumber int, pass *analysis.Pass) (body *ast.BlockStmt, pos gotoken.Pos) {
	ast.Inspect(node, func(n ast.Node) bool {
		// Check if the node is a function declaration
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		if pass.Fset.Position(n.Pos()).Line != lineNumber {
			return true
		}

		body = funcDecl.Body
		pos = n.Pos()
		return false
	})

	return body, pos

}

func (p *Analyzer) checkGoroutineDefinition(ctx context.Context, pos gotoken.Position, pass *analysis.Pass) (gotoken.Pos, error) {
	defFilePath, defLineNumber, err := p.lsp.Definition(ctx, pos.Filename, pos.Line, pos.Column)
	if err != nil {
		p.logger.Error("failed to find function definition", zap.Error(err))
		return 0, err
	}

	file, err := p.parseFile(defFilePath, pass)
	if err != nil {
		p.logger.Error("failed to parse file", zap.Error(err))
		return 0, err
	}

	body, defPosition := p.getFunctionBody(file, defLineNumber, pass)
	return defPosition, p.checkGoroutine(body)
}

func (p *Analyzer) logLinterError(pass *analysis.Pass, errPos gotoken.Pos, callPos gotoken.Pos, err error) {
	errPosition := pass.Fset.Position(errPos)
	callPosition := pass.Fset.Position(callPos)

	if p.skip(errPosition.Filename) || p.skip(callPosition.Filename) {
		return
	}

	message := fmt.Sprintf("missing %s()", Pattern)
	p.logger.Warn(message,
		utils.ZapURI(errPosition.Filename, errPosition.Line),
		zap.String("details", err.Error()))

	if callPos == errPos {
		pass.Reportf(errPos, "missing defer call to %s", Pattern)
	} else {
		pass.Reportf(callPos, "missing defer call to %s", Pattern)
	}
}

func (p *Analyzer) skip(filepath string) bool {
	return p.cfg.SkipDir != "" && strings.HasPrefix(filepath, p.cfg.SkipDir)
}
