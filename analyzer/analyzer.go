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

type stringSet = map[string]struct{}

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
			localErrNames   = make(stringSet)
			checkedErrNames = make(stringSet)
			wraps           = make(map[string]stringSet)
		)

		inspectStatements(pass, localErrNames, checkedErrNames, wraps, commentMap, funcNode.Body.List)
	})

	return nil, nil
}

func getLocalErrorNames(statements []ast.Stmt, pass *analysis.Pass) (stringSet, map[string]stringSet) {
	names := make(stringSet)
	wraps := make(map[string]stringSet)

	for _, stmt := range statements {
		var (
			nms  []string
			wrps map[string]stringSet
		)

		switch s := stmt.(type) {
		case *ast.DeclStmt:
			nms, wrps = getErrorNamesFromDeclStmt(pass, s)
		case *ast.AssignStmt:
			nms, wrps = getErrorNamesFromAssignStmt(pass, s)
		}

		for _, nm := range nms {
			names[nm] = struct{}{}
		}
		for k, v := range wrps {
			if wraps[k] == nil {
				wraps[k] = v
			} else {
				maps.Copy(wraps[k], v)
			}
		}
	}

	return names, wraps
}

func getErrorNamesFromDeclStmt(pass *analysis.Pass, decl *ast.DeclStmt) ([]string, map[string]stringSet) {
	genDecl, _ := decl.Decl.(*ast.GenDecl)
	if genDecl == nil {
		return nil, nil
	}

	if genDecl.Tok != token.VAR {
		return nil, nil
	}

	var (
		names []string
		wraps = make(map[string]stringSet)
	)

	for _, spec := range genDecl.Specs {
		valSpec, _ := spec.(*ast.ValueSpec)
		if valSpec == nil {
			continue
		}

		for _, name := range valSpec.Names {
			names = append(names, name.Name)
		}

		var callErrNames []string
		for _, expr := range valSpec.Values {
			switch e := expr.(type) {
			case *ast.CallExpr:
				callErrNames = append(callErrNames, scanCallForErrNames(e, pass)...)
			case *ast.Ident:
				if exprIsError(e, pass.TypesInfo) {
					callErrNames = append(callErrNames, e.Name)
				}
			}
		}
		if len(callErrNames) > 0 {
			for _, name := range valSpec.Names {
				for _, callErrN := range callErrNames {
					if wraps[name.Name] == nil {
						wraps[name.Name] = stringSet{callErrN: struct{}{}}
					} else {
						wraps[name.Name][callErrN] = struct{}{}
					}
				}
			}
		}
	}

	return names, wraps
}

func getErrorNamesFromAssignStmt(pass *analysis.Pass, assign *ast.AssignStmt) ([]string, map[string]stringSet) {
	var names []string

	for _, leftExpr := range assign.Lhs {
		leftIdent, _ := leftExpr.(*ast.Ident)
		if leftIdent != nil {
			names = append(names, leftIdent.Name)
		}
	}

	wraps := getErrWrapsFromAssignStmt(pass, assign, names)

	return names, wraps
}

func getErrWrapsFromAssignStmt(pass *analysis.Pass, assign *ast.AssignStmt, names []string) map[string]stringSet {
	var rightErrNames []string

	for _, rightExpr := range assign.Rhs {
		switch expr := rightExpr.(type) {
		case *ast.CallExpr:
			rightErrNames = append(rightErrNames, scanCallForErrNames(expr, pass)...)
		case *ast.Ident:
			if exprIsError(expr, pass.TypesInfo) {
				rightErrNames = append(rightErrNames, expr.Name)
			}
		}
	}

	var wraps = make(map[string]stringSet)
	if len(rightErrNames) > 0 {
		for _, errName := range names {
			for _, callErrN := range rightErrNames {
				if wraps[errName] == nil {
					wraps[errName] = stringSet{callErrN: struct{}{}}
				} else {
					wraps[errName][callErrN] = struct{}{}
				}
			}
		}
	}

	return wraps
}

func scanCallForErrNames(call *ast.CallExpr, pass *analysis.Pass) []string {
	var errNames []string

	for _, arg := range call.Args {
		if !exprIsError(arg, pass.TypesInfo) {
			continue
		}

		switch typedArg := arg.(type) {
		case *ast.Ident:
			errNames = append(errNames, typedArg.Name)
		case *ast.CallExpr:
			errNames = append(errNames, scanCallForErrNames(typedArg, pass)...)
		case *ast.SelectorExpr:
			errNames = append(errNames, typedArg.Sel.Name)
		}
	}

	return errNames
}

func inspectStatements(
	pass *analysis.Pass,
	localErrNames, checkedErrorNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	statements []ast.Stmt,
) {
	newLocalErrNames, newWraps := getLocalErrorNames(statements, pass)
	if len(newLocalErrNames) > 0 {
		localErrNames = maps.Clone(localErrNames)
		maps.Copy(localErrNames, newLocalErrNames)

		wraps = cloneStringToStringSetMap(wraps)
		for k, v := range newWraps {
			if wraps[k] == nil {
				wraps[k] = v
			} else {
				maps.Copy(wraps[k], v)
			}
		}
	}

	for _, stmt := range statements {
		inspectStatement(pass, localErrNames, checkedErrorNames, wraps, commentMap, stmt)
	}
}

