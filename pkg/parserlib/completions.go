package parserlib

func (tt *TraceTree) GetCompletions() ([]string, error) {
	rule := tt.grammar.ruleForID[tt.RuleID]
	switch tRule := rule.(type) {
	case *choice:
		// TODO: we sometimes want to return multiple choices here...
		// maybe only if we're on the left edge
		if tt.CursorPos == 0 {
			return tRule.Completions(tt.grammar, tt.CursorPos), nil
		}
		return tt.ChoiceTrace.GetCompletions()
	case *sequence:
		return tt.ItemTraces[tt.AtItemIdx].GetCompletions()
	case *keyword:
		if tt.CursorPos == 0 {
			return []string{tRule.value}, nil
		}
		return []string{}, nil
	case *ref:
		return tt.RefTrace.GetCompletions()
	default:
		return []string{}, nil
	}
}

func (m *mapper) Completions(g *Grammar, cursor int) []string {
	return m.innerRule.Completions(g, cursor)
}

func (c *choice) Completions(g *Grammar, cursor int) []string {
	var out []string
	for _, choice := range c.choices {
		out = append(out, choice.Completions(g, cursor)...)
	}
	return out
}

func (s *sequence) Completions(_ *Grammar, _ int) []string {
	// TODO: which index are we at? maybe a rule method
	// is the wrong way to do this
	return []string{}
}

func (k *keyword) Completions(_ *Grammar, _ int) []string {
	return []string{k.value}
}

func (r *ref) Completions(g *Grammar, cursor int) []string {
	rule := g.rules[r.name]
	return rule.Completions(g, cursor)
}

func (r *regex) Completions(_ *Grammar, _ int) []string {
	// TODO: derive minimum value that passes regex?
	// get rid of regexes altogether and just build them
	// using the parser itself?
	return []string{}
}

func (s *succeed) Completions(_ *Grammar, _ int) []string {
	return []string{}
}
