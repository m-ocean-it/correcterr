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

		var (
			localErrNames   = make(map[string]struct{})
			checkedErrNames = make(map[string]struct{})
		)

		inspectStatements(pass, localErrNames, checkedErrNames, commentMap, funcNode.Body.List)
	})

	return nil, nil
}

func getLocalErrorNames(statements []ast.Stmt) map[string]struct{} {
	names := make(map[string]struct{})

	for _, stmt := range statements {
		switch s := stmt.(type) {

		case *ast.DeclStmt:
			for _, errName := range getErrorNamesFromDeclStmt(s) {
				names[errName] = struct{}{}
			}

		case *ast.AssignStmt:
			for _, errName := range getErrorNamesFromAssignStmt(s) {
				names[errName] = struct{}{}
			}
		}
	}

	return names
}

func getErrorNamesFromDeclStmt(decl *ast.DeclStmt) []string {
	genDecl, _ := decl.Decl.(*ast.GenDecl)
	if genDecl == nil {
		return nil
	}

	if genDecl.Tok != token.VAR {
		return nil
	}

	var names []string

	for _, spec := range genDecl.Specs {
		valSpec, _ := spec.(*ast.ValueSpec)
		if valSpec == nil {
			continue
		}

		for _, name := range valSpec.Names {
			names = append(names, name.Name)
		}
	}

	return names
}

func getErrorNamesFromAssignStmt(assign *ast.AssignStmt) []string {
	var names []string

	for _, leftExpr := range assign.Lhs {
		leftIdent, _ := leftExpr.(*ast.Ident)
		if leftIdent != nil {
			names = append(names, leftIdent.Name)
		}

	}

	return names
}

func inspectStatements(
	pass *analysis.Pass,
	localErrNames, checkedErrorNames map[string]struct{},
	commentMap ast.CommentMap,
	statements []ast.Stmt,
) {
	newLocalErrNames := getLocalErrorNames(statements)
	if len(newLocalErrNames) > 0 {
		localErrNames = copyMap(localErrNames)
		maps.Copy(localErrNames, newLocalErrNames)
	}

	for _, stmt := range statements {
		inspectStatement(pass, localErrNames, checkedErrorNames, commentMap, stmt)
	}
}

func inspectStatement(
	pass *analysis.Pass,
	localErrNames, checkedErrorNames map[string]struct{},
	commentMap ast.CommentMap,
	stmt ast.Stmt,
) {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		inspectIfStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	case *ast.SwitchStmt:
		inspectSwitchStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	case *ast.ForStmt:
		inspectForStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	case *ast.RangeStmt:
		inspectRangeStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	case *ast.ExprStmt:
		inspectExprStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	case *ast.AssignStmt:
		inspectAssignStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	case *ast.DeclStmt:
		inspectDeclStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	case *ast.ReturnStmt:
		inspectReturnStmt(pass, localErrNames, checkedErrorNames, commentMap, s)
	}
}

func inspectIfStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	ifStmt *ast.IfStmt,
) {
	maybeCheckedErr := tryGetCheckedErrFromIfStmt(pass, ifStmt)
	if maybeCheckedErr != nil {
		checkedErrNames = copyMap(checkedErrNames)
		checkedErrNames[maybeCheckedErr.Name] = struct{}{}
	}

	inspectStatements(pass, localErrNames, checkedErrNames, commentMap, ifStmt.Body.List)
}

func inspectSwitchStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	switchStmt *ast.SwitchStmt,
) {
	for _, stmt := range switchStmt.Body.List {
		caseClause, _ := stmt.(*ast.CaseClause)
		if caseClause == nil {
			continue
		}

		inspectStatements(pass, localErrNames, checkedErrNames, commentMap, caseClause.Body)
	}
}

func inspectForStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	forStmt *ast.ForStmt,
) {
	inspectStatements(pass, localErrNames, checkedErrNames, commentMap, forStmt.Body.List)
}

func inspectRangeStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	rangeStmt *ast.RangeStmt,
) {
	inspectStatements(pass, localErrNames, checkedErrNames, commentMap, rangeStmt.Body.List)
}

func inspectExprStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	exprStmt *ast.ExprStmt,
) {
	inspectExpr(pass, localErrNames, checkedErrNames, commentMap, exprStmt.X)
}

func inspectExpr(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	expr ast.Expr,
) {
	switch x := expr.(type) {
	case *ast.CallExpr:
		inspectCallExpr(pass, localErrNames, checkedErrNames, commentMap, x)
	case *ast.FuncLit:
		inspectFuncLit(pass, localErrNames, checkedErrNames, commentMap, x)
	}
}

func inspectCallExpr(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	callExpr *ast.CallExpr,
) {
	funcLit, _ := callExpr.Fun.(*ast.FuncLit)
	if funcLit != nil {
		inspectFuncLit(pass, localErrNames, checkedErrNames, commentMap, funcLit)
	}

	for _, arg := range callExpr.Args {
		inspectExpr(pass, localErrNames, checkedErrNames, commentMap, arg)
	}
}

func inspectFuncLit(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	funcLit *ast.FuncLit,
) {
	inspectStatements(pass, localErrNames, checkedErrNames, commentMap, funcLit.Body.List)
}

func inspectAssignStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	assignStmt *ast.AssignStmt,
) {
	for _, rightExpr := range assignStmt.Rhs {
		inspectExpr(pass, localErrNames, checkedErrNames, commentMap, rightExpr)
	}
}

func inspectDeclStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	declStmt *ast.DeclStmt,
) {
	genDecl, _ := declStmt.Decl.(*ast.GenDecl)
	if genDecl == nil {
		return
	}

	if genDecl.Tok != token.VAR {
		return
	}

	for _, spec := range genDecl.Specs {
		valSpec, _ := spec.(*ast.ValueSpec)
		if valSpec == nil {
			continue
		}

		for _, expr := range valSpec.Values {
			inspectExpr(pass, localErrNames, checkedErrNames, commentMap, expr)
		}
	}
}

func inspectReturnStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	commentMap ast.CommentMap,
	retStmt *ast.ReturnStmt,
) {
	if retStmtCommentGroup, ok := commentMap[retStmt]; ok {
		if checkCommentGroupsForNoLint(retStmtCommentGroup) {
			return
		}
	}

	for _, res := range retStmt.Results {
		if !exprIsError(res, pass.TypesInfo) {
			continue
		}

		var toReport bool

		switch returnVal := res.(type) {

		case *ast.Ident:
			if len(checkedErrNames) > 0 &&
				!mapHas(checkedErrNames, returnVal.Name) &&
				mapHas(localErrNames, returnVal.Name) {

				toReport = true
			}

		case *ast.CallExpr:
			if inspectCall(pass, localErrNames, checkedErrNames, returnVal) {
				toReport = true
			}
		}

		if toReport {
			pass.Reportf(retStmt.Pos(), "returning not the error that was checked")

			return
		}
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

func inspectCall(
	pass *analysis.Pass,
	localErrNames, checkedErrNames map[string]struct{},
	call *ast.CallExpr,
) bool {
	for _, arg := range call.Args {
		if !exprIsError(arg, pass.TypesInfo) {
			continue
		}

		switch errArg := arg.(type) {
		case *ast.Ident:
			if len(checkedErrNames) > 0 &&
				!mapHas(checkedErrNames, errArg.Name) &&
				mapHas(localErrNames, errArg.Name) {

				return true
			}
		case *ast.CallExpr:
			if inspectCall(pass, localErrNames, checkedErrNames, errArg) {
				return true
			}
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
