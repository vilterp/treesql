package lang

import (
	"fmt"
	"os"
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

type notInScopeError struct {
	name string
}

func (e *notInScopeError) Error() string {
	return fmt.Sprintf("not in scope: %s", e.name)
}

func (s *Scope) find(name string) (Value, error) {
	val, ok := s.vals[name]
	if !ok {
		if s.parent != nil {
			return s.parent.find(name)
		}
		return nil, &notInScopeError{
			name: name,
		}
	}
	return val, nil
}

func (s *Scope) Add(name string, value Value) {
	s.vals[name] = value
}

func (s *Scope) ToTypeScope() *TypeScope {
	fmt.Println("==============")
	var parentScope *TypeScope
	if s.parent != nil {
		parentScope = s.parent.ToTypeScope()
	}
	ts := NewTypeScope(parentScope)
	for name, val := range s.vals {
		fmt.Fprintln(os.Stderr, "getting type for val", val.Format().Render())
		ts.add(name, val.GetType())
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
		return nil, &notInScopeError{
			name: name,
		}
	}
	return val, nil
}
