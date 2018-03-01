package parserlib

import "regexp"

func ListRule(ruleName string, listName string, sep Rule) Rule {
	return Choice([]Rule{
		Sequence([]Rule{
			Ref(ruleName),
			sep,
			Ref(listName),
		}),
		Ref(ruleName),
	})
}

func Opt(r Rule) Rule {
	return &choice{
		choices: []Rule{
			r,
			Succeed,
		},
	}
}

var OptWhitespace = Opt(Whitespace)

func WhitespaceSeq(items []Rule) Rule {
	// hoo, a generic intercalate function sure would be nice
	var outItems []Rule
	for idx, item := range items {
		if idx > 0 {
			outItems = append(outItems, Whitespace)
		}
		outItems = append(outItems, item)
	}
	return &sequence{
		items: outItems,
	}
}

func OptWhitespaceSeq(items []Rule) Rule {
	// hoo, a generic intercalate function sure would be nice
	var outItems []Rule
	for idx, item := range items {
		if idx > 0 {
			outItems = append(outItems, Opt(Whitespace))
		}
		outItems = append(outItems, item)
	}
	return &sequence{
		items: outItems,
	}
}

func OptWhitespaceSurround(r Rule) Rule {
	return Sequence([]Rule{
		Opt(Whitespace),
		r,
		Opt(Whitespace),
	})
}

var Whitespace = &regex{regex: regexp.MustCompile("\\s+")}

var UnsignedIntLit = &regex{regex: regexp.MustCompile("[0-9]+")}

var SignedIntLit = &regex{regex: regexp.MustCompile("-?[0-9]+")}

// Thank you https://stackoverflow.com/a/2039820
var StringLit = &regex{regex: regexp.MustCompile(`\"(\\.|[^"\\])*\"`)}

var Ident = &regex{regex: regexp.MustCompile("[a-zA-Z][a-zA-Z0-9_]*")}
