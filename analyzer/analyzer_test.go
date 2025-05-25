package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/pkg/errors"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}

	testdata := filepath.Join(filepath.Dir(filepath.Dir(wd)), "correcterr/testdata")
	analysistest.Run(t, testdata, Analyzer, "pkg")
}
