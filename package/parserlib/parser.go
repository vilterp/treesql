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

func (ps *ParserState) callRule(rule Rule, pos Position) (*TraceTree, error) {
	// Create and push stack frame.
	stackFrame := &ParserStackFrame{
		rule: rule,
		pos:  pos,
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

func (ps *ParserState) indent() string {
	return strings.Repeat("  ", len(ps.stack))
}

func (ps *ParserState) runRule() (*TraceTree, error) {
	frame := ps.stack[len(ps.stack)-1]
	rule := frame.rule
	switch tRule := rule.(type) {
	case *Choice:
		for choiceIdx, choice := range tRule.Choices {
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
		return nil, fmt.Errorf(`no match for rule "%s" at pos %v`, rule.String(), frame.pos)
	case *Sequence:
		trace := &TraceTree{
			rule:       rule,
			itemTraces: make([]*TraceTree, len(tRule.Items)),
		}
		for itemIdx, item := range tRule.Items {
			itemTrace, err := ps.callRule(item, frame.pos)
			if err != nil {
				return nil, fmt.Errorf(
					"no match for sequence item %d: %v", itemIdx, err,
				)
			}
			frame.pos = itemTrace.endPos
			trace.itemTraces[itemIdx] = itemTrace
		}
		trace.endPos = frame.pos
		return trace, nil
	case *Keyword:
		nextNChars := ps.input[frame.pos.Offset : frame.pos.Offset+len(tRule.Value)]
		if nextNChars == tRule.Value {
			return &TraceTree{
				rule:   rule,
				endPos: frame.pos.MoreOnLine(len(tRule.Value)),
			}, nil
		}
		return nil, fmt.Errorf(`expected "%s"; got "%s"`, tRule.Value, nextNChars)
	case *Ref:
		refRule, ok := ps.grammar.rules[tRule.Name]
		if !ok {
			panic(fmt.Sprintf("nonexistent rule slipped through validation: %s", tRule.Name))
		}
		refTrace, err := ps.callRule(refRule, frame.pos)
		if err != nil {
			return nil, fmt.Errorf(`no match for rule "%s": %v`, tRule.Name, err)
		}
		return &TraceTree{
			rule:     rule,
			endPos:   refTrace.endPos,
			refTrace: refTrace,
		}, nil
	case *Regex:
		loc := tRule.Regex.FindStringIndex(ps.input[frame.pos.Offset:])
		if loc == nil || loc[0] != 0 {
			return nil, fmt.Errorf("no match found for regex %s", tRule.Regex)
		}
		matchText := ps.input[frame.pos.Offset : frame.pos.Offset+loc[1]]
		return &TraceTree{
			rule:       rule,
			endPos:     frame.pos.MoreOnLine(loc[1]),
			regexMatch: matchText,
		}, nil
	default:
		panic(fmt.Sprintf("not implemented: %T", rule))
	}
}
