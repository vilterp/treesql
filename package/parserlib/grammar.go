package parserlib

import (
	"fmt"
	"regexp"
	"strings"
)

type RuleName string

type Grammar struct {
	rules map[string]Rule
}

func (g *Grammar) Validate() error {
	for ruleName, rule := range g.rules {
		if err := rule.Validate(g); err != nil {
			return fmt.Errorf(`in rule "%s": %v`, ruleName, err)
		}
	}
	return nil
}

type Rule interface {
	String() string
	Validate(g *Grammar) error
}

// Choice

type Choice struct {
	Choices []Rule
}

var _ Rule = &Choice{}

func (c *Choice) String() string {
	choicesStrs := make([]string, len(c.Choices))
	for idx, choice := range c.Choices {
		choicesStrs[idx] = choice.String()
	}
	return strings.Join(choicesStrs, " | ")
}

func (c *Choice) Validate(g *Grammar) error {
	for idx, choice := range c.Choices {
		if err := choice.Validate(g); err != nil {
			return fmt.Errorf("in choice %d: %v", idx, err)
		}
	}
	return nil
}

// Sequence

type Sequence struct {
	Items []Rule
}

var _ Rule = &Sequence{}

func (s *Sequence) String() string {
	itemsStrs := make([]string, len(s.Items))
	for idx, item := range s.Items {
		itemsStrs[idx] = item.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(itemsStrs, ", "))
}

func (s *Sequence) Validate(g *Grammar) error {
	for idx, item := range s.Items {
		if err := item.Validate(g); err != nil {
			return fmt.Errorf("in seq item %d: %v", idx, err)
		}
	}
	return nil
}

// Keyword

type Keyword struct {
	Value string
}

var _ Rule = &Keyword{}

func (k *Keyword) String() string {
	return fmt.Sprintf(`"%s"`, k.Value)
}

func (k *Keyword) Validate(_ *Grammar) error {
	return nil
}

// Rule ref

type Ref struct {
	Name string
}

var _ Rule = &Ref{}

func (r *Ref) String() string {
	return r.Name
}

func (r *Ref) Validate(g *Grammar) error {
	if _, ok := g.rules[r.Name]; !ok {
		return fmt.Errorf(`ref not found: "%s"`, r.Name)
	}
	return nil
}

// Regex

type Regex struct {
	Regex regexp.Regexp
}

var _ Rule = &Regex{}

func (r *Regex) String() string {
	return fmt.Sprintf("/%s/", r.Regex.String())
}

func (r *Regex) Validate(g *Grammar) error {
	return nil
}
