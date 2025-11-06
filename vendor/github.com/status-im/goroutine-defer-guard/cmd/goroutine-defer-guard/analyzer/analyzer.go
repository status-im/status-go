package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"

	"github.com/status-im/goroutine-defer-guard/cmd/goroutine-defer-guard/utils"
)

const Pattern = "LogOnPanic"

type Analyzer struct {
	logger              *zap.Logger
	cfg                 *Config
	processedGoroutines sync.Map
}

func New(logger *zap.Logger) (*analysis.Analyzer, error) {
	cfg := Config{}
	flags, err := cfg.ParseFlags()
	if err != nil {
		return nil, err
	}

	logger.Info("creating analyzer")

	processor := newAnalyzer(logger, &cfg)

	analyzer := &analysis.Analyzer{
		Name:     "goroutinedeferguard",
		Doc:      fmt.Sprintf("reports missing defer call to %s", Pattern),
		Flags:    flags,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return processor.Run(pass)
		},
	}

	return analyzer, nil
}

func newAnalyzer(logger *zap.Logger, cfg *Config) *Analyzer {
	return &Analyzer{
		logger:              logger.Named("processor"),
		cfg:                 cfg.WithAbsolutePaths(),
		processedGoroutines: sync.Map{},
	}
}

func (p *Analyzer) Run(pass *analysis.Pass) (interface{}, error) {
	inspected, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("analyzer is not type *inspector.Inspector")
	}

	// Create a nodes filter for goroutines (GoStmt represents a 'go' statement)
	nodeFilter := []ast.Node{
		(*ast.GoStmt)(nil),
	}

	// Inspect go statements
	inspected.Preorder(nodeFilter, func(n ast.Node) {
		p.ProcessNode(pass, n)
	})

	return nil, nil
}

func (p *Analyzer) ProcessNode(pass *analysis.Pass, n ast.Node) {
	goStmt, ok := n.(*ast.GoStmt)
	if !ok {
		panic("unexpected node type")
	}

	// Skip if we've already processed this goroutine statement
	// This can happen when the same package is analyzed multiple times
	pos := goStmt.Pos()
	if _, loaded := p.processedGoroutines.LoadOrStore(pos, struct{}{}); loaded {
		return
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

		if err := p.checkGoroutineDefinition(pass, fun, goStmt.Pos()); err != nil {
			p.logLinterError(pass, goStmt.Pos(), goStmt.Pos(), err)
		}

	case *ast.Ident: // function call
		pos := pass.Fset.Position(fun.Pos())
		p.logger.Info("found function call as goroutine",
			zap.String("functionName", fun.Name),
			utils.ZapURI(pos.Filename, pos.Line),
			zap.Int("column", pos.Column),
		)

		if err := p.checkGoroutineDefinition(pass, fun, goStmt.Pos()); err != nil {
			p.logLinterError(pass, goStmt.Pos(), goStmt.Pos(), err)
		}

	default:
		p.logger.Error("unexpected goroutine type",
			zap.String("type", fmt.Sprintf("%T", fun)),
		)
	}
}

