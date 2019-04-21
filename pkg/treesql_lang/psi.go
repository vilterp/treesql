package treesql_lang

import p "github.com/vilterp/treesql/pkg/parserlib"

var Language *p.Language

func init() {
	Language = &p.Language{
		Grammar:        Grammar,
		ParseTreeToPSI: traceTreeToPSI,
	}
}

func traceTreeToPSI(tt *p.TraceTree) p.PSINode {
	if tt.RefTrace != nil {
		// this is a named rule
		if Grammar.NameForID(tt.RuleID) == "select" { // TODO get actual rule id
			return extractManyQuery(tt.RefTrace)
		}
	}

	return &Query{}
}

func extractManyQuery(tt *p.TraceTree) p.PSINode {
	return &Query{
		span: tt.GetSpan(),
	}
}

type Query struct {
	tableName string
	isOne     bool
	span      p.SourceSpan

	selections []*Selection
}

var _ p.PSINode = &Query{}

func (q *Query) GetChildren() []p.PSINode {
	out := make([]p.PSINode, len(q.selections))
	for i, sel := range q.selections {
		out[i] = sel
	}
	return out
}

func (q *Query) GetName() string {
	if q.isOne {
		return "ONE"
	}
	return "MANY"
}

func (q *Query) GetSpan() p.SourceSpan {
	return q.span
}

type Selection struct {
	name  string
	query *Query // nil if we are just selecting a column
	span  p.SourceSpan
}

var _ p.PSINode = &Selection{}

func (s *Selection) GetChildren() []p.PSINode {
	return []p.PSINode{}
}

func (s *Selection) GetName() string {
	return "SELECTION"
}

func (s *Selection) GetSpan() p.SourceSpan {
	return s.span
}