func inspectStatement(
	pass *analysis.Pass,
	localErrNames, checkedErrorNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	stmt ast.Stmt,
) {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		inspectIfStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	case *ast.SwitchStmt:
		inspectSwitchStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	case *ast.ForStmt:
		inspectForStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	case *ast.RangeStmt:
		inspectRangeStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	case *ast.ExprStmt:
		inspectExprStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	case *ast.AssignStmt:
		inspectAssignStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	case *ast.DeclStmt:
		inspectDeclStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	case *ast.ReturnStmt:
		inspectReturnStmt(pass, localErrNames, checkedErrorNames, wraps, commentMap, s)
	}
}

func inspectIfStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	ifStmt *ast.IfStmt,
) {
	maybeCheckedErr := tryGetCheckedErrFromIfStmt(pass, ifStmt)
	if maybeCheckedErr != nil {
		checkedErrNames = maps.Clone(checkedErrNames)
		checkedErrNames[maybeCheckedErr.Name] = struct{}{}
	}

	inspectStatements(pass, localErrNames, checkedErrNames, wraps, commentMap, ifStmt.Body.List)
}

func inspectSwitchStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	switchStmt *ast.SwitchStmt,
) {
	for _, stmt := range switchStmt.Body.List {
		caseClause, _ := stmt.(*ast.CaseClause)
		if caseClause == nil {
			continue
		}

		inspectStatements(pass, localErrNames, checkedErrNames, wraps, commentMap, caseClause.Body)
	}
}

func inspectForStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	forStmt *ast.ForStmt,
) {
	inspectStatements(pass, localErrNames, checkedErrNames, wraps, commentMap, forStmt.Body.List)
}

func inspectRangeStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	rangeStmt *ast.RangeStmt,
) {
	inspectStatements(pass, localErrNames, checkedErrNames, wraps, commentMap, rangeStmt.Body.List)
}

func inspectExprStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	exprStmt *ast.ExprStmt,
) {
	inspectExpr(pass, localErrNames, checkedErrNames, wraps, commentMap, exprStmt.X)
}

func inspectExpr(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	expr ast.Expr,
) {
	switch x := expr.(type) {
	case *ast.CallExpr:
		inspectCallExpr(pass, localErrNames, checkedErrNames, wraps, commentMap, x)
	case *ast.FuncLit:
		inspectFuncLit(pass, localErrNames, checkedErrNames, wraps, commentMap, x)
	}
}

func inspectCallExpr(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	callExpr *ast.CallExpr,
) {
	funcLit, _ := callExpr.Fun.(*ast.FuncLit)
	if funcLit != nil {
		inspectFuncLit(pass, localErrNames, checkedErrNames, wraps, commentMap, funcLit)
	}

	for _, arg := range callExpr.Args {
		inspectExpr(pass, localErrNames, checkedErrNames, wraps, commentMap, arg)
	}
}

func inspectFuncLit(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	funcLit *ast.FuncLit,
) {
	inspectStatements(pass, localErrNames, checkedErrNames, wraps, commentMap, funcLit.Body.List)
}

func inspectAssignStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	commentMap ast.CommentMap,
	assignStmt *ast.AssignStmt,
) {
	for _, rightExpr := range assignStmt.Rhs {
		inspectExpr(pass, localErrNames, checkedErrNames, wraps, commentMap, rightExpr)
	}
}

func inspectDeclStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
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
			inspectExpr(pass, localErrNames, checkedErrNames, wraps, commentMap, expr)
		}
	}
}

func inspectReturnStmt(
	pass *analysis.Pass,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
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
			if !returnedErrIsFine(returnVal.Name, localErrNames, checkedErrNames, wraps) {
				toReport = true
			}

		case *ast.CallExpr:
			if inspectCall(pass, localErrNames, checkedErrNames, wraps, returnVal) {
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
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	call *ast.CallExpr,
) bool {
	for _, arg := range call.Args {
		if !exprIsError(arg, pass.TypesInfo) {
			continue
		}

		switch errArg := arg.(type) {
		case *ast.Ident:
			if !returnedErrIsFine(errArg.Name, localErrNames, checkedErrNames, wraps) {
				return true
			}
		case *ast.CallExpr:
			if inspectCall(pass, localErrNames, checkedErrNames, wraps, errArg) {
				return true
			}
		}
	}

	return false
}

func returnedErrIsFine(
	errName string,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
) bool {
	if len(checkedErrNames) == 0 {
		return true
	}

	return returnedErrIsFineInner(errName, localErrNames, checkedErrNames, wraps, make(stringSet))
}

func returnedErrIsFineInner(
	errName string,
	localErrNames, checkedErrNames stringSet,
	wraps map[string]stringSet,
	alreadyChecked stringSet,
) bool {
	if _, ok := alreadyChecked[errName]; ok {
		return false
	}

	if _, ok := checkedErrNames[errName]; ok {
		return true
	}

	if _, ok := localErrNames[errName]; !ok {
		return true
	}

	alreadyChecked = maps.Clone(alreadyChecked)
	alreadyChecked[errName] = struct{}{}

	wrappedNames, ok := wraps[errName]
	if !ok {
		return false
	}

	for wrName := range wrappedNames {
		if returnedErrIsFineInner(wrName, localErrNames, checkedErrNames, wraps, alreadyChecked) {
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

func cloneStringToStringSetMap(m map[string]stringSet) map[string]stringSet {
	newM := make(map[string]stringSet)

	for k, v := range m {
		newM[k] = maps.Clone(v)
	}

	return newM

}
