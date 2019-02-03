package parserlib

import (
	"fmt"
	"strings"
)

// TODO: structured parse errors
// each one has a position
// print out with position
// maybe store whole trace

type ParserState struct {
	grammar *Grammar

	input string

	stack []*ParserStackFrame

	trace *TraceTree
}

type ParserStackFrame struct {
	input string
	// position we're at, exclusive
	// TODO: record start pos
	pos Position

	rule Rule
}

func (g *Grammar) Parse(startRuleName string, input string, cursor int) (*TraceTree, error) {
	ps := ParserState{
		grammar: g,
		input:   input,
	}
	initPos := Position{Line: 1, Col: 1, Offset: 0}
	startRule, ok := ps.grammar.rules[startRuleName]
	if !ok {
		return nil, fmt.Errorf("nonexistent start rule: %s", startRuleName)
	}
	traceTree, err := ps.callRule(startRule, initPos, cursor)
	if err != nil {
		return traceTree, err
	}
	if traceTree.EndPos.Offset != len(input) {
		return traceTree, fmt.Errorf("%d extra chars at end of input", len(input)-traceTree.EndPos.Offset)
	}
	return traceTree, nil
}

func (ps *ParserState) callRule(rule Rule, pos Position, cursor int) (*TraceTree, *ParseError) {
	// Create and push stack frame.
	stackFrame := &ParserStackFrame{
		input: ps.input,
		rule:  rule,
		pos:   pos,
	}
	ps.stack = append(ps.stack, stackFrame)
	// Run the rule.
	traceTree, err := ps.runRule(cursor)
	// Pop the stack frame.
	ps.stack = ps.stack[:len(ps.stack)-1]
	if traceTree == nil {
		panic(fmt.Sprintf("nil trace tree returned for rule %v", rule))
	}
	// Return.
	if err != nil {
		return traceTree, err
	}
	return traceTree, nil
}

func (sf *ParserStackFrame) Errorf(
	innerErr *ParseError, fmtString string, params ...interface{},
) *ParseError {
	return &ParseError{
		input:    sf.input,
		innerErr: innerErr,
		msg:      fmt.Sprintf(fmtString, params...),
		pos:      sf.pos,
	}
}

