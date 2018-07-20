package main

import (
	"flag"

	"fmt"
	"log"
	"net/http"

	"github.com/vilterp/treesql/pkg/parserlib_test_harness"
	"github.com/vilterp/treesql/pkg/treesql_lang"
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

	server := parserlib_test_harness.NewServer(treesql_lang.Language)

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("serving on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatal(err)
	}
}
