package main

import (
	_ "flag"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var Analyzer = &analysis.Analyzer{
	Name: "correcterr",
	Doc:  "Checks that the returned error is the one that was checked",
	Run:  run,
}

func main() {
	// Don't use it: just to not crash on -unsafeptr flag from go vet
	// flag.Bool("unsafeptr", false, "")

	singlechecker.Main(Analyzer)
}

func run(pass *analysis.Pass) (any, error) {
	inspect := func(node ast.Node) bool {
		if node == nil {
			return false
		}

		ifStmt, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}

		binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
		if !ok {
			return true
		}

		if binExpr.Op != token.NEQ {
			return true
		}

		if !ExprIsError(binExpr.X, pass.TypesInfo) {
			return true
		}

		xIdent, ok := binExpr.X.(*ast.Ident)
		if !ok {
			return true
		}

		yIdent, ok := binExpr.Y.(*ast.Ident)
		if !ok {
			return true
		}

		if yIdent.Obj != nil {
			return true
		}

		if yIdent.Name != "nil" {
			return true
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

				errIdent, ok := res.(*ast.Ident)
				if !ok {
					continue
				}

				if errIdent.Name != xIdent.Name {
					pass.Reportf(node.Pos(), "returning not the error that was checked")
				}
			}
		}

		return true

	}

	for _, f := range pass.Files {
		ast.Inspect(f, inspect)
	}

	return nil, nil
}

func ExprIsError(v ast.Expr, info *types.Info) bool {
	if n, ok := info.TypeOf(v).(*types.Named); ok {
		o := n.Obj()
		return o != nil && o.Pkg() == nil && o.Name() == "error"
	}

	return false
}
