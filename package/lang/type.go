package lang

import (
	"sort"

	"fmt"

	pp "github.com/vilterp/treesql/package/pretty_print"
)

type Type interface {
	Format() pp.Doc
	Matches(Type) (bool, TypeVarBindings)
	substitute(TypeVarBindings) Type
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

func (tInt) Matches(other Type) (bool, TypeVarBindings) {
	return other == TInt, nil
}

func (ti *tInt) substitute(TypeVarBindings) Type { return ti }

// String

type tString struct{}

var TString = &tString{}
var _ Type = TString

func (tString) Format() pp.Doc {
	return pp.Text("string")
}

func (tString) Matches(other Type) (bool, TypeVarBindings) {
	return other == TString, nil
}

func (ts *tString) substitute(TypeVarBindings) Type { return ts }

// Object

type TObject struct {
	Types map[string]Type
}

var _ Type = &TObject{}

func (to TObject) Format() pp.Doc {
	// Sort keys
	keys := make([]string, len(to.Types))
	idx := 0
	for k := range to.Types {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)

	kvDocs := make([]pp.Doc, len(to.Types))
	for idx, key := range keys {
		kvDocs[idx] = pp.Concat([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			to.Types[key].Format(),
		})
	}

	return pp.Concat([]pp.Doc{
		pp.Text("{"), pp.Newline,
		pp.Nest(2, pp.Join(kvDocs, pp.CommaNewline)),
		pp.Newline,
		pp.Text("}"),
	})
}

func (to *TObject) Matches(other Type) (bool, TypeVarBindings) {
	otherTO, ok := other.(*TObject)
	if !ok {
		return false, nil
	}
	if len(otherTO.Types) != len(to.Types) {
		return false, nil
	}
	for name, typ := range to.Types {
		otherTyp, ok := otherTO.Types[name]
		if !ok {
			return false, nil
		}
		if matches, _ := typ.Matches(otherTyp); !matches {
			return false, nil
		}
	}
	return true, nil
}

func (ts *TObject) substitute(tvb TypeVarBindings) Type {
	types := map[string]Type{}
	for name, typ := range ts.Types {
		types[name] = typ.substitute(tvb)
	}
	return &TObject{Types: types}
}

// Iterator

type tIterator struct {
	innerType Type
}

var _ Type = &tIterator{}

func (ti tIterator) Format() pp.Doc {
	return pp.Concat([]pp.Doc{
		pp.Text("Iterator<"),
		ti.innerType.Format(),
		pp.Text(">"),
	})
}

func (ti tIterator) Matches(other Type) (bool, TypeVarBindings) {
	oti, ok := other.(*tIterator)
	if !ok {
		return false, nil
	}
	return ti.innerType.Matches(oti.innerType)
}

func (ti *tIterator) substitute(tvb TypeVarBindings) Type {
	return &tIterator{
		innerType: ti.innerType.substitute(tvb),
	}
}

// Function

type tFunction struct {
	params  ParamList
	retType Type
}

var _ Type = &tFunction{}

func (tf *tFunction) Format() pp.Doc {
	return pp.Concat([]pp.Doc{
		pp.Text("("),
		tf.params.Format(),
		pp.Text(") => "),
		tf.retType.Format(),
	})
}

func (tf *tFunction) Matches(other Type) (bool, TypeVarBindings) {
	otherFunc, ok := other.(*tFunction)
	if !ok {
		return false, nil
	}
	fmt.Println("matching func", tf.Format().Render(), "with func", other.Format().Render())
	bindings := make(TypeVarBindings)
	// match args
	paramsMatch, paramBindings := tf.params.Matches(otherFunc.params)
	if !paramsMatch {
		return false, nil
	}
	bindings.extend(paramBindings)
	// match ret type
	retMatches, retBindings := tf.retType.Matches(otherFunc.retType)
	if !retMatches {
		return false, nil
	}
	bindings.extend(retBindings)
	fmt.Println("matches with bindings", bindings)
	return true, bindings
}

func (tf *tFunction) substitute(tvb TypeVarBindings) Type {
	return &tFunction{
		params:  tf.params.substitute(tvb),
		retType: tf.retType.substitute(tvb),
	}
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

func (tv *tVar) Matches(other Type) (bool, TypeVarBindings) {
	_, isTVar := other.(*tVar)
	if isTVar {
		return false, nil
	}
	return true, map[tVar]Type{
		*tv: other,
	}
}

func (tv *tVar) substitute(tvb TypeVarBindings) Type {
	fmt.Println("tvb", tvb)
	binding, ok := tvb[*tv]
	if !ok {
		// TODO: return error, don't panic
		panic(fmt.Sprintf("missing type var: %s", *tv))
	}
	return binding
}

// TODO: .isConcrete or something

// TODO: ADTs
