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

		var checkedErrVars []*ast.Ident

		switch ifExpr := ifStmt.Cond.(type) {
		case *ast.BinaryExpr:
			binExpr := ifExpr

			if binExpr.Op != token.NEQ {
				return
			}

			if !exprIsError(binExpr.X, pass.TypesInfo) {
				return
			}

			id, ok := binExpr.X.(*ast.Ident)
			if !ok {
				return
			}

			rightVar, ok := binExpr.Y.(*ast.Ident)
			if !ok {
				return
			}

			if rightVar.Obj != nil {
				return
			}

			if rightVar.Name != "nil" {
				return
			}

			checkedErrVars = append(checkedErrVars, id)
		case *ast.CallExpr:
			checkedErrVars = append(checkedErrVars, scanCallForErrs(ifExpr, pass)...)
		default:
			return
		}

		if len(checkedErrVars) == 0 {
			return
		}

		for _, bodyStmt := range ifStmt.Body.List {
			retStmt, ok := bodyStmt.(*ast.ReturnStmt)
			if !ok {
				continue
			}

			var (
				returns        bool
				callExpectsErr bool
			)

		RETURN_RESULTS:
			for _, res := range retStmt.Results {
				if exprIsError(res, pass.TypesInfo) {
					callExpectsErr = true
				} else if !exprIsString(res, pass.TypesInfo) {
					continue
				}

				switch returnVal := res.(type) {
				case *ast.Ident:
					if errIdentIsInList(returnVal, checkedErrVars) {
						returns = true
						break RETURN_RESULTS
					}
				case *ast.CallExpr:
					rets, expects := inspectErrCall(checkedErrVars, returnVal, pass)
					if rets {
						returns = true
						break RETURN_RESULTS
					}
					if expects {
						callExpectsErr = true
					}
				}
			}

			if callExpectsErr && !returns {
				pass.Reportf(retStmt.Pos(), "returning not the error that was checked")
			}
		}
	})

	return nil, nil
}

func inspectErrCall(checkedErrs []*ast.Ident, call *ast.CallExpr, pass *analysis.Pass) (bool, bool) {
	for _, checkedErr := range checkedErrs {
		if callIsErrDotErrorOnTarget(call, checkedErr) {
			return true, false
		}
	}

	var returns bool
	var callExpectsErr bool

LOOP:
	for _, arg := range call.Args {
		if exprIsError(arg, pass.TypesInfo) {
			callExpectsErr = true
		} else if !exprIsString(arg, pass.TypesInfo) {
			continue
		}

		switch typedArg := arg.(type) {
		case *ast.Ident:
			if errIdentIsInList(typedArg, checkedErrs) {
				returns = true
				break LOOP
			}
		case *ast.CallExpr:
			rets, _ := inspectErrCall(checkedErrs, typedArg, pass)
			if rets {
				returns = true
				break LOOP
			}
		}
	}

	return returns, callExpectsErr
}

func exprIsError(v ast.Expr, info *types.Info) bool {
	if n, ok := info.TypeOf(v).(*types.Named); ok {
		o := n.Obj()
		return o != nil && o.Pkg() == nil && o.Name() == "error"
	}

	return false
}

func exprIsString(v ast.Expr, info *types.Info) bool {
	if basicType, ok := info.TypeOf(v).(*types.Basic); ok {
		return basicType.Name() == "string"
	}

	return false
}

func callIsErrDotErrorOnTarget(call *ast.CallExpr, target *ast.Ident) bool {
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selExpr == nil || selExpr.Sel == nil {
		return false
	}

	xIdent, ok := selExpr.X.(*ast.Ident)
	if !ok || xIdent == nil {
		return false
	}

	if xIdent.Name != target.Name {
		return false
	}

	return selExpr.Sel.Name == "Error" && selExpr.Sel.Obj == nil
}

func scanCallForErrs(call *ast.CallExpr, pass *analysis.Pass) []*ast.Ident {
	var errIdents []*ast.Ident

	for _, arg := range call.Args {
		if !exprIsError(arg, pass.TypesInfo) {
			continue
		}

		switch typedArg := arg.(type) {
		case *ast.Ident:
			errIdents = append(errIdents, typedArg)
		case *ast.CallExpr:
			errIdents = append(errIdents, scanCallForErrs(typedArg, pass)...)
		}
	}

	return errIdents
}

func errIdentIsInList(target *ast.Ident, errIdents []*ast.Ident) bool {
	for _, eID := range errIdents {
		if eID.Name == target.Name {
			return true
		}
	}

	return false
}
