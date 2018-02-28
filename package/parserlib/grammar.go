package parserlib

import (
	"fmt"
	"regexp"
	"strings"
)

type Grammar struct {
	rules map[string]Rule
}

func NewGrammar(rules map[string]Rule) (*Grammar, error) {
	g := &Grammar{rules: rules}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *Grammar) Validate() error {
	for ruleName, rule := range g.rules {
		if err := rule.Validate(g); err != nil {
			return fmt.Errorf(`in rule "%s": %v`, ruleName, err)
		}
	}
	return nil
}

func (g *Grammar) String() string {
	var rulesStrings []string
	for name, rule := range g.rules {
		rulesStrings = append(rulesStrings, fmt.Sprintf("%s: %s", name, rule))
	}
	return strings.Join(rulesStrings, "\n")
}

type Rule interface {
	String() string
	Validate(g *Grammar) error
	Completions(g *Grammar) []string
}

// choice

type choice struct {
	choices []Rule
}

var _ Rule = &choice{}

func Choice(choices []Rule) *choice {
	return &choice{
		choices: choices,
	}
}

func (c *choice) String() string {
	choicesStrs := make([]string, len(c.choices))
	for idx, choice := range c.choices {
		choicesStrs[idx] = choice.String()
	}
	return strings.Join(choicesStrs, " | ")
}

func (c *choice) Validate(g *Grammar) error {
	for idx, choice := range c.choices {
		if err := choice.Validate(g); err != nil {
			return fmt.Errorf("in choice %d: %v", idx, err)
		}
	}
	return nil
}

// sequence

type sequence struct {
	items []Rule
}

var _ Rule = &sequence{}

func Sequence(items []Rule) *sequence {
	return &sequence{
		items: items,
	}
}

func (s *sequence) String() string {
	itemsStrs := make([]string, len(s.items))
	for idx, item := range s.items {
		itemsStrs[idx] = item.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(itemsStrs, ", "))
}

func (s *sequence) Validate(g *Grammar) error {
	for idx, item := range s.items {
		if err := item.Validate(g); err != nil {
			return fmt.Errorf("in seq item %d: %v", idx, err)
		}
	}
	return nil
}

// keyword

type keyword struct {
	value string
}

var _ Rule = &keyword{}

// TODO: case insensitivity
func Keyword(value string) *keyword {
	return &keyword{
		value: value,
	}
}

func (k *keyword) String() string {
	return fmt.Sprintf(`"%s"`, k.value)
}

func (k *keyword) Validate(_ *Grammar) error {
	for _, char := range k.value {
		if char == '\n' {
			return fmt.Errorf("newlines not allowed in keywords: %v", k.value)
		}
	}
	return nil
}

// Rule ref

type ref struct {
	name string
}

var _ Rule = &ref{}

func Ref(name string) *ref {
	return &ref{
		name: name,
	}
}

func (r *ref) String() string {
	return string(r.name)
}

func (r *ref) Validate(g *Grammar) error {
	if _, ok := g.rules[r.name]; !ok {
		return fmt.Errorf(`ref not found: "%s"`, r.name)
	}
	return nil
}

// regex

type regex struct {
	regex *regexp.Regexp
}

var _ Rule = &regex{}

func Regex(re *regexp.Regexp) *regex {
	return &regex{
		regex: re,
	}
}

func (r *regex) String() string {
	return fmt.Sprintf("/%s/", r.regex.String())
}

func (r *regex) Validate(g *Grammar) error {
	return nil
}

// Succeed

var Succeed = &succeed{}

type succeed struct{}

var _ Rule = &succeed{}

func (s *succeed) String() string {
	return "<succeed>"
}

func (s *succeed) Validate(g *Grammar) error {
	return nil
}
