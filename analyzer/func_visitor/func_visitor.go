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

type FuncVisitor struct {
	pass              *analysis.Pass
	commentMap        ast.CommentMap
	checkedErrorDecls anySet
	wraps             map[*ast.Ident][]*ast.Ident

	maybeCurentIfStmtScope *types.Scope
	errScopes              map[*ast.Ident]*types.Scope
}

func New(pass *analysis.Pass, commentMap ast.CommentMap) *FuncVisitor {
	return &FuncVisitor{
		pass:                   pass,
		commentMap:             commentMap,
		checkedErrorDecls:      make(anySet),
		wraps:                  make(map[*ast.Ident][]*ast.Ident),
		maybeCurentIfStmtScope: nil,
	}
}

func cloneVisitor(
	v *FuncVisitor,
	checkedErrorDecls anySet,
	wraps map[*ast.Ident][]*ast.Ident,
	maybeIfStmtScope *types.Scope,
) *FuncVisitor {
	return &FuncVisitor{
		pass:                   v.pass,
		commentMap:             v.commentMap,
		checkedErrorDecls:      checkedErrorDecls,
		wraps:                  wraps,
		maybeCurentIfStmtScope: maybeIfStmtScope,
	}
}

func (fv *FuncVisitor) Visit(node ast.Node, _ bool, stack []ast.Node) bool {
	if node == nil {
		return false
	}

	return fv.visit(node, stack)
}

func (fv *FuncVisitor) visit(node ast.Node, stack []ast.Node) bool {
	switch n := node.(type) {
	case *ast.Ident:
		fv.visitIdent(n, stack)
	case *ast.IfStmt:
		visitIfStmt(fv, n)
	case *ast.ReturnStmt:
		visitReturnStmt(fv, n)
	case *ast.AssignStmt:
		visitAssignStmt(fv, n)
	}

	return true
}

func (fv *FuncVisitor) visitIdent(ident *ast.Ident, stack []ast.Node) {

}

func visitAssignStmt(v *FuncVisitor, assignStmt *ast.AssignStmt) {
	var leftErrors, rightErrors *ast.Ident

	for _, leftExpr := range assignStmt.Lhs {
		if !utils.ExprIsError(leftExpr, v.pass.TypesInfo) {
			continue
		}

		leftIdent, ok := leftExpr.(*ast.Ident)
		if !ok {
			continue
		}

		// v.errScopes[leftIdent] =

		_ = leftIdent
	}

	_ = leftErrors
	_ = rightErrors
}

func visitIfStmt(visitor *FuncVisitor, ifStmt *ast.IfStmt) *FuncVisitor {
	checkedErrDecls := visitor.checkedErrorDecls

	maybeCheckedError := getMaybeCheckedErrorDeclFromIfCondition(visitor.pass, ifStmt)
	if maybeCheckedError != nil {
		checkedErrDecls = maps.Clone(checkedErrDecls)
		checkedErrDecls[maybeCheckedError] = struct{}{}
	}

	return cloneVisitor( // TODO надо соотносить это со стэком нодов
		// В данном случае if-scope как бы крепиться к какому-то уровню стэка.
		// Если стэк сократиться больше этого уровня, то инфа об ифе
		// должна быть забыта.
		visitor,
		checkedErrDecls,
		visitor.wraps,
		visitor.pass.TypesInfo.Scopes[ifStmt],
	)
}

func visitReturnStmt(visitor *FuncVisitor, returnStmt *ast.ReturnStmt) *FuncVisitor {
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

func checkReturnErrIdent(v *FuncVisitor, returnIdent *ast.Ident) bool {
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

	// returnIdentDecl

	// TODO check wraps

	v.pass.Reportf(returnIdent.Pos(), "returning not the error that was checked")

	return false
}

func checkReturnErrCall(v *FuncVisitor, call *ast.CallExpr) bool {
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
