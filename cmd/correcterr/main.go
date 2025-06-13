package main

import (
	_ "flag"

	"github.com/m-ocean-it/correcterr/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
