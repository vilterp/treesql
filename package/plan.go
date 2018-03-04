package treesql

import (
	"bytes"
	"fmt"
	"strings"
)

func FormatPlan(p PlanNode) string {
	buf := bytes.NewBufferString("")
	p.Format(buf, 0, 0, 0)
	buf.WriteString("return results0\n")
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

	Format(buf *bytes.Buffer, depth int, nextRowVar int, nextResultsVar int)
}

type FullScanNode struct {
	table  *TableDescriptor
	filter *Filter

	// Really these should all be expressions...
	selectColumns []string
	childNodes    map[string]PlanNode
}

var _ PlanNode = &FullScanNode{}

func (s *FullScanNode) Format(
	buf *bytes.Buffer, depth int, nextRowVar int, nextResultsVar int,
) {
	buf.WriteString(fmt.Sprintf(
		"%sresults%d = []\n",
		strings.Repeat("  ", depth), nextResultsVar,
	))
	buf.WriteString(fmt.Sprintf(
		"%sfor row%d in %s.indexes.%s:\n",
		strings.Repeat("  ", depth), nextRowVar, s.table.Name, s.table.PrimaryKey),
	)
	if s.filter != nil {
		depth++
		buf.WriteString(fmt.Sprintf(
			"%sif %s:\n",
			strings.Repeat("  ", depth), s.filter.Format(),
		))
	}
	depth++
	buf.WriteString(fmt.Sprintf(
		"%sresult = {}\n", strings.Repeat("  ", depth),
	))
	for _, colName := range s.selectColumns {
		buf.WriteString(fmt.Sprintf(
			"%sresult.%s = row%d.%s\n",
			strings.Repeat("  ", depth), colName, nextRowVar, colName,
		))
	}
	for name, childNode := range s.childNodes {
		givenResultsVar := nextResultsVar + 1
		childNode.Format(buf, depth, nextRowVar+1, givenResultsVar)
		buf.WriteString(fmt.Sprintf(
			"%sresult.%s = results%d\n",
			strings.Repeat("  ", depth), name, givenResultsVar,
		))
	}
	// TODO: fill in selection
	buf.WriteString(fmt.Sprintf(
		"%sresults%d.append(result)\n",
		strings.Repeat("  ", depth), nextResultsVar,
	))
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

	selectColumns []string
	childNodes    map[string]PlanNode
}

var _ PlanNode = &IndexScanNode{}

func (s *IndexScanNode) Format(buf *bytes.Buffer, depth int, nextRowVar int, nextResultsVar int) {
	// TODO: probably can't just subtract one
	buf.WriteString(fmt.Sprintf(
		"%sresults%d = []\n",
		strings.Repeat("  ", depth), nextResultsVar,
	))
	buf.WriteString(fmt.Sprintf(
		"%sfor row%d in %s.indexes.%s[row%d.%s]:\n",
		strings.Repeat("  ", depth), nextRowVar, s.table.Name, s.colName, nextResultsVar-1, s.matchExpr.Format()),
	)
	depth++
	// TODO: this is pretty much entirely the same as FullScanNode's format
	buf.WriteString(fmt.Sprintf(
		"%sresult = {}\n", strings.Repeat("  ", depth),
	))
	for _, colName := range s.selectColumns {
		buf.WriteString(fmt.Sprintf(
			"%sresult.%s = row%d.%s\n",
			strings.Repeat("  ", depth), colName, nextRowVar, colName,
		))
	}
	for name, childNode := range s.childNodes {
		givenResultsVar := nextResultsVar + 1
		childNode.Format(buf, depth, nextRowVar+1, givenResultsVar)
		buf.WriteString(fmt.Sprintf(
			"%sresult.%s = results%d\n",
			strings.Repeat("  ", depth), name, givenResultsVar,
		))
	}
	// TODO: fill in selection
	buf.WriteString(fmt.Sprintf(
		"%sresults%d.append(result)\n",
		strings.Repeat("  ", depth), nextResultsVar,
	))
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
