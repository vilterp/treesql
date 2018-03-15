package lang

import (
	"fmt"

	pp "github.com/vilterp/treesql/pkg/prettyprint"
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

func (s *Scope) toTypeScope() *TypeScope {
	var parentScope *TypeScope
	if s.parent != nil {
		parentScope = s.parent.toTypeScope()
	}
	ts := NewTypeScope(parentScope)
	for name, val := range s.vals {
		typ := val.GetType()
		ts.Add(name, typ)
	}
	return ts
}

func (s *Scope) Format() pp.Doc {
	docs := make([]pp.Doc, len(s.vals))
	idx := 0
	for name, val := range s.vals {
		docs[idx] = pp.Seq([]pp.Doc{
			pp.Text(name),
			pp.Text(": "),
			val.Format(),
		})
		idx++
	}

	var parentDoc pp.Doc
	if s.parent == nil {
		parentDoc = pp.Text("<nil>")
	} else {
		parentDoc = s.parent.Format()
	}

	return pp.Seq([]pp.Doc{
		pp.Text("Scope{"), pp.Newline,
		pp.Nest(2, pp.Seq([]pp.Doc{
			pp.Text("vals: {"), pp.Newline,
			pp.Nest(2, pp.Seq([]pp.Doc{
				pp.Join(docs, pp.CommaNewline),
			})),
			pp.Newline, pp.Text("},"), pp.Newline,
			pp.Text("parent: "),
			parentDoc,
		})),
		pp.CommaNewline, pp.Text("}"),
	})
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

func (ts *TypeScope) Add(name string, typ Type) {
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

func (ts *TypeScope) Format() pp.Doc {
	// TODO: DRY with Scope
	docs := make([]pp.Doc, len(ts.types))
	idx := 0
	for name, val := range ts.types {
		docs[idx] = pp.Seq([]pp.Doc{
			pp.Text(name),
			pp.Text(": "),
			val.Format(),
		})
		idx++
	}

	var parentDoc pp.Doc
	if ts.parent == nil {
		parentDoc = pp.Text("<nil>")
	} else {
		parentDoc = ts.parent.Format()
	}

	return pp.Seq([]pp.Doc{
		pp.Text("Scope{"), pp.Newline,
		pp.Nest(2, pp.Seq([]pp.Doc{
			pp.Text("vals: {"), pp.Newline,
			pp.Nest(2, pp.Seq([]pp.Doc{
				pp.Join(docs, pp.CommaNewline),
			})),
			pp.Newline, pp.Text("},"), pp.Newline,
			pp.Text("parent: "),
			parentDoc,
		})),
		pp.CommaNewline, pp.Text("}"),
	})
}

// Param List

// (maybe there is a better place for this)

type paramList []Param

func (pl paramList) Format() pp.Doc {
	paramDocs := make([]pp.Doc, len(pl))
	for idx, param := range pl {
		paramDocs[idx] = pp.Text(param.Name)
	}
	return pp.Join(paramDocs, pp.Text(", "))
}

func (pl paramList) Matches(other paramList) (bool, typeVarBindings) {
	if len(pl) != len(other) {
		return false, nil
	}
	bindings := make(typeVarBindings)
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
func (pl paramList) substitute(tvb typeVarBindings) (paramList, bool, error) {
	out := make(paramList, len(pl))
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

func (pl paramList) createTypeScope(parentScope *TypeScope) *TypeScope {
	newTS := NewTypeScope(parentScope)
	for _, param := range pl {
		newTS.Add(param.Name, param.Typ)
	}
	return newTS
}
