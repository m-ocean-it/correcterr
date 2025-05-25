package main

import (
	_ "flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"

	_ "github.com/m-ocean-it/correcterr/pkg/analyzer"
	_ "golang.org/x/tools/go/analysis/singlechecker"
)

// func main() {
// 	// Don't use it: just to not crash on -unsafeptr flag from go vet
// 	// flag.Bool("unsafeptr", false, "")

// 	// singlechecker.Main(analyzer.Analyzer)
// }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No arguments provided. Exiting.")
		os.Exit(1)
	}

	for _, filePath := range os.Args[1:] {
		if filePath == "--" {
			continue
		}

		if !strings.HasSuffix(filePath, ".go") {
			continue
		}

		lintFile(filePath)
	}
}

func lintFile(path string) {
	isError := func(v ast.Expr, info *types.Info) bool {
		if n, ok := info.TypeOf(v).(*types.Named); ok {
			o := n.Obj()
			return o != nil && o.Pkg() == nil && o.Name() == "error"
		}

		return false
	}

	// We parse the AST
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
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

		if !isError(binExpr.X, info) {
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
				if !isError(res, info) {
					continue
				}

				errIdent, ok := res.(*ast.Ident)
				if !ok {
					continue
				}

				if errIdent.Name != xIdent.Name {
					fmt.Printf("%s: returning not the error that was checked\n",
						fset.Position(errIdent.Pos()))
				}
			}
		}

		return true
	})
}
