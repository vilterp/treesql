package lang

import (
	"fmt"
	"sort"

	pp "github.com/vilterp/treesql/package/pretty_print"
)

type Type interface {
	Format() pp.Doc
	matches(Type) (bool, TypeVarBindings)

	// Returns substituted type, isConcrete, and an error.
	substitute(TypeVarBindings) (Type, bool, error)
}

func ParseType(name string) (Type, error) {
	switch name {
	case "string":
		return TString, nil
	case "int":
		return TInt, nil
	default:
		return nil, fmt.Errorf("can't parse type %s", name)
	}
}

func TypeIsConcrete(t Type) bool {
	_, isConcrete, err := t.substitute(make(TypeVarBindings))
	if err != nil {
		return false
	}
	return isConcrete
}

type TypeVarBindings map[tVar]Type

func (tvb TypeVarBindings) extend(other TypeVarBindings) {
	// TODO: error out if overwriting one that doesn't match
	for name, typ := range other {
		tvb[name] = typ
	}
}

// Int

type tInt struct{}

var TInt = &tInt{}
var _ Type = TInt

func (tInt) Format() pp.Doc {
	return pp.Text("int")
}

func (tInt) matches(other Type) (bool, TypeVarBindings) {
	return other == TInt, nil
}

func (ti *tInt) substitute(TypeVarBindings) (Type, bool, error) { return ti, true, nil }

// String

type tString struct{}

var TString = &tString{}
var _ Type = TString

func (tString) Format() pp.Doc {
	return pp.Text("string")
}

func (tString) matches(other Type) (bool, TypeVarBindings) {
	return other == TString, nil
}

func (ts *tString) substitute(TypeVarBindings) (Type, bool, error) { return ts, true, nil }

// Record

type TRecord struct {
	Types map[string]Type
}

var _ Type = &TRecord{}

func (tr TRecord) Format() pp.Doc {
	// Sort keys
	keys := make([]string, len(tr.Types))
	idx := 0
	for k := range tr.Types {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)

	kvDocs := make([]pp.Doc, len(tr.Types))
	for idx, key := range keys {
		kvDocs[idx] = pp.Seq([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			tr.Types[key].Format(),
		})
	}

	return pp.Seq([]pp.Doc{
		pp.Text("{"), pp.Newline,
		pp.Nest(2, pp.Join(kvDocs, pp.CommaNewline)),
		pp.CommaNewline,
		pp.Text("}"),
	})
}

func (tr *TRecord) matches(other Type) (bool, TypeVarBindings) {
	otherTO, ok := other.(*TRecord)
	if !ok {
		return false, nil
	}
	if len(otherTO.Types) != len(tr.Types) {
		return false, nil
	}
	for name, typ := range tr.Types {
		otherTyp, ok := otherTO.Types[name]
		if !ok {
			return false, nil
		}
		if matches, _ := typ.matches(otherTyp); !matches {
			return false, nil
		}
	}
	return true, nil
}

func (tr *TRecord) substitute(tvb TypeVarBindings) (Type, bool, error) {
	types := map[string]Type{}
	isConcrete := true
	for name, typ := range tr.Types {
		newTyp, typConcrete, err := typ.substitute(tvb)
		if err != nil {
			return nil, false, err
		}
		types[name] = newTyp
		isConcrete = isConcrete && typConcrete
	}
	return &TRecord{Types: types}, isConcrete, nil
}

// Iterator

type tIterator struct {
	innerType Type
}

var _ Type = &tIterator{}

func (ti tIterator) Format() pp.Doc {
	return pp.Seq([]pp.Doc{
		pp.Text("Iterator<"),
		ti.innerType.Format(),
		pp.Text(">"),
	})
}

func (ti tIterator) matches(other Type) (bool, TypeVarBindings) {
	oti, ok := other.(*tIterator)
	if !ok {
		return false, nil
	}
	return ti.innerType.matches(oti.innerType)
}

func (ti *tIterator) substitute(tvb TypeVarBindings) (Type, bool, error) {
	innerTyp, innerConcrete, err := ti.innerType.substitute(tvb)
	if err != nil {
		return nil, false, err
	}
	return &tIterator{
		innerType: innerTyp,
	}, innerConcrete, nil
}

// Function

type tFunction struct {
	params  ParamList
	retType Type
}

var _ Type = &tFunction{}

func (tf *tFunction) Format() pp.Doc {
	return pp.Seq([]pp.Doc{
		pp.Text("("),
		tf.params.Format(),
		pp.Text(") => "),
		tf.retType.Format(),
	})
}

func (tf *tFunction) matches(other Type) (bool, TypeVarBindings) {
	otherFunc, ok := other.(*tFunction)
	if !ok {
		return false, nil
	}
	bindings := make(TypeVarBindings)
	// match args
	paramsMatch, paramBindings := tf.params.Matches(otherFunc.params)
	if !paramsMatch {
		return false, nil
	}
	bindings.extend(paramBindings)
	// match ret type
	retMatches, retBindings := tf.retType.matches(otherFunc.retType)
	if !retMatches {
		return false, nil
	}
	bindings.extend(retBindings)
	return true, bindings
}

func (tf *tFunction) substitute(tvb TypeVarBindings) (Type, bool, error) {
	params, paramsConcrete, err := tf.params.substitute(tvb)
	if err != nil {
		return nil, false, err
	}
	ret, retConcrete, err := tf.retType.substitute(tvb)
	if err != nil {
		return nil, false, err
	}
	concrete := retConcrete && paramsConcrete
	return &tFunction{
		params:  params,
		retType: ret,
	}, concrete, nil
}

// Type variables

type tVar string

var _ Type = NewTVar("A")

func NewTVar(name string) *tVar {
	t := tVar(name)
	return &t
}

func (tv *tVar) Format() pp.Doc {
	return pp.Text(string(*tv))
}

func (tv *tVar) matches(other Type) (bool, TypeVarBindings) {
	_, isTVar := other.(*tVar)
	if isTVar {
		return false, nil
	}
	return true, map[tVar]Type{
		*tv: other,
	}
}

func (tv *tVar) substitute(tvb TypeVarBindings) (Type, bool, error) {
	binding, ok := tvb[*tv]
	if !ok {
		return nil, false, fmt.Errorf("missing type var: %s", *tv)
	}
	return binding, false, nil
}

// TODO: ADTs
