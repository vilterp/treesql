package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/package/util"
)

func FormatPlan(p PlanNode) string {
	buf := util.NewIndentBuffer("  ")
	varNums := VarNums{
		nextResultsVar: 0,
		nextRowVar:     0,
	}
	p.Format(buf, varNums)
	buf.Printlnf("return results0")
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

	Format(buf *util.IndentBuffer, nums VarNums) VarNums
}

type selections struct {
	selectColumns []string
	childNodes    map[string]PlanNode
}

type VarNums struct {
	nextRowVar     int
	nextResultsVar int
}

func (s *selections) Format(buf *util.IndentBuffer, nums VarNums) VarNums {
	givenResultVar := nums.nextResultsVar
	buf.Printlnf("result = {")
	buf.Indent()
	for _, colName := range s.selectColumns {
		buf.Printlnf("%s: row%d.%s,", colName, nums.nextRowVar, colName)
	}
	buf.Dedent()
	buf.Printlnf("}")
	nums.nextRowVar++
	for name, childNode := range s.childNodes {
		buf.Printlnf("# %s", name)
		nums.nextResultsVar++
		nextResultVar := nums.nextResultsVar
		nums = childNode.Format(buf, nums)
		buf.Printlnf("result.%s = results%d", name, nextResultVar)
	}
	buf.Printlnf("results%d.append(result)", givenResultVar)
	return nums
}

type FullScanNode struct {
	table  *TableDescriptor
	filter *Filter

	selections
}

var _ PlanNode = &FullScanNode{}

func (s *FullScanNode) Format(buf *util.IndentBuffer, nums VarNums) VarNums {
	buf.Printlnf("results%d = []", nums.nextResultsVar)
	buf.Printlnf("for row%d in %s.indexes.%s:", nums.nextRowVar, s.table.Name, s.table.PrimaryKey)
	if s.filter != nil {
		buf.Indent()
		buf.Printlnf("if %s:", s.filter.Format())
	}
	buf.Indent()
	s.selections.Format(buf, nums)
	buf.Dedent()
	return nums
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

func (s *IndexScanNode) Format(buf *util.IndentBuffer, nums VarNums) VarNums {
	buf.Printlnf("results%d = []", nums.nextResultsVar)
	buf.Printlnf(
		"for row%d in %s.indexes.%s[row%d.%s]:",
		nums.nextRowVar, s.table.Name, s.colName, nums.nextRowVar-1, s.matchExpr.Format(),
	)
	buf.Indent()
	s.selections.Format(buf, nums)
	buf.Dedent()
	return nums
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