func (p *Analyzer) checkGoroutine(body *ast.BlockStmt) error {
	if body == nil {
		p.logger.Warn("missing function body")
		return nil
	}

	if len(body.List) == 0 {
		// empty goroutine is weird, but it is not a linter error
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

	if selectorExpr.Sel.Name != Pattern {
		return errors.Errorf("first statement is not %s", Pattern)
	}

	return nil
}

func (p *Analyzer) checkGoroutineDefinition(pass *analysis.Pass, fun ast.Expr, callPos token.Pos) error {
	var funcName string
	var receiverType types.Type

	// Extract function name and receiver type if it's a method
	switch e := fun.(type) {
	case *ast.Ident:
		funcName = e.Name
		// Check if this identifier refers to a variable holding a function literal
		obj := pass.TypesInfo.ObjectOf(e)
		if obj == nil {
			break
		}

		varObj, ok := obj.(*types.Var)
		if !ok {
			break
		}

		// This is a variable, try to find all its function literal assignments
		funcLits := p.findAllFunctionLiteralAssignments(pass, varObj)
		if len(funcLits) == 0 {
			break
		}

		// Check all assignments - if any don't have the defer, report error
		for _, funcLit := range funcLits {
			if err := p.checkGoroutine(funcLit.Body); err != nil {
				return err
			}
		}
		// All assignments are valid
		return nil
	case *ast.SelectorExpr:
		funcName = e.Sel.Name
		// Determine the static type of the receiver expression directly
		// This works for identifiers, field selectors, and parenthesized forms.
		if t := pass.TypesInfo.TypeOf(e.X); t != nil {
			receiverType = t
		} else if ident, ok := e.X.(*ast.Ident); ok {
			// Fallback to object-based lookup for identifiers
			if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
				if varObj, ok := obj.(*types.Var); ok {
					receiverType = varObj.Type()
				}
			}
		}

		// If the receiver is an interface, verify all concrete implementations
		if receiverType != nil {
			if _, isInterface := receiverType.Underlying().(*types.Interface); isInterface {
				if err := p.checkInterfaceMethodCall(pass, funcName, receiverType, callPos); err != nil {
					p.logger.Warn("cannot verify interface method call",
						zap.String("method", funcName),
						zap.String("interface", receiverType.String()),
						zap.String("reason", err.Error()),
					)
					// Don't report an error for interface calls we can't verify
					return nil
				}
				return nil
			}
		}

		// Try to resolve external method or package-qualified function via type info
		if sel := pass.TypesInfo.Selections[e]; sel != nil {
			if fn, ok := sel.Obj().(*types.Func); ok {
				if fn.Pkg() != nil && pass.Pkg != nil && fn.Pkg().Path() != pass.Pkg.Path() {
					if err := p.checkExternalFunc(pass, fn); err != nil {
						return err
					}
					return nil
				}
			}
		} else {
			// Not a method selection; this is likely a package-qualified function call: pkg.Func()
			if obj, ok := pass.TypesInfo.Uses[e.Sel].(*types.Func); ok {
				if obj.Pkg() != nil && pass.Pkg != nil && obj.Pkg().Path() != pass.Pkg.Path() {
					if err := p.checkExternalFunc(pass, obj); err != nil {
						return err
					}
					return nil
				}
			}
		}
	default:
		return errors.New("unsupported function expression type")
	}

	// Find the function declaration using AST traversal
	for _, file := range pass.Files {
		var body *ast.BlockStmt

		if funcName == "Listen" {
			p.logger.Debug("looking for Listen", zap.String("filePath", file.Name.Name))
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				// Check if this is the function we're looking for
				if node.Name.Name != funcName {
					break
				}

				// If we're looking for a method, check the receiver
				if receiverType != nil && node.Recv != nil && len(node.Recv.List) > 0 {
					// For methods, we need to match the receiver type
					for _, field := range node.Recv.List {
						// Get the type from the declaration using TypesInfo
						declType := pass.TypesInfo.TypeOf(field.Type)
						if declType != nil && types.Identical(receiverType, declType) {
							body = node.Body
							return false
						}
					}
				} else if receiverType == nil && node.Recv == nil {
					// For regular functions, no receiver should be present
					body = node.Body
					return false
				}
			}
			return true
		})

		if body != nil {
			return p.checkGoroutine(body)
		}
	}

	return errors.New("could not find function body")
}

func (p *Analyzer) logLinterError(pass *analysis.Pass, errPos token.Pos, callPos token.Pos, err error) {
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
		pass.Reportf(errPos, "missing defer call to %s: %s", Pattern, err.Error())
	} else {
		pass.Reportf(callPos, "missing defer call to %s: %s", Pattern, err.Error())
	}
}

func (p *Analyzer) skip(filepath string) bool {
	return p.cfg.SkipDir != "" && strings.HasPrefix(filepath, p.cfg.SkipDir)
}

// checkInterfaceMethodCall attempts to find and verify all concrete implementations
// of an interface method that could be called at runtime
func (p *Analyzer) checkInterfaceMethodCall(pass *analysis.Pass, methodName string, interfaceType types.Type, callPos token.Pos) error {
	iface, ok := interfaceType.Underlying().(*types.Interface)
	if !ok {
		return errors.New("not an interface type")
	}

	// Find all types in the current package that implement this interface
	var implementations []*ast.FuncDecl
	var implementationTypes []types.Type

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				return true
			}

			// Check if this is the method we're looking for
			if funcDecl.Name.Name != methodName {
				return true
			}

			// Get the receiver type
			recvType := pass.TypesInfo.TypeOf(funcDecl.Recv.List[0].Type)
			if recvType == nil {
				return true
			}

			// Check if this receiver type implements the interface
			if types.Implements(recvType, iface) || types.Implements(types.NewPointer(recvType), iface) {
				implementations = append(implementations, funcDecl)
				implementationTypes = append(implementationTypes, recvType)
			}

			return true
		})
	}

	if len(implementations) == 0 {
		return errors.New("no implementations found in current package")
	}

	// Check all implementations - directly report missing defer at the call site
	for range implementationTypes {
		// no-op: keep parallel arrays consistent if needed later
	}
	for i, impl := range implementations {
		if err := p.checkGoroutine(impl.Body); err != nil {
			// Report: error position is implementation method, call position is the goroutine call site
			_ = implementationTypes[i] // reserved for future message enrichment
			p.logLinterError(pass, impl.Pos(), callPos, err)
		}
	}

	p.logger.Debug("all interface implementations verified",
		zap.String("interface", interfaceType.String()),
		zap.String("method", methodName),
		zap.Int("implementations", len(implementations)),
	)

	return nil
}

