package lang

import (
	"fmt"

	pp "github.com/vilterp/treesql/package/pretty_print"
)

// Value Scope

type Scope struct {
	parent *Scope
	vals   map[string]Value
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		vals:   map[string]Value{},
		parent: parent,
	}
}

func (s *Scope) find(name string) (Value, error) {
	val, ok := s.vals[name]
	if !ok {
		if s.parent != nil {
			return s.parent.find(name)
		}
		return nil, fmt.Errorf("not in scope: %s", name)
	}
	return val, nil
}

func (s *Scope) Add(name string, value Value) {
	s.vals[name] = value
}

func (s *Scope) ToTypeScope() *TypeScope {
	var parentScope *TypeScope
	if s.parent != nil {
		parentScope = s.parent.ToTypeScope()
	}
	ts := NewTypeScope(parentScope)
	for name, val := range s.vals {
		typ := val.GetType()
		ts.add(name, typ)
	}
	return ts
}

// Type Scope

type TypeScope struct {
	parent *TypeScope
	types  map[string]Type
}

func NewTypeScope(parent *TypeScope) *TypeScope {
	return &TypeScope{
		parent: parent,
		types:  make(map[string]Type),
	}
}

func (ts *TypeScope) add(name string, typ Type) {
	ts.types[name] = typ
}

func (ts *TypeScope) find(name string) (Type, error) {
	val, ok := ts.types[name]
	if !ok {
		if ts.parent != nil {
			return ts.parent.find(name)
		}
		return nil, fmt.Errorf("not in type scope: %s", name)
	}
	return val, nil
}

// Param List

// (maybe there is a better place for this)

type ParamList []Param

func (pl ParamList) Format() pp.Doc {
	paramDocs := make([]pp.Doc, len(pl))
	for idx, param := range pl {
		paramDocs[idx] = pp.Concat([]pp.Doc{
			pp.Text(param.Name),
			pp.Text(" "),
			param.Typ.Format(),
		})
	}
	return pp.Join(paramDocs, pp.Text(", "))
}

func (pl ParamList) Matches(other ParamList) (bool, TypeVarBindings) {
	if len(pl) != len(other) {
		return false, nil
	}
	bindings := make(TypeVarBindings)
	for idx, param := range pl {
		otherParam := other[idx]
		matches, paramBindings := param.Typ.matches(otherParam.Typ)
		if !matches {
			return false, nil
		}
		bindings.extend(paramBindings)
	}
	return true, bindings
}

// substitute returns new param list, isConcrete, and an error.
func (pl ParamList) substitute(tvb TypeVarBindings) (ParamList, bool, error) {
	out := make(ParamList, len(pl))
	isConcrete := true
	for idx, param := range pl {
		newTyp, concrete, err := param.Typ.substitute(tvb)
		if err != nil {
			return nil, false, err
		}
		out[idx] = Param{
			Typ:  newTyp,
			Name: param.Name,
		}
		isConcrete = isConcrete && concrete
	}
	return out, isConcrete, nil
}

func (pl ParamList) createTypeScope(parentScope *TypeScope) *TypeScope {
	newTS := NewTypeScope(parentScope)
	for _, param := range pl {
		newTS.add(param.Name, param.Typ)
	}
	return newTS
}
