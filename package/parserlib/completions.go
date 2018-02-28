package parserlib

import "fmt"

func (g *Grammar) GetCompletions(rule string, input string) ([]string, error) {
	trace, err := g.Parse(rule, input)
	switch err.(type) {
	case *ParseError:
		break
	default:
		return nil, err
	}
	switch tRule := trace.rule.(type) {
	case *choice:
		return tRule.Completions(g), nil
	case *sequence:
		stoppedAtRule := tRule.items[trace.atItemIdx]
		return stoppedAtRule.Completions(g), nil
	default:
		panic(fmt.Sprintf("unimplemented: %T", trace.rule))
	}
}

func (c *choice) Completions(g *Grammar) []string {
	var out []string
	for _, choice := range c.choices {
		out = append(out, choice.Completions(g)...)
	}
	return out
}

func (s *sequence) Completions(_ *Grammar) []string {
	// TODO: which index are we at? maybe a rule method
	// is the wrong way to do this
	return []string{}
}

func (k *keyword) Completions(_ *Grammar) []string {
	return []string{k.value}
}

func (r *ref) Completions(g *Grammar) []string {
	rule := g.rules[r.name]
	return rule.Completions(g)
}

func (r *regex) Completions(_ *Grammar) []string {
	// TODO: derive minimum value that passes regex?
	// get rid of regexes altogether and just build them
	// using the parser itself?
	return []string{}
}

func (s *succeed) Completions(_ *Grammar) []string {
	return []string{}
}