func (ps *ParserState) runRule(cursor int) (*TraceTree, *ParseError) {
	frame := ps.stack[len(ps.stack)-1]
	rule := frame.rule
	startPos := frame.pos
	minimalTrace := &TraceTree{
		grammar:   ps.grammar,
		RuleID:    ps.grammar.idForRule[rule],
		StartPos:  startPos,
		CursorPos: 0,
		EndPos:    startPos,
	}
	switch tRule := rule.(type) {
	case *choice:
		trace := &TraceTree{
			grammar:   ps.grammar,
			RuleID:    ps.grammar.idForRule[rule],
			StartPos:  startPos,
			CursorPos: cursor,
		}
		maxAdvancement := 0
		maxAdvancementTraceIndex := 0
		var maxAdvancementTrace *TraceTree
		for choiceIdx, choice := range tRule.choices {
			choiceTrace, err := ps.callRule(choice, frame.pos, cursor)
			advancement := choiceTrace.EndPos.Offset - choiceTrace.StartPos.Offset
			if advancement >= maxAdvancement {
				maxAdvancement = advancement
				maxAdvancementTrace = choiceTrace
				maxAdvancementTraceIndex = choiceIdx
			}
			if err == nil {
				// We found a match!
				trace.EndPos = choiceTrace.EndPos
				trace.ChoiceIdx = choiceIdx
				trace.ChoiceTrace = choiceTrace
				return trace, nil
			}
		}
		trace.EndPos = maxAdvancementTrace.EndPos
		trace.ChoiceIdx = maxAdvancementTraceIndex
		trace.ChoiceTrace = maxAdvancementTrace
		return trace, frame.Errorf(nil, "no match for rule `%s`", rule.String())
	case *sequence:
		trace := &TraceTree{
			grammar:    ps.grammar,
			RuleID:     ps.grammar.idForRule[rule],
			StartPos:   startPos,
			CursorPos:  cursor,
			ItemTraces: make([]*TraceTree, len(tRule.items)),
		}
		advancement := 0
		// bla [foo, bar, baz]
		// ---  ---  ---  ---
		//   3
		//            ^
		for itemIdx, item := range tRule.items {
			trace.AtItemIdx = itemIdx
			itemTrace, err := ps.callRule(item, frame.pos, cursor-advancement)
			advancement += itemTrace.GetSpan().Length()
			trace.EndPos = itemTrace.EndPos
			trace.ItemTraces[itemIdx] = itemTrace
			if err != nil {
				return trace, frame.Errorf(err, "no match for sequence item %d", itemIdx)
			}
			frame.pos = itemTrace.EndPos
		}
		trace.EndPos = frame.pos
		return trace, nil
	case *keyword:
		remainingInput := ps.input[frame.pos.Offset:]
		if len(tRule.value) > len(remainingInput) {
			trimmed := strings.TrimPrefix(tRule.value, remainingInput)
			if len(trimmed) < len(tRule.value) {
				minimalTrace.EndPos = minimalTrace.StartPos.MoreOnLine(len(trimmed))
			}
			return minimalTrace, frame.Errorf(
				nil, `expected "%s"; got "%s"<EOF>`, tRule.value, remainingInput,
			)
		}
		trimmed := strings.TrimPrefix(remainingInput, tRule.value)
		advancement := len(remainingInput) - len(trimmed)
		minimalTrace.EndPos = minimalTrace.StartPos.MoreOnLine(advancement)
		if advancement == len(tRule.value) {
			minimalTrace.CursorPos = cursor
			return minimalTrace, nil
		}
		return minimalTrace, frame.Errorf(nil, `expected "%s"; got "%s"`, tRule.value, remainingInput)
	case *ref:
		refRule, ok := ps.grammar.rules[tRule.name]
		if !ok {
			panic(fmt.Sprintf("nonexistent rule slipped through validation: %s", tRule.name))
		}
		refTrace, err := ps.callRule(refRule, frame.pos, cursor)
		minimalTrace.RefTrace = refTrace
		minimalTrace.EndPos = refTrace.EndPos
		if err != nil {
			return minimalTrace, frame.Errorf(err, `no match for rule "%s"`, tRule.name)
		}
		return &TraceTree{
			grammar:   ps.grammar,
			RuleID:    ps.grammar.idForRule[rule],
			StartPos:  startPos,
			CursorPos: cursor,
			EndPos:    refTrace.EndPos,
			RefTrace:  refTrace,
		}, nil
	case *regex:
		loc := tRule.regex.FindStringIndex(ps.input[frame.pos.Offset:])
		if loc == nil || loc[0] != 0 {
			return minimalTrace, frame.Errorf(nil, "no match found for regex %s", tRule.regex)
		}
		matchText := ps.input[frame.pos.Offset : frame.pos.Offset+loc[1]]
		endPos := frame.pos
		for _, char := range matchText {
			if char == '\n' {
				endPos = endPos.Newline()
			} else {
				endPos = endPos.MoreOnLine(1)
			}
		}
		return &TraceTree{
			grammar:    ps.grammar,
			RuleID:     ps.grammar.idForRule[rule],
			StartPos:   startPos,
			CursorPos:  cursor,
			EndPos:     endPos,
			RegexMatch: matchText,
		}, nil
	case *mapper:
		innerTrace, err := ps.callRule(tRule.innerRule, frame.pos, cursor)
		minimalTrace.InnerTrace = innerTrace
		minimalTrace.EndPos = innerTrace.EndPos
		if err != nil {
			return minimalTrace, err
		}
		res := tRule.fun(innerTrace)
		minimalTrace.MapRes = res
		return minimalTrace, nil
	case *succeed:
		minimalTrace.Success = true
		return minimalTrace, nil
	default:
		panic(fmt.Sprintf("not implemented: %T", rule))
	}
}

func max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
