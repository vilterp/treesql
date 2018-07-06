package main

import (
	"flag"

	p "github.com/vilterp/treesql/pkg/parserlib"
	"github.com/vilterp/treesql/pkg/parserlib_test_harness"
)

var port = flag.String("port", "9999", "port to listen on")

func main() {
	flag.Parse()

	g, err := p.NewGrammar(map[string]p.Rule{
		"expr": p.Choice([]p.Rule{
			p.Ref("call"),
			p.Ref("var"),
		}),
		"call": p.Sequence([]p.Rule{
			p.Ref("var"),
			p.Keyword("("),
			p.Keyword(")"),
		}),
		"var": p.Choice([]p.Rule{
			p.Keyword("foo"),
			p.Keyword("bar"),
		}),
	})
	if err != nil {
		panic(err)
	}

	parserlib_test_harness.NewServer(*port, g, "call")
}
