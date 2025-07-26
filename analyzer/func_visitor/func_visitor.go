package func_visitor

import (
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"slices"
	"strings"

	"github.com/m-ocean-it/correcterr/analyzer/utils"
	"golang.org/x/tools/go/analysis"
)

const (
	nolintDirective = "nolint"
	nolintName      = "correcterr"
	nolintAll       = "all"
)

type anySet = map[any]struct{}

type funcVisitor struct {
	pass                   *analysis.Pass
	commentMap             ast.CommentMap
	checkedErrorDecls      anySet
	wraps                  map[*ast.Ident]*ast.Ident
	maybeCurentIfStmtScope *types.Scope
}

func New(pass *analysis.Pass, commentMap ast.CommentMap) ast.Visitor {
	return &funcVisitor{
		pass:                   pass,
		commentMap:             commentMap,
		checkedErrorDecls:      make(anySet),
		wraps:                  make(map[*ast.Ident]*ast.Ident),
		maybeCurentIfStmtScope: nil,
	}
}

func cloneVisitor(
	v *funcVisitor,
	checkedErrorDecls anySet,
	wraps map[*ast.Ident]*ast.Ident,
	maybeIfStmtScope *types.Scope,
) *funcVisitor {
	return &funcVisitor{
		pass:                   v.pass,
		commentMap:             v.commentMap,
		checkedErrorDecls:      checkedErrorDecls,
		wraps:                  wraps,
		maybeCurentIfStmtScope: maybeIfStmtScope,
	}
}

func (fv *funcVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.IfStmt:
		return visitIfStmt(fv, n)
	case *ast.ReturnStmt:
		return visitReturnStmt(fv, n)
	default:
		return fv
	}
}

func visitIfStmt(visitor *funcVisitor, ifStmt *ast.IfStmt) *funcVisitor {
	checkedErrDecls := visitor.checkedErrorDecls

	maybeCheckedError := getMaybeCheckedErrorDeclFromIfCondition(visitor.pass, ifStmt)
	if maybeCheckedError != nil {
		checkedErrDecls = maps.Clone(checkedErrDecls)
		checkedErrDecls[maybeCheckedError] = struct{}{}
	}

	return cloneVisitor(
		visitor,
		checkedErrDecls,
		visitor.wraps,
		visitor.pass.TypesInfo.Scopes[ifStmt],
	)
}

func visitReturnStmt(visitor *funcVisitor, returnStmt *ast.ReturnStmt) *funcVisitor {
	if retStmtCommentGroup, ok := visitor.commentMap[returnStmt]; ok {
		if checkCommentGroupsForNoLint(retStmtCommentGroup) {
			return visitor
		}
	}

	for _, expr := range returnStmt.Results {
		if !utils.ExprIsError(expr, visitor.pass.TypesInfo) {
			continue
		}

		switch e := expr.(type) {
		case *ast.Ident:
			checkReturnErrIdent(visitor, e)
		case *ast.CallExpr:
			checkReturnErrCall(visitor, e)
		}
	}

	return nil
}

func getMaybeCheckedErrorDeclFromIfCondition(pass *analysis.Pass, ifStmt *ast.IfStmt) any {
	binaryCondition, _ := ifStmt.Cond.(*ast.BinaryExpr)
	if binaryCondition == nil {
		return nil
	}

	if binaryCondition.Op != token.NEQ {
		return nil
	}

	if !utils.ExprIsError(binaryCondition.X, pass.TypesInfo) {
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

func checkReturnErrIdent(v *funcVisitor, returnIdent *ast.Ident) bool {
	if len(v.checkedErrorDecls) == 0 {
		return true
	}

	if v.maybeCurentIfStmtScope != nil && v.maybeCurentIfStmtScope.Contains(returnIdent.Obj.Pos()) {
		return true
	}

	returnIdentDecl := returnIdent.Obj.Decl

	_, wasChecked := v.checkedErrorDecls[returnIdentDecl]

	if wasChecked {
		return true
	}

	v.pass.Reportf(returnIdent.Pos(), "returning not the error that was checked")

	return false
}

func checkReturnErrCall(v *funcVisitor, call *ast.CallExpr) bool {
	var hasErrors bool

	for _, arg := range call.Args {
		if !utils.ExprIsError(arg, v.pass.TypesInfo) {
			continue
		}
		hasErrors = true

		switch errArg := arg.(type) {
		case *ast.Ident:
			if checkReturnErrIdent(v, errArg) {
				return true
			}
		case *ast.CallExpr:
			if checkReturnErrCall(v, errArg) {
				return true
			}
		case *ast.SelectorExpr:
			if checkReturnErrIdent(v, errArg.Sel) {
				return true
			}
		}
	}

	if hasErrors {
		return false
	}

	return true
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
