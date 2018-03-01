package parserlib

import (
	"fmt"
	"strings"
)

type TraceTree struct {
	RuleID   RuleID
	StartPos Position
	EndPos   Position

	// If it's a choice node.
	ChoiceIdx   int
	ChoiceTrace *TraceTree
	// If it's a sequence
	AtItemIdx  int
	ItemTraces []*TraceTree
	// If it's a regex
	RegexMatch string
	// If it's a ref
	RefTrace *TraceTree
}

func (tt *TraceTree) String(g *Grammar) string {
	if tt == nil {
		return "<nil>"
	}
	return fmt.Sprintf("<%s => %s>", tt.stringInner(g), tt.EndPos.CompactString())
}

func (tt *TraceTree) stringInner(g *Grammar) string {
	rule := g.ruleForID[tt.RuleID]
	switch tRule := rule.(type) {
	case *choice:
		return fmt.Sprintf("CHOICE %d %s", tt.ChoiceIdx, tt.ChoiceTrace.String(g))
	case *sequence:
		seqTraces := make([]string, len(tt.ItemTraces))
		for idx, itemTrace := range tt.ItemTraces {
			seqTraces[idx] = itemTrace.String(g)
		}
		return fmt.Sprintf("SEQ [%s]", strings.Join(seqTraces, ", "))
	case *keyword:
		return fmt.Sprintf("KW %#v", tRule.value)
	case *regex:
		return fmt.Sprintf(`REGEX "%s"`, tt.RegexMatch)
	case *ref:
		return fmt.Sprintf("REF %s %s", tRule.name, tt.RefTrace)
	case *succeed:
		return "<succeed>"
	default:
		panic(fmt.Sprintf("unimplemented: %T", rule))
	}
}
