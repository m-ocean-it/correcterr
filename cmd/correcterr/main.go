package main

import (
	_ "flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	_ "github.com/m-ocean-it/correcterr/pkg/analyzer"
	_ "golang.org/x/tools/go/analysis/singlechecker"
)

// func main() {
// 	// Don't use it: just to not crash on -unsafeptr flag from go vet
// 	// flag.Bool("unsafeptr", false, "")

// 	// singlechecker.Main(analyzer.Analyzer)
// }

func main() {
	isError := func(v ast.Expr, info *types.Info) bool {
		if n, ok := info.TypeOf(v).(*types.Named); ok {
			o := n.Obj()
			return o != nil && o.Pkg() == nil && o.Name() == "error"
		}

		return false
	}

	// We parse the AST
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "testdata/src/p/err_mistakes.go", nil, 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	// We extract type info
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	conf := types.Config{Importer: importer.Default()}
	if _, err = conf.Check("p", fset, []*ast.File{f}, info); err != nil {
		fmt.Println(err)
		return
	}

	ast.Inspect(f, func(node ast.Node) bool {
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

		xIsErr := isError(binExpr.X, info)
		yIsErr := isError(binExpr.Y, info)

		fmt.Println("x is err:", xIsErr)
		fmt.Println("y is err:", yIsErr)

		return true
	})
}
