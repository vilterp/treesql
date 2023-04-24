package main

import (
	"flag"

	"github.com/vilterp/treesql/pkg/lang"
	"github.com/vilterp/treesql/pkg/parserlib_test_harness"
)

var port = flag.String("port", "9999", "port to listen on")

func main() {
	flag.Parse()

	parserlib_test_harness.NewServer(*port, lang.Grammar, "expr")
}
