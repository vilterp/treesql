package treesql_lang

import (
	"fmt"
	"testing"

	"github.com/vilterp/treesql/pkg/parserlib"
)

func TestPSI(t *testing.T) {
	tt, err := Language.Grammar.Parse("select", "MANY foo { bar, baz }", 0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tt.Format().Debug())
	node := Language.ParseTreeToPSI(tt)
	fmt.Printf("tree: %#v\n", node)
	t.Fatal(parserlib.PrintPSINode(node))
}
