package main

import (
	"flag"

	"fmt"
	"log"
	"net/http"

	p "github.com/vilterp/treesql/pkg/parserlib"
	"github.com/vilterp/treesql/pkg/parserlib_test_harness"
)

var port = flag.String("port", "9999", "port to listen on")

func main() {
	flag.Parse()

	//g, err := p.NewGrammar(map[string]p.Rule{
	//	//"expr": p.Choice([]p.Rule{
	//	//	p.Ref("call"),
	//	//	p.Ref("var"),
	//	//}),
	//	//"call": p.Sequence([]p.Rule{
	//	//	p.Ref("var"),
	//	//	p.Keyword("("),
	//	//	p.Keyword(")"),
	//	//}),
	//	//"var": p.Choice([]p.Rule{
	//	//	p.Keyword("foo"),
	//	//	p.Keyword("bar"),
	//	//}),
	//	"expr": p.Sequence([]p.Rule{
	//		p.Keyword("foo"),
	//		p.Keyword("bar"),
	//		p.Choice([]p.Rule{
	//			p.Keyword("baz"),
	//			p.Keyword("bin"),
	//		}),
	//	}),
	//})
	//if err != nil {
	//	panic(err)
	//}
	g, err := p.TestTreeSQLGrammar()
	if err != nil {
		panic(err)
	}

	language := p.Language{
		Grammar: g,
		ParseTreeToPSI: func(tt *p.TraceTree) p.PSINode {
			return nil
		},
	}

	server := parserlib_test_harness.NewServer(language, "select")

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("serving on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}
