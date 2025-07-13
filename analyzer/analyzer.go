package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"maps"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	nolintDirective = "nolint"
	nolintName      = "correcterr"
	nolintAll       = "all"
)

var Analyzer = &analysis.Analyzer{
	Name:     "correcterr",
	Doc:      "Checks that the returned error is the one that was checked",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

type anySet = map[any]struct{}

type state struct {
	pass              *analysis.Pass
	checkedErrorDecls anySet
	wraps             map[*ast.Ident]*ast.Ident
	commentMap        ast.CommentMap
}

func run(pass *analysis.Pass) (any, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	commentMap := make(ast.CommentMap)
	for _, f := range pass.Files {
		cmap := ast.NewCommentMap(pass.Fset, f, f.Comments)
		maps.Copy(commentMap, cmap)
	}

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		funcNode, _ := node.(*ast.FuncDecl)
		if funcNode == nil {
			return
		}

		st := state{
			pass:              pass,
			checkedErrorDecls: make(anySet),
			wraps:             make(map[*ast.Ident]*ast.Ident),
			commentMap:        commentMap,
		}

		processFuncNode(st, funcNode)
	})

	return nil, nil
}

func processFuncNode(st state, funcNode *ast.FuncDecl) {
	for _, bodyElem := range funcNode.Body.List {
		ifStmt, ok := bodyElem.(*ast.IfStmt)
		if !ok {
			continue
		}

		processIfStmt(st, ifStmt)
	}
}

func processIfStmt(st state, ifStmt *ast.IfStmt) {
	maybeCheckedError := getMaybeCheckedErrorDeclFromIfCondition(st.pass, ifStmt)
	if maybeCheckedError != nil {
		st.checkedErrorDecls = maps.Clone(st.checkedErrorDecls)
		st.checkedErrorDecls[maybeCheckedError] = struct{}{}
	}

	for _, ifBodyElem := range ifStmt.Body.List {
		switch e := ifBodyElem.(type) {
		case *ast.IfStmt:
			processIfStmt(st, e)
		case *ast.ReturnStmt:
			processReturnStmt(st, e)
		}
	}
}

func getMaybeCheckedErrorDeclFromIfCondition(pass *analysis.Pass, ifStmt *ast.IfStmt) any {
	binaryCondition, _ := ifStmt.Cond.(*ast.BinaryExpr)
	if binaryCondition == nil {
		return nil
	}

	if binaryCondition.Op != token.NEQ {
		return nil
	}

	if !exprIsError(binaryCondition.X, pass.TypesInfo) {
		return nil
	}

	checkedError, ok := binaryCondition.X.(*ast.Ident)
	if !ok {
		return nil
	}

	rightVar, ok := binaryCondition.Y.(*ast.Ident)
	if !ok {
		return nil
	}

	if rightVar.Obj != nil {
		return nil
	}

	if rightVar.Name != "nil" {
		return nil
	}

	return checkedError.Obj.Decl
}

func processReturnStmt(st state, returnStmt *ast.ReturnStmt) {
	for _, expr := range returnStmt.Results {
		switch e := expr.(type) {
		case *ast.Ident:
			checkReturnIdent(st, e)
		}
	}
}

func checkReturnIdent(st state, returnIdent *ast.Ident) {
	returnIdentDecl := returnIdent.Obj.Decl

	_, wasChecked := st.checkedErrorDecls[returnIdentDecl]

	if !wasChecked {
		st.pass.Reportf(returnIdent.Pos(), "returning not the error that was checked")
	}
}

func exprIsError(v ast.Expr, info *types.Info) bool {
	if n, ok := info.TypeOf(v).(*types.Named); ok {
		o := n.Obj()
		return o != nil && o.Pkg() == nil && o.Name() == "error"
	}

	return false
}
