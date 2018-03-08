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
		matches, _ := testCase.a.matches(testCase.b)
		if matches != testCase.match {
			t.Errorf("case %d: expected %v got %v", idx, testCase.match, matches)
		}
	}
}

func TestTypeIsConcrete(t *testing.T) {
	cases := []struct {
		typ      Type
		concrete bool
	}{
		{TInt, true},
		{TString, true},
		{
			&TObject{Types: map[string]Type{"foo": TString, "bar": TInt}},
			true,
		},
		{
			&tFunction{params: []Param{{"a", TInt}}, retType: TString},
			true,
		},
		{
			&tFunction{params: []Param{{"a", NewTVar("A")}}, retType: NewTVar("B")},
			false,
		},
		{
			&TObject{Types: map[string]Type{"foo": TString, "bar": NewTVar("A")}},
			false,
		},
		{
			NewTVar("A"),
			false,
		},
	}

	for idx, testCase := range cases {
		concrete := TypeIsConcrete(testCase.typ)
		if concrete != testCase.concrete {
			t.Errorf("case %d: expected %v; got %v", idx, testCase.concrete, concrete)
		}
	}
}
