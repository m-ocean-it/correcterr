package main

import (
	_ "flag"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/singlechecker"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "correcterr",
	Doc:      "Checks that the returned error is the one that was checked",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func main() {
	// Don't use it: just to not crash on -unsafeptr flag from go vet
	// flag.Bool("unsafeptr", false, "")

	singlechecker.Main(Analyzer)
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

				errIdent, ok := res.(*ast.Ident)
				if !ok {
					continue
				}

				if errIdent.Name != xIdent.Name {
					pass.Reportf(node.Pos(), "returning not the error that was checked")
				}
			}
		}
	})

	return nil, nil
}

func ExprIsError(v ast.Expr, info *types.Info) bool {
	if n, ok := info.TypeOf(v).(*types.Named); ok {
		o := n.Obj()
		return o != nil && o.Pkg() == nil && o.Name() == "error"
	}

	return false
}
