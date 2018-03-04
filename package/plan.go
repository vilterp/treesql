package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/package/util"
)

func FormatPlan(p PlanNode) string {
	buf := util.NewIndentBuffer("  ")
	varName := p.Format(buf)
	buf.Printlnf("return %s", varName)
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

	Format(buf *util.IndentBuffer) string
}

type selections struct {
	selectColumns []string
	childNodes    map[string]PlanNode
}

func (s *selections) Format(tableName string, buf *util.IndentBuffer) {
	buf.Printlnf("%s_result = {", tableName)
	buf.Indent()
	for _, colName := range s.selectColumns {
		buf.Printlnf("%s: row.%s,", colName, colName)
	}
	buf.Dedent()
	buf.Printlnf("}")
	for selectionName, childNode := range s.childNodes {
		buf.Printlnf("# %s", selectionName)
		varName := childNode.Format(buf)
		buf.Printlnf("%s_result.%s = %s", tableName, selectionName, varName)
	}
	buf.Printlnf("%s_results.append(%s_result)", tableName, tableName)
}

type FullScanNode struct {
	table  *TableDescriptor
	filter *Filter

	selections
}

var _ PlanNode = &FullScanNode{}

func (s *FullScanNode) Format(buf *util.IndentBuffer) string {
	buf.Printlnf("%s_results = []", s.table.Name)
	buf.Printlnf("for row in %s.indexes.%s:", s.table.Name, s.table.PrimaryKey)
	if s.filter != nil {
		buf.Indent()
		buf.Printlnf("if %s:", s.filter.Format())
	}
	buf.Indent()
	s.selections.Format(s.table.Name, buf)
	buf.Dedent()
	return fmt.Sprintf("%s_results", s.table.Name)
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

func (s *IndexScanNode) Format(buf *util.IndentBuffer) string {
	buf.Printlnf("%s_results = []", s.table.Name)
	buf.Printlnf(
		"for row in %s.indexes.%s[row.%s]:",
		s.table.Name, s.colName, s.matchExpr.Format(),
	)
	buf.Indent()
	s.selections.Format(s.table.Name, buf)
	buf.Dedent()
	return fmt.Sprintf("%s_results", s.table.Name)
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
