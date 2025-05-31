package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "correcterr",
	Doc:      "Checks that the returned error is the one that was checked",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (any, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{(*ast.IfStmt)(nil)}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		if node == nil {
			return
		}

		ifStmt := node.(*ast.IfStmt)

		binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
		if !ok {
			return
		}

		if binExpr.Op != token.NEQ {
			return
		}

		if !ExprIsError(binExpr.X, pass.TypesInfo) {
			return
		}

		xIdent, ok := binExpr.X.(*ast.Ident)
		if !ok {
			return
		}

		yIdent, ok := binExpr.Y.(*ast.Ident)
		if !ok {
			return
		}

		if yIdent.Obj != nil {
			return
		}

		if yIdent.Name != "nil" {
			return
		}

		for _, bodyStmt := range ifStmt.Body.List {
			retStmt, ok := bodyStmt.(*ast.ReturnStmt)
			if !ok {
				continue
			}

			for _, res := range retStmt.Results {
				if !ExprIsError(res, pass.TypesInfo) {
					continue
				}

				switch t := res.(type) {
				case *ast.Ident:
					if t.Name != xIdent.Name {
						pass.Reportf(retStmt.Pos(), "returning not the error that was checked")
					}
				case *ast.CallExpr:
					returns, callExpectsErr := inspectErrCall(xIdent, t, pass)
					if callExpectsErr && !returns {
						pass.Reportf(retStmt.Pos(), "returning not the error that was checked")
					}
				}
			}
		}
	})

	return nil, nil
}

func inspectErrCall(leftErrVar *ast.Ident, call *ast.CallExpr, pass *analysis.Pass) (bool, bool) {
	var returns bool
	var callExpectsErr bool

LOOP:
	for _, arg := range call.Args {
		if !ExprIsError(arg, pass.TypesInfo) {
			continue
		}

		callExpectsErr = true

		switch typedArg := arg.(type) {
		case *ast.Ident:
			if typedArg.Name == leftErrVar.Name {
				returns = true
				break LOOP
			}
		case *ast.CallExpr:
			rets, _ := inspectErrCall(leftErrVar, typedArg, pass)
			if rets {
				returns = true
				break LOOP
			}
		}
	}

	return returns, callExpectsErr
}

func ExprIsError(v ast.Expr, info *types.Info) bool {
	if n, ok := info.TypeOf(v).(*types.Named); ok {
		o := n.Obj()
		return o != nil && o.Pkg() == nil && o.Name() == "error"
	}

	return false
}
