package parserlib

import (
	"fmt"
	"strings"
)

type TraceTree struct {
	Rule   Rule
	EndPos Position

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

func (tt *TraceTree) String() string {
	if tt == nil {
		return "<nil>"
	}
	return fmt.Sprintf("<%s => %s>", tt.stringInner(), tt.EndPos.CompactString())
}

func (tt *TraceTree) stringInner() string {
	switch tRule := tt.Rule.(type) {
	case *choice:
		return fmt.Sprintf("CHOICE %d %s", tt.ChoiceIdx, tt.ChoiceTrace.String())
	case *sequence:
		seqTraces := make([]string, len(tt.ItemTraces))
		for idx, itemTrace := range tt.ItemTraces {
			seqTraces[idx] = itemTrace.String()
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
		panic(fmt.Sprintf("unimplemented: %T", tt.Rule))
	}
}
