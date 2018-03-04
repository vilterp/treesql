package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/package/util"
)

func FormatPlan(p PlanNode) string {
	buf := util.NewIndentBuffer("  ")
	buf.Printlnf("[")
	buf.Indent()
	p.Format(buf)
	buf.Dedent()
	buf.Printlnf("]")
	return buf.String()
}

type Expr struct {
	// only one of these is set. sigh.
	Var   string
	Value Value
}

func (e *Expr) Format() string {
	if e.Var != "" {
		return e.Var
	}
	return e.Value.Format()
}

type PlanNode interface {
	GetResults() map[string]interface{}

	Format(buf *util.IndentBuffer)
}

type selections struct {
	selectColumns []string
	childNodes    map[string]PlanNode
}

func (s *selections) Format(tableName string, buf *util.IndentBuffer) {
	buf.Printlnf("yield {")
	buf.Indent()
	for _, colName := range s.selectColumns {
		buf.Printlnf("%s: row.%s,", colName, colName)
	}
	for selectionName, childNode := range s.childNodes {
		buf.Printlnf("%s: [", selectionName)
		buf.Indent()
		childNode.Format(buf)
		buf.Dedent()
		buf.Printlnf("],")
	}
	buf.Dedent()
	buf.Printlnf("}")
}

type FullScanNode struct {
	table  *TableDescriptor
	filter *Filter

	selections
}

var _ PlanNode = &FullScanNode{}

func (s *FullScanNode) Format(buf *util.IndentBuffer) {
	buf.Printlnf("for row in %s.by_%s {", s.table.Name, s.table.PrimaryKey)
	if s.filter != nil {
		buf.Indent()
		buf.Printlnf("if %s:", s.filter.Format())
	}
	buf.Indent()
	s.selections.Format(s.table.Name, buf)
	buf.Dedent()
	buf.Printlnf("}")
}

func (s *FullScanNode) GetResults() map[string]interface{} {
	return nil
}

type IndexScanNode struct {
	table   *TableDescriptor
	colName string

	// An expression which will be evaluated in the scope
	// above this. This scan will return a row if
	// table.Indexes[colID][
	matchExpr Expr

	selections
}

var _ PlanNode = &IndexScanNode{}

func (s *IndexScanNode) Format(buf *util.IndentBuffer) {
	buf.Printlnf(
		"for row in %s.by_%s[row.%s] {",
		s.table.Name, s.colName, s.matchExpr.Format(),
	)
	buf.Indent()
	s.selections.Format(s.table.Name, buf)
	buf.Dedent()
	buf.Printlnf("}")
}

func (s *IndexScanNode) GetResults() map[string]interface{} {
	return nil
}

type Filter struct {
	left  Expr
	right Expr
}

func (f *Filter) Format() string {
	return fmt.Sprintf("%s = %s", f.left.Format(), f.right.Format())
}

func Plan(query *Select) (PlanNode, error) {
	return nil, nil
}
