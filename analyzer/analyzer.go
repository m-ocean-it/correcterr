package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"slices"
	"strings"

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

		for _, funcBodyElement := range funcNode.Body.List {
			inspectStatement(pass, funcNode, commentMap, funcBodyElement)
		}
	})

	return nil, nil
}

func inspectStatement(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		inspectIfStmt(pass, funcNode, commentMap, s)
	case *ast.SwitchStmt:
		inspectSwitchStmt(pass, funcNode, commentMap, s)
	case *ast.ForStmt:
		inspectForStmt(pass, funcNode, commentMap, s)
	case *ast.RangeStmt:
		inspectRangeStmt(pass, funcNode, commentMap, s)
	case *ast.ExprStmt:
		inspectExprStmt(pass, funcNode, commentMap, s)
	case *ast.AssignStmt:
		inspectAssignStmt(pass, funcNode, commentMap, s)
	}
}

func inspectIfStmt(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, ifStmt *ast.IfStmt) {
	maybeCheckedErr := tryGetCheckedErrFromIfStmt(pass, ifStmt)

	if maybeCheckedErr != nil {
		checkedError := maybeCheckedErr

		for _, bodyStmt := range ifStmt.Body.List {
			retStmt, ok := bodyStmt.(*ast.ReturnStmt)
			if !ok {
				continue
			}

			if retStmtCommentGroup, ok := commentMap[retStmt]; ok {
				if checkCommentGroupsForNoLint(retStmtCommentGroup) {
					continue
				}
			}

		CHECK_RETURN_RESULTS:
			for _, res := range retStmt.Results {
				if !exprIsError(res, pass.TypesInfo) {
					continue
				}

				var toReport bool

				switch returnVal := res.(type) {

				case *ast.Ident:
					if returnVal.Name != checkedError.Name && errIdentIsDeclaredInFunc(funcNode, returnVal) {
						toReport = true
					}

				case *ast.CallExpr:
					if inspectCall(pass, funcNode, checkedError, returnVal) {
						toReport = true
					}
				}

				if toReport {
					pass.Reportf(retStmt.Pos(), "returning not the error that was checked")

					break CHECK_RETURN_RESULTS
				}
			}
		}
	}

	for _, bodyStmt := range ifStmt.Body.List {
		inspectStatement(pass, funcNode, commentMap, bodyStmt)
	}
}

func inspectSwitchStmt(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, switchStmt *ast.SwitchStmt) {
	for _, stmt := range switchStmt.Body.List {
		caseClause, _ := stmt.(*ast.CaseClause)
		if caseClause == nil {
			continue
		}

		for _, caseClauseStmt := range caseClause.Body {
			inspectStatement(pass, funcNode, commentMap, caseClauseStmt)
		}
	}
}

func inspectForStmt(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, forStmt *ast.ForStmt) {
	for _, stmt := range forStmt.Body.List {
		inspectStatement(pass, funcNode, commentMap, stmt)
	}
}

func inspectRangeStmt(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, rangeStmt *ast.RangeStmt) {
	for _, stmt := range rangeStmt.Body.List {
		inspectStatement(pass, funcNode, commentMap, stmt)
	}
}

func inspectExprStmt(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, exprStmt *ast.ExprStmt) {
	inspectExpr(pass, funcNode, commentMap, exprStmt.X)
}

func inspectExpr(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, expr ast.Expr) {
	switch x := expr.(type) {
	case *ast.CallExpr:
		inspectCallExpr(pass, funcNode, commentMap, x)
	}
}

func inspectCallExpr(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, callExpr *ast.CallExpr) {
	funcLit, _ := callExpr.Fun.(*ast.FuncLit)
	if funcLit == nil {
		return
	}

	for _, stmt := range funcLit.Body.List {
		inspectStatement(pass, funcNode, commentMap, stmt)
	}
}

func inspectAssignStmt(pass *analysis.Pass, funcNode *ast.FuncDecl, commentMap ast.CommentMap, assignStmt *ast.AssignStmt) {
	for _, rightExpr := range assignStmt.Rhs {
		inspectExpr(pass, funcNode, commentMap, rightExpr)
	}
}

func tryGetCheckedErrFromIfStmt(pass *analysis.Pass, ifStmt *ast.IfStmt) *ast.Ident {
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

	return checkedError
}

func inspectCall(pass *analysis.Pass, funcNode *ast.FuncDecl, checkedErr *ast.Ident, call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if !exprIsError(arg, pass.TypesInfo) {
			continue
		}

		switch errArg := arg.(type) {
		case *ast.Ident:
			if errArg.Name != checkedErr.Name && errIdentIsDeclaredInFunc(funcNode, errArg) {
				return true
			}
		case *ast.CallExpr:
			if inspectCall(pass, funcNode, checkedErr, errArg) {
				return true
			}
		}
	}

	return false
}

func errIdentIsDeclaredInFunc(funcNode *ast.FuncDecl, errIdent *ast.Ident) bool {
	for _, stmt := range funcNode.Body.List {
		switch s := stmt.(type) {

		case *ast.DeclStmt:
			if declIsErrIdent(s, errIdent) {
				return true
			}

		case *ast.AssignStmt:
			if assignIsErrIdent(s, errIdent) {
				return true
			}
		}
	}

	return false
}

func declIsErrIdent(decl *ast.DeclStmt, errIdent *ast.Ident) bool {
	genDecl, _ := decl.Decl.(*ast.GenDecl)
	if genDecl == nil {
		return false
	}

	if genDecl.Tok != token.VAR {
		return false
	}

	for _, spec := range genDecl.Specs {
		valSpec, _ := spec.(*ast.ValueSpec)
		if valSpec == nil {
			continue
		}

		for _, name := range valSpec.Names {
			if name.Name == errIdent.Name {
				return true
			}
		}
	}

	return false
}

func assignIsErrIdent(assign *ast.AssignStmt, errIdent *ast.Ident) bool {
	for _, leftExpr := range assign.Lhs {
		leftIdent, _ := leftExpr.(*ast.Ident)
		if leftIdent == nil {
			continue
		}

		if leftIdent.Name == errIdent.Name {
			return true
		}
	}

	return false
}

func exprIsError(v ast.Expr, info *types.Info) bool {
	if n, ok := info.TypeOf(v).(*types.Named); ok {
		o := n.Obj()
		return o != nil && o.Pkg() == nil && o.Name() == "error"
	}

	return false
}

// TODO: use as an iterator
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
		case *ast.SelectorExpr:
			errIdents = append(errIdents, typedArg.Sel)
		}
	}

	return errIdents
}

func checkCommentGroupsForNoLint(commGroups []*ast.CommentGroup) bool {
	for _, cgroup := range commGroups {
		for _, comment := range cgroup.List {
			nolintTrimmed := strings.TrimPrefix(comment.Text, "//"+nolintDirective)
			if len(nolintTrimmed) == len(comment.Text) {
				continue
			}

			if nolintTrimmed == "" {
				return true
			}

			colonTrimmed := strings.TrimPrefix(nolintTrimmed, ":")
			if len(colonTrimmed) == len(nolintTrimmed) {
				continue
			}

			nolintList := func() []string {
				list := strings.Split(colonTrimmed, ",")
				for i, linterName := range list {
					list[i] = strings.TrimSpace(linterName)
				}

				return list
			}()

			if slices.Contains(nolintList, nolintAll) || slices.Contains(nolintList, nolintName) {
				return true
			}
		}
	}

	return false
}
