package correcterr

import (
	"github.com/m-ocean-it/correcterr/analyzer"
	"golang.org/x/tools/go/analysis"
)

func NewAnalyzer() *analysis.Analyzer {
	return analyzer.Analyzer
}
