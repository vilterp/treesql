package lang

import (
	"fmt"
	"sort"

	pp "github.com/vilterp/treesql/pkg/prettyprint"
)

type Type interface {
	Format() pp.Doc
	matches(Type) (bool, typeVarBindings)

	// Returns substituted type, isConcrete, and an error.
	substitute(typeVarBindings) (Type, bool, error)
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

func typeIsConcrete(t Type) bool {
	_, isConcrete, err := t.substitute(make(typeVarBindings))
	if err != nil {
		return false
	}
	return isConcrete
}

type typeVarBindings map[tVar]Type

func (tvb typeVarBindings) extend(other typeVarBindings) {
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

func (tInt) matches(other Type) (bool, typeVarBindings) {
	return other == TInt, nil
}

func (ti *tInt) substitute(typeVarBindings) (Type, bool, error) { return ti, true, nil }

// Bool

type tBool struct{}

var TBool = &tBool{}
var _ Type = TBool

func (tBool) Format() pp.Doc {
	return pp.Text("bool")
}

func (tBool) matches(other Type) (bool, typeVarBindings) {
	return other == TBool, nil
}

func (tb *tBool) substitute(typeVarBindings) (Type, bool, error) { return tb, true, nil }

// String

type tString struct{}

var TString = &tString{}
var _ Type = TString

func (tString) Format() pp.Doc {
	return pp.Text("string")
}

func (tString) matches(other Type) (bool, typeVarBindings) {
	return other == TString, nil
}

func (ts *tString) substitute(typeVarBindings) (Type, bool, error) { return ts, true, nil }

// Record

type TRecord struct {
	types map[string]Type
}

var _ Type = &TRecord{}

func NewRecordType(types map[string]Type) *TRecord {
	return &TRecord{
		types: types,
	}
}

func (tr TRecord) Format() pp.Doc {
	// Sort keys
	keys := make([]string, len(tr.types))
	idx := 0
	for k := range tr.types {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)

	kvDocs := make([]pp.Doc, len(tr.types))
	for idx, key := range keys {
		kvDocs[idx] = pp.Seq([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			tr.types[key].Format(),
		})
	}

	return pp.Seq([]pp.Doc{
		pp.Text("{"), pp.Newline,
		pp.Nest(2, pp.Join(kvDocs, pp.CommaNewline)),
		pp.CommaNewline,
		pp.Text("}"),
	})
}

func (tr *TRecord) matches(other Type) (bool, typeVarBindings) {
	otherTO, ok := other.(*TRecord)
	if !ok {
		return false, nil
	}
	if len(otherTO.types) != len(tr.types) {
		return false, nil
	}
	for name, typ := range tr.types {
		otherTyp, ok := otherTO.types[name]
		if !ok {
			return false, nil
		}
		if matches, _ := typ.matches(otherTyp); !matches {
			return false, nil
		}
	}
	return true, nil
}

func (tr *TRecord) substitute(tvb typeVarBindings) (Type, bool, error) {
	types := map[string]Type{}
	isConcrete := true
	for name, typ := range tr.types {
		newTyp, typConcrete, err := typ.substitute(tvb)
		if err != nil {
			return nil, false, err
		}
		types[name] = newTyp
		isConcrete = isConcrete && typConcrete
	}
	return &TRecord{types: types}, isConcrete, nil
}

// Iterator

type tIterator struct {
	innerType Type
}

var _ Type = &tIterator{}

func NewTIterator(innerType Type) *tIterator {
	return &tIterator{
		innerType: innerType,
	}
}

func (ti tIterator) Format() pp.Doc {
	return pp.Seq([]pp.Doc{
		pp.Text("Iterator<"),
		ti.innerType.Format(),
		pp.Text(">"),
	})
}

func (ti tIterator) matches(other Type) (bool, typeVarBindings) {
	oti, ok := other.(*tIterator)
	if !ok {
		return false, nil
	}
	return ti.innerType.matches(oti.innerType)
}

func (ti *tIterator) substitute(tvb typeVarBindings) (Type, bool, error) {
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
	params  paramList
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

func (tf *tFunction) matches(other Type) (bool, typeVarBindings) {
	otherFunc, ok := other.(*tFunction)
	if !ok {
		return false, nil
	}
	bindings := make(typeVarBindings)
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

func (tf *tFunction) substitute(tvb typeVarBindings) (Type, bool, error) {
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

func (tv *tVar) matches(other Type) (bool, typeVarBindings) {
	_, isTVar := other.(*tVar)
	if isTVar {
		return false, nil
	}
	return true, map[tVar]Type{
		*tv: other,
	}
}

func (tv *tVar) substitute(tvb typeVarBindings) (Type, bool, error) {
	binding, ok := tvb[*tv]
	if !ok {
		return nil, false, fmt.Errorf("missing type var: %s", *tv)
	}
	return binding, false, nil
}

// TODO: ADTs
