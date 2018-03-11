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
	ChoiceTrace *TraceTree `json:",omitempty"`
	// If it's a sequence
	AtItemIdx  int
	ItemTraces []*TraceTree `json:",omitempty"`
	// If it's a regex
	RegexMatch string
	// If it's a ref
	RefTrace *TraceTree `json:",omitempty"`
	// If it's a mapper
	InnerTrace *TraceTree `json:",omitempty"`
	MapRes     interface{}
	// If it's a success
	Success bool
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

func (tt *TraceTree) GetMapRes() interface{} {
	if tt.MapRes != nil {
		return tt.MapRes
	}
	if tt.RefTrace != nil {
		return tt.RefTrace.GetMapRes()
	}
	if tt.ChoiceTrace != nil {
		return tt.ChoiceTrace.GetMapRes()
	}
	if tt.ItemTraces != nil {
		results := make([]interface{}, len(tt.ItemTraces))
		for idx, thing := range tt.ItemTraces {
			results[idx] = thing.GetMapRes()
		}
	}
	if tt.Success {
		return nil
	}
	//panic(fmt.Sprintf("can't get map res for %+v", tt))
	return nil
}
