package parserlib

import (
	"fmt"
	"strings"
)

type ParserState struct {
	grammar *Grammar

	input string

	stack []*ParserStackFrame
}

type Position struct {
	Line   int
	Col    int
	Offset int
}

func (pos *Position) String() string {
	return fmt.Sprintf("line %d, col %d", pos.Line, pos.Col)
}

func (pos *Position) MoreOnLine(n int) Position {
	return Position{
		Col:    pos.Col + n,
		Line:   pos.Line,
		Offset: pos.Offset + n,
	}
}

func (pos *Position) Newline() Position {
	return Position{
		Col:    1,
		Line:   pos.Line + 1,
		Offset: pos.Offset + 1,
	}
}

type ParserStackFrame struct {
	// position we're at, exclusive
	// TODO: record start pos
	pos Position

	rule Rule

	// If it's a choice rule
	//choiceIdx int
	//// If it's a sequence rule
	//currentItem int
	//items       [][]*ParserStackFrame
	//// If it's a regex rule
	//matchedValue string
	//// if it's a ref rule
}

func (psf *ParserStackFrame) String() string {
	// TODO: rule-specific state
	return fmt.Sprintf("%s %s", psf.pos, psf.rule)
}

// TODO: return something other than just an error or not
func Parse(g *Grammar, startRuleName string, input string) error {
	ps := ParserState{
		grammar: g,
		input:   input,
	}
	initPos := Position{Line: 1, Col: 1, Offset: 0}
	startRule, ok := ps.grammar.rules[startRuleName]
	if !ok {
		return fmt.Errorf("nonexistent start rule: %s", startRuleName)
	}
	ps.pushRule(startRule, initPos)
	newPos, err := ps.step()
	ps.popRule()
	if err != nil {
		return err
	}
	if newPos.Offset != len(input) {
		return fmt.Errorf("%d extra chars at end of input", len(input)-newPos.Offset)
	}
	return nil
}

func (ps *ParserState) pushRule(rule Rule, pos Position) {
	fmt.Printf("%spush %s (%s)\n", ps.indent(), rule, pos.String())
	stackFrame := &ParserStackFrame{
		rule: rule,
		pos:  pos,
	}
	// TODO: record a bunch of info in the stack that we can use later!
	//switch tRule := rule.(type) {
	//case *Choice:
	//	//stackFrame.choiceIdx = 0
	//case *Sequence:
	//	//stackFrame.currentItem = 0
	//	//stackFrame.items = make([][]*ParserStackFrame, len(tRule.Items))
	//case *Keyword:
	//case *Regex:
	//}
	ps.stack = append(ps.stack, stackFrame)
}

func (ps *ParserState) indent() string {
	return strings.Repeat("  ", len(ps.stack))
}

func (ps *ParserState) popRule() {
	fmt.Printf("%spop\n", ps.indent())
	ps.stack = ps.stack[:len(ps.stack)-1]
}

// just for logging purposes
func (ps *ParserState) step() (Position, error) {
	indent := ps.indent()
	pos, err := ps.doStep()
	if err != nil {
		fmt.Printf("%sstep: ERR %v\n", indent, err)
	} else {
		fmt.Printf("%sstep: MATCH newPos: %v\n", indent, pos)
	}
	return pos, err
}

func (ps *ParserState) doStep() (Position, error) {
	frame := ps.stack[len(ps.stack)-1]
	rule := frame.rule
	switch tRule := rule.(type) {
	case *Choice:
		for _, choice := range tRule.Choices {
			//frame.choiceIdx = choiceIdx
			ps.pushRule(choice, frame.pos)
			newPos, err := ps.step()
			ps.popRule()
			if err == nil {
				return newPos, nil
			}
		}
		return Position{}, fmt.Errorf("no match for rule `%s` at pos %v", rule.String(), frame.pos)
	case *Sequence:
		for itemIdx, item := range tRule.Items {
			ps.pushRule(item, frame.pos)
			newPos, err := ps.step()
			ps.popRule()
			if err != nil {
				return Position{}, fmt.Errorf(
					"no match for sequence item %d (`%s`): %v", itemIdx, item.String(), err,
				)
			}
			frame.pos = newPos
		}
		return frame.pos, nil
	case *Keyword:
		nextNChars := ps.input[frame.pos.Offset : frame.pos.Offset+len(tRule.Value)]
		if nextNChars == tRule.Value {
			return frame.pos.MoreOnLine(len(tRule.Value)), nil
		}
		return Position{}, fmt.Errorf(`expected "%s"; got "%s"`, tRule.Value, nextNChars)
	case *Ref:
		rule, ok := ps.grammar.rules[tRule.Name]
		if !ok {
			panic(fmt.Sprintf("nonexistent rule slipped through validation: %s", tRule.Name))
		}
		ps.pushRule(rule, frame.pos)
		newPos, err := ps.step()
		ps.popRule()
		if err != nil {
			return Position{}, fmt.Errorf("no match for rule %s: %v", tRule.Name, err)
		}
		return newPos, nil
	default:
		panic(fmt.Sprintf("not implemented: %s", rule.String()))
	}
	panic("shouldn't get here")
	return Position{}, nil
}
