package parserlib

// return the grammar in a format where all rules are resolved to IDs

type SerializedRule struct {
	RuleType string

	Choices   []RuleID `json:",omitempty"`
	SeqItems  []RuleID `json:",omitempty"`
	InnerRule RuleID
	Ref       string `json:",omitempty"`
	Regex     string `json:",omitempty"`
	Keyword   string `json:",omitempty"`
}

type SerializedGrammar struct {
	TopLevelRules map[string]RuleID
	RulesByID     map[RuleID]SerializedRule
}

func (g *Grammar) Serialize() *SerializedGrammar {
	sg := &SerializedGrammar{
		RulesByID:     make(map[RuleID]SerializedRule),
		TopLevelRules: make(map[string]RuleID),
	}
	for name, rule := range g.rules {
		sg.TopLevelRules[name] = g.idForRule[rule]
	}
	for id, rule := range g.ruleForID {
		sg.RulesByID[id] = rule.Serialize(g)
	}
	return sg
}

func (m *mapper) Serialize(g *Grammar) SerializedRule {
	return SerializedRule{
		RuleType:  "MAP",
		InnerRule: g.idForRule[m.innerRule],
	}
}

func (c *choice) Serialize(g *Grammar) SerializedRule {
	choices := make([]RuleID, len(c.choices))
	for idx, choice := range c.choices {
		choices[idx] = g.idForRule[choice]
	}
	return SerializedRule{
		RuleType: "CHOICE",
		Choices:  choices,
	}
}

func (s *sequence) Serialize(g *Grammar) SerializedRule {
	items := make([]RuleID, len(s.items))
	for idx, choice := range s.items {
		items[idx] = g.idForRule[choice]
	}
	return SerializedRule{
		RuleType: "SEQUENCE",
		SeqItems: items,
	}
}

func (k *keyword) Serialize(g *Grammar) SerializedRule {
	return SerializedRule{
		RuleType: "KEYWORD",
		Keyword:  k.value,
	}
}

func (r *ref) Serialize(g *Grammar) SerializedRule {
	return SerializedRule{
		RuleType: "REF",
		Ref:      r.name,
	}
}

func (r *regex) Serialize(g *Grammar) SerializedRule {
	return SerializedRule{
		RuleType: "REGEX",
		Regex:    r.regex.String(),
	}
}

func (s *succeed) Serialize(g *Grammar) SerializedRule {
	return SerializedRule{
		RuleType: "SUCCEED",
	}
}
