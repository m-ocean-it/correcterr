package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"go/ast"
	"go/types"

	_ "github.com/pkg/errors"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}

	testdata := filepath.Join(filepath.Dir(filepath.Dir(wd)), "correcterr/testdata")
	analysistest.Run(t, testdata, Analyzer, "pkg")
}

func Test_callIsErrDotErrorOnTarget(t *testing.T) {
	t.Parallel()

	callExp := &ast.CallExpr{
		Fun: &ast.SelectorExpr{},
		Args: []ast.Expr{
			&ast.Ident{
				Name: "err",
			},
		},
	}

	testCases := []struct {
		name    string
		call    *ast.CallExpr
		targets []*ast.Ident
		types   *types.Info
		expect  bool
	}{
		{
			name: "err.Error()",
			call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "err",
					},
					Sel: &ast.Ident{
						Name: "Error",
					},
				},
			},
			targets: []*ast.Ident{
				{Name: "err"},
			},
			expect: true,
		},
		{
			name: "Wrap(err).Error()",
			call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: callExp,
					Sel: &ast.Ident{
						Name: "Error",
					},
				},
			},
			targets: []*ast.Ident{
				{Name: "err"},
			},
			types: &types.Info{
				Types: map[ast.Expr]types.TypeAndValue{
					callExp: {
						Type: types.NewNamed(types.NewTypeName(1, nil, "error", nil), nil, nil),
					},
				},
			},
			expect: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := callIsErrDotErrorOnTarget(tc.call, tc.targets, tc.types)

			if got != tc.expect {
				t.FailNow()
			}
		})
	}
}