// findAllFunctionLiteralAssignments finds ALL function literal assignments to a variable.
// It uses TypesInfo to ensure we match the exact variable object, not just any variable with the same name.
// This is important because a variable can be reassigned, and we need to check all possible values.
func (p *Analyzer) findAllFunctionLiteralAssignments(pass *analysis.Pass, varObj *types.Var) []*ast.FuncLit {
	var funcLits []*ast.FuncLit

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Look for assignment statements or variable declarations
			switch node := n.(type) {
			case *ast.AssignStmt:
				// Check if any LHS is our exact variable (using object identity)
				for i, lhs := range node.Lhs {
					ident, ok := lhs.(*ast.Ident)
					if !ok {
						continue
					}

					// For short declarations (:=), the LHS is a definition (Defs)
					// For reassignments (=), the LHS is a use (Uses)
					// We need to check both
					def := pass.TypesInfo.Defs[ident]
					use := pass.TypesInfo.Uses[ident]

					if def != varObj && use != varObj {
						continue
					}

					if i >= len(node.Rhs) {
						continue
					}

					lit, ok := node.Rhs[i].(*ast.FuncLit)
					if !ok {
						continue
					}

					funcLits = append(funcLits, lit)
				}
			case *ast.ValueSpec:
				// Check variable declarations like: var x = func() {}
				for i, name := range node.Names {
					// Use Defs to get the object being defined here
					if pass.TypesInfo.Defs[name] != varObj {
						continue
					}

					if i >= len(node.Values) {
						continue
					}

					lit, ok := node.Values[i].(*ast.FuncLit)
					if !ok {
						continue
					}

					funcLits = append(funcLits, lit)
				}
			}
			return true
		})
	}

	return funcLits
}

// checkExternalFunc attempts to load the defining package for the given function
// and check its body for the required defer statement. If the body cannot be
// found or the package cannot be loaded, it returns nil to avoid false positives.
func (p *Analyzer) checkExternalFunc(pass *analysis.Pass, fn *types.Func) error {
	body, err := p.findFuncBodyInObjectPackage(fn)
	if err != nil {
		p.logger.Debug("cannot load external function body",
			zap.String("function", fn.FullName()),
			zap.String("pkg", fn.Pkg().Path()),
			zap.String("reason", err.Error()),
		)
		// Avoid false positive when we cannot resolve external bodies
		return nil
	}
	if body == nil {
		return nil
	}
	return p.checkGoroutine(body)
}

// findFuncBodyInObjectPackage loads the package where the function is defined and
// returns the corresponding *ast.BlockStmt body if found.
func (p *Analyzer) findFuncBodyInObjectPackage(fn *types.Func) (*ast.BlockStmt, error) {
	if fn == nil || fn.Pkg() == nil {
		return nil, errors.New("function has no package")
	}

	pkgPath := fn.Pkg().Path()
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, errors.Wrap(err, "packages.Load failed")
	}
	if packages.PrintErrors(pkgs) > 0 || len(pkgs) == 0 {
		return nil, errors.Errorf("failed to load package %s", pkgPath)
	}

	targetName := fn.Name()
	var targetRecv string
	if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
		targetRecv = types.TypeString(sig.Recv().Type(), func(p *types.Package) string { return p.Path() })
	}

	for _, pkg := range pkgs {
		ti := pkg.TypesInfo
		for _, f := range pkg.Syntax {
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Name == nil || fd.Name.Name != targetName {
					continue
				}
				if targetRecv == "" {
					if fd.Recv == nil {
						return fd.Body, nil
					}
					continue
				}
				if fd.Recv == nil || len(fd.Recv.List) == 0 {
					continue
				}
				recvType := ti.TypeOf(fd.Recv.List[0].Type)
				if recvType == nil {
					continue
				}
				recvStr := types.TypeString(recvType, func(p *types.Package) string { return p.Path() })
				if recvStr == targetRecv {
					return fd.Body, nil
				}
			}
		}
	}

	return nil, errors.New("function body not found in loaded package")
}
