package parserlib

import (
	"fmt"
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

func (psf *ParserStackFrame) String() string {
	// TODO: rule-specific state
	return fmt.Sprintf("%s %s", psf.pos, psf.rule)
}

// TODO: return something other than just an error or not
func Parse(g *Grammar, startRuleName string, input string) (*TraceTree, error) {
	ps := ParserState{
		grammar: g,
		input:   input,
	}
	initPos := Position{Line: 1, Col: 1, Offset: 0}
	startRule, ok := ps.grammar.rules[startRuleName]
	if !ok {
		return nil, fmt.Errorf("nonexistent start rule: %s", startRuleName)
	}
	traceTree, err := ps.callRule(startRule, initPos)
	if err != nil {
		return nil, err
	}
	if traceTree.endPos.Offset != len(input) {
		return nil, fmt.Errorf("%d extra chars at end of input", len(input)-traceTree.endPos.Offset)
	}
	return traceTree, nil
}

func (ps *ParserState) callRule(rule Rule, pos Position) (*TraceTree, *ParseError) {
	// Create and push stack frame.
	stackFrame := &ParserStackFrame{
		input: ps.input,
		rule:  rule,
		pos:   pos,
	}
	ps.stack = append(ps.stack, stackFrame)
	// Run the rule.
	traceTree, err := ps.runRule()
	// Pop the stack frame.
	ps.stack = ps.stack[:len(ps.stack)-1]
	// Return.
	if err != nil {
		return nil, err
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

func (ps *ParserState) runRule() (*TraceTree, *ParseError) {
	frame := ps.stack[len(ps.stack)-1]
	rule := frame.rule
	switch tRule := rule.(type) {
	case *choice:
		for choiceIdx, choice := range tRule.choices {
			trace, err := ps.callRule(choice, frame.pos)
			if err == nil {
				// We found a match!
				return &TraceTree{
					rule:        rule,
					endPos:      trace.endPos,
					choiceIdx:   choiceIdx,
					choiceTrace: trace,
				}, nil
			}
		}
		return nil, frame.Errorf(nil, `no match for rule "%s"`, rule.String())
	case *sequence:
		trace := &TraceTree{
			rule:       rule,
			itemTraces: make([]*TraceTree, len(tRule.items)),
		}
		for itemIdx, item := range tRule.items {
			itemTrace, err := ps.callRule(item, frame.pos)
			if err != nil {
				return nil, frame.Errorf(err, "no match for sequence item %d", itemIdx)
			}
			frame.pos = itemTrace.endPos
			trace.itemTraces[itemIdx] = itemTrace
		}
		trace.endPos = frame.pos
		return trace, nil
	case *keyword:
		inputLeft := len(ps.input) - frame.pos.Offset
		if len(tRule.value) > inputLeft {
			return nil, frame.Errorf(
				nil, `expected "%s"; got "%s"<EOF>`, tRule.value, ps.input[frame.pos.Offset:],
			)
		}
		nextNChars := ps.input[frame.pos.Offset : frame.pos.Offset+len(tRule.value)]
		if nextNChars == tRule.value {
			return &TraceTree{
				rule:   rule,
				endPos: frame.pos.MoreOnLine(len(tRule.value)),
			}, nil
		}
		return nil, frame.Errorf(nil, `expected "%s"; got "%s"`, tRule.value, nextNChars)
	case *ref:
		refRule, ok := ps.grammar.rules[tRule.name]
		if !ok {
			panic(fmt.Sprintf("nonexistent rule slipped through validation: %s", tRule.name))
		}
		refTrace, err := ps.callRule(refRule, frame.pos)
		if err != nil {
			return nil, frame.Errorf(err, `no match for rule "%s"`, tRule.name)
		}
		return &TraceTree{
			rule:     rule,
			endPos:   refTrace.endPos,
			refTrace: refTrace,
		}, nil
	case *regex:
		loc := tRule.regex.FindStringIndex(ps.input[frame.pos.Offset:])
		if loc == nil || loc[0] != 0 {
			return nil, frame.Errorf(nil, "no match found for regex %s", tRule.regex)
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
			rule:       rule,
			endPos:     endPos,
			regexMatch: matchText,
		}, nil
	case *succeed:
		return &TraceTree{
			rule:   rule,
			endPos: frame.pos,
		}, nil
	default:
		panic(fmt.Sprintf("not implemented: %T", rule))
	}
}
