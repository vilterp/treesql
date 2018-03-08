package lang

import "testing"

func TestTypeMatches(t *testing.T) {
	cases := []struct {
		a        Type
		b        Type
		match    bool
		bindings TypeVarBindings
	}{
		{TInt, TInt, true, nil},
		{TInt, TString, false, nil},
		{TString, TString, true, nil},
		{
			&TObject{Types: map[string]Type{"foo": TString, "bar": TInt}},
			&TObject{Types: map[string]Type{"foo": TString, "bar": TInt}},
			true,
			nil,
		},
		// TODO: switching the order breaks them.
		{
			&tIterator{innerType: NewTVar("A")},
			&tIterator{innerType: TInt},
			true,
			map[tVar]Type{tVar("A"): TInt},
		},
		{
			&tFunction{params: []Param{{"a", NewTVar("A")}}, retType: NewTVar("B")},
			&tFunction{params: []Param{{"a", TInt}}, retType: TString},
			true,
			map[tVar]Type{tVar("A"): TInt, tVar("B"): TString},
		},
	}

	for idx, testCase := range cases {
		matches, _ := testCase.a.Matches(testCase.b)
		if matches != testCase.match {
			t.Errorf("case %d: expected %v got %v", idx, testCase.match, matches)
		}
	}
}
