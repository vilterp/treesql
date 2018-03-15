package parserlib

import (
	"fmt"

	pp "github.com/vilterp/treesql/pkg/prettyprint"
)

type TraceTree struct {
	grammar *Grammar

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

func (tt *TraceTree) Format() pp.Doc {
	rule := tt.grammar.ruleForID[tt.RuleID]

	switch tRule := rule.(type) {
	case *choice:
		return pp.Seq([]pp.Doc{
			pp.Textf("CHOICE(%d, ", tt.ChoiceIdx),
			pp.Newline,
			pp.Nest(2, tt.ChoiceTrace.Format()),
			pp.Newline,
			pp.Text(")"),
		})
	case *sequence:
		seqDocs := make([]pp.Doc, len(tt.ItemTraces))
		for idx, item := range tt.ItemTraces {
			seqDocs[idx] = item.Format()
		}
		return pp.Seq([]pp.Doc{
			pp.Text("SEQUENCE("),
			pp.Newline,
			pp.Nest(2, pp.Join(seqDocs, pp.CommaNewline)),
			pp.Newline,
			pp.Text(")"),
		})
	case *regex:
		return pp.Textf("REGEX(%#v)", tt.RegexMatch)
	case *succeed:
		return pp.Text("SUCCESS")
	case *ref:
		return pp.Seq([]pp.Doc{
			pp.Textf("REF(%s,", tRule.name),
			pp.Newline,
			pp.Nest(2, tt.RefTrace.Format()),
			pp.Newline,
			pp.Text(")"),
		})
	case *keyword:
		return pp.Textf("%#v", tRule.value)
	case *mapper:
		return pp.Seq([]pp.Doc{
			pp.Text("MAP("),
			pp.Newline,
			pp.Nest(2, tt.InnerTrace.Format()),
			pp.Newline,
			pp.Text(")"),
		})
	default:
		panic(fmt.Sprintf("don't know how to format a %T trace", rule))
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
		return results
	}
	if tt.Success {
		return nil
	}
	return nil
}

func (tt *TraceTree) GetListRes() []interface{} {
	// Get list ref.
	anyItemsChoice := tt
	// Return empty array if there's nothing.
	if anyItemsChoice.ChoiceIdx == 1 {
		return []interface{}{}
	}
	return anyItemsChoice.ChoiceTrace.GetList1Res()
}

func (tt *TraceTree) GetList1Res() []interface{} {
	justOneItemChoice := tt
	// If there's just one item, return it.
	if justOneItemChoice.ChoiceIdx == 1 {
		return []interface{}{
			justOneItemChoice.ChoiceTrace.GetMapRes(),
		}
	}
	// Otherwise, there are at least one items.
	out := make([]interface{}, 1)
	// Get the first item.
	seqTrace := justOneItemChoice.ChoiceTrace
	refTrace := seqTrace.ItemTraces[0].RefTrace
	out[0] = refTrace.GetMapRes()
	// Now get the rest.
	rest := seqTrace.ItemTraces[2].RefTrace.InnerTrace.GetListRes()
	out = append(out, rest...)
	return out
}

func (tt *TraceTree) OptWhitespaceSurroundRes() *TraceTree {
	whitespaceSeq := tt
	return whitespaceSeq.ItemTraces[1]
}
