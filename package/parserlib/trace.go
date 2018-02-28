package parserlib

import (
	"fmt"
	"strings"
)

type TraceTree struct {
	rule   Rule
	endPos Position

	// If it's a choice node.
	choiceIdx   int
	choiceTrace *TraceTree
	// If it's a sequence
	atItemIdx  int
	itemTraces []*TraceTree
	// If it's a regex
	regexMatch string
	// If it's a ref
	refTrace *TraceTree
}

func (tt *TraceTree) String() string {
	if tt == nil {
		return "<nil>"
	}
	return fmt.Sprintf("<%s => %s>", tt.stringInner(), tt.endPos.CompactString())
}

func (tt *TraceTree) stringInner() string {
	switch tRule := tt.rule.(type) {
	case *choice:
		return fmt.Sprintf("CHOICE %d %s", tt.choiceIdx, tt.choiceTrace.String())
	case *sequence:
		seqTraces := make([]string, len(tt.itemTraces))
		for idx, itemTrace := range tt.itemTraces {
			seqTraces[idx] = itemTrace.String()
		}
		return fmt.Sprintf("SEQ [%s]", strings.Join(seqTraces, ", "))
	case *keyword:
		return fmt.Sprintf("KW %#v", tRule.value)
	case *regex:
		return fmt.Sprintf(`REGEX "%s"`, tt.regexMatch)
	case *ref:
		return fmt.Sprintf("REF %s %s", tRule.name, tt.refTrace)
	case *succeed:
		return "<succeed>"
	default:
		panic(fmt.Sprintf("unimplemented: %T", tt.rule))
	}
}
