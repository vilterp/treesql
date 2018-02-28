package parserlib

func Intercalate(r Rule, sep Rule) Rule {
	return &Choice{
		Choices: []Rule{
			&Sequence{Items: []Rule{sep, r}},
			r,
		},
	}
}
