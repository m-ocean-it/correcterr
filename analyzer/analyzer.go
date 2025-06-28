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

type state struct {
	pass       *analysis.Pass
	errNames   errorNames
	wraps      map[string]stringSet
	commentMap ast.CommentMap
}

type errorNames struct {
	funcScope      stringSet
	immediateScope stringSet
	checked        stringSet
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
			pass: pass,
			errNames: errorNames{
				funcScope:      make(stringSet),
				checked:        make(stringSet),
				immediateScope: make(stringSet),
			},
			wraps:      make(map[string]stringSet),
			commentMap: commentMap,
		}

		inspectStatements(st, funcNode.Body.List)
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

func inspectStatements(st state, statements []ast.Stmt) {
	newLocalErrNames, newWraps := getLocalErrorNames(statements, st.pass)
	if len(newLocalErrNames) > 0 {
		st.errNames.funcScope = maps.Clone(st.errNames.funcScope)
		maps.Copy(st.errNames.funcScope, newLocalErrNames)

		st.wraps = cloneStringToStringSetMap(st.wraps)
		for k, v := range newWraps {
			if st.wraps[k] == nil {
				st.wraps[k] = v
			} else {
				maps.Copy(st.wraps[k], v)
			}
		}
	}
	st.errNames.immediateScope = newLocalErrNames

	for _, stmt := range statements {
		inspectStatement(st, stmt)
	}
}

func inspectStatement(st state, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		inspectIfStmt(st, s)
	case *ast.SwitchStmt:
		inspectSwitchStmt(st, s)
	case *ast.ForStmt:
		inspectForStmt(st, s)
	case *ast.RangeStmt:
		inspectRangeStmt(st, s)
	case *ast.ExprStmt:
		inspectExprStmt(st, s)
	case *ast.AssignStmt:
		inspectAssignStmt(st, s)
	case *ast.DeclStmt:
		inspectDeclStmt(st, s)
	case *ast.ReturnStmt:
		inspectReturnStmt(st, s)
	}
}

func inspectIfStmt(st state, ifStmt *ast.IfStmt) {
	maybeCheckedErr := tryGetCheckedErrFromIfStmt(st.pass, ifStmt)
	if maybeCheckedErr != nil {
		st.errNames.checked = maps.Clone(st.errNames.checked)
		st.errNames.checked[maybeCheckedErr.Name] = struct{}{}
	}

	inspectStatements(st, ifStmt.Body.List)
}

func inspectSwitchStmt(st state, switchStmt *ast.SwitchStmt) {
	for _, stmt := range switchStmt.Body.List {
		caseClause, _ := stmt.(*ast.CaseClause)
		if caseClause == nil {
			continue
		}

		inspectStatements(st, caseClause.Body)
	}
}

func inspectForStmt(st state, forStmt *ast.ForStmt) {
	inspectStatements(st, forStmt.Body.List)
}

func inspectRangeStmt(st state, rangeStmt *ast.RangeStmt) {
	inspectStatements(st, rangeStmt.Body.List)
}

func inspectExprStmt(st state, exprStmt *ast.ExprStmt) {
	inspectExpr(st, exprStmt.X)
}

func inspectExpr(st state, expr ast.Expr) {
	switch x := expr.(type) {
	case *ast.CallExpr:
		inspectCallExpr(st, x)
	case *ast.FuncLit:
		inspectFuncLit(st, x)
	}
}

func inspectCallExpr(st state, callExpr *ast.CallExpr) {
	funcLit, _ := callExpr.Fun.(*ast.FuncLit)
	if funcLit != nil {
		inspectFuncLit(st, funcLit)
	}

	for _, arg := range callExpr.Args {
		inspectExpr(st, arg)
	}
}

func inspectFuncLit(st state, funcLit *ast.FuncLit) {
	inspectStatements(st, funcLit.Body.List)
}

func inspectAssignStmt(st state, assignStmt *ast.AssignStmt) {
	for _, rightExpr := range assignStmt.Rhs {
		inspectExpr(st, rightExpr)
	}
}

func inspectDeclStmt(st state, declStmt *ast.DeclStmt) {
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
			inspectExpr(st, expr)
		}
	}
}

func inspectReturnStmt(st state, retStmt *ast.ReturnStmt) {
	if retStmtCommentGroup, ok := st.commentMap[retStmt]; ok {
		if checkCommentGroupsForNoLint(retStmtCommentGroup) {
			return
		}
	}

	var hasErrors bool

	for _, res := range retStmt.Results {
		if !exprIsError(res, st.pass.TypesInfo) {
			continue
		}
		hasErrors = true

		switch returnVal := res.(type) {

		case *ast.Ident:
			if returnedErrIsFine(st, returnVal.Name) {
				return
			}

		case *ast.CallExpr:
			if inspectCall(st, returnVal) {
				return
			}

		case *ast.SelectorExpr:
			if returnedErrIsFine(st, returnVal.Sel.Name) {
				return
			}

		default:
			return
		}
	}

	if hasErrors {
		st.pass.Reportf(retStmt.Pos(), "returning not the error that was checked")
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

func inspectCall(st state, call *ast.CallExpr) bool {
	var hasErrors bool

	for _, arg := range call.Args {
		if !exprIsError(arg, st.pass.TypesInfo) {
			continue
		}
		hasErrors = true

		switch errArg := arg.(type) {
		case *ast.Ident:
			if returnedErrIsFine(st, errArg.Name) {
				return true
			}
		case *ast.CallExpr:
			if inspectCall(st, errArg) {
				return true
			}
		case *ast.SelectorExpr:
			if returnedErrIsFine(st, errArg.Sel.Name) {
				return true
			}
		}
	}

	if hasErrors {
		return false
	}

	return true
}

func returnedErrIsFine(st state, errName string) bool {
	if len(st.errNames.checked) == 0 {
		return true
	}

	return returnedErrIsFineInner(st, errName, make(stringSet))
}

func returnedErrIsFineInner(st state, errName string, alreadyChecked stringSet) bool {
	if _, ok := alreadyChecked[errName]; ok {
		return false
	}

	if len(st.errNames.checked) == 0 {
		return true
	}

	if _, ok := st.errNames.checked[errName]; ok {
		return true
	}

	if _, ok := st.errNames.funcScope[errName]; !ok {
		return true
	}

	if _, ok := st.errNames.immediateScope[errName]; ok {
		return true
	}

	alreadyChecked = maps.Clone(alreadyChecked)
	alreadyChecked[errName] = struct{}{}

	wrappedNames, ok := st.wraps[errName]
	if !ok {
		return false
	}

	for wrName := range wrappedNames {
		if returnedErrIsFineInner(st, wrName, alreadyChecked) {
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
