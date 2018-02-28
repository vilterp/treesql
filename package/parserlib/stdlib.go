package parserlib

import "regexp"

// TODO: this only does it once. need "repeat" combinator of some kind
func Intercalate(r Rule, sep Rule) Rule {
	return &Choice{
		Choices: []Rule{
			&Sequence{Items: []Rule{sep, r}},
			r, // TODO: recur here, not just r
		},
	}
}

func Opt(r Rule) Rule {
	return &Choice{
		Choices: []Rule{
			r,
			Succeed,
		},
	}
}

func WhitespaceSeq(items []Rule) Rule {
	// hoo, a generic intercalate function sure would be nice
	var outItems []Rule
	for idx, item := range items {
		if idx > 0 {
			outItems = append(outItems, Whitespace)
		}
		outItems = append(outItems, item)
	}
	return &Sequence{
		Items: outItems,
	}
}

var Whitespace = &Regex{Regex: regexp.MustCompile("\\s+")}

var UnsignedIntLit = &Regex{Regex: regexp.MustCompile("[0-9]+")}

var SignedIntLit = &Regex{Regex: regexp.MustCompile("-?[0-9]+")}

// Thank you https://stackoverflow.com/a/2039820
var StringLit = &Regex{Regex: regexp.MustCompile(`\"(\\.|[^"\\])*\"`)}

var Ident = &Regex{Regex: regexp.MustCompile("[a-zA-Z][a-zA-Z0-9_]+")}
