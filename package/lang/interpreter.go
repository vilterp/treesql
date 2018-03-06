package lang

import (
	"fmt"
)

type interpreter struct {
	stackTop *stackFrame
}

func newInterpreter(rootScope *Scope, expr Expr) *interpreter {
	return &interpreter{
		stackTop: &stackFrame{
			expr:  expr,
			scope: rootScope,
		},
	}
}

func (i *interpreter) interpret() (Value, error) {
	panic("unimplemented")
}

// TODO: push/pop frame, etc

type Scope struct {
	parent *Scope
	vals   map[string]Value
}

type notInScopeError struct {
	name string
}

func (e *notInScopeError) Error() string {
	return fmt.Sprintf("not in scope: %s", e.name)
}

func NewScope(vals map[string]Value) *Scope {
	return &Scope{
		vals: vals,
	}
}

func (s *Scope) find(name string) (Value, error) {
	val, ok := s.vals[name]
	if !ok {
		if s.parent != nil {
			return s.parent.find(name)
		}
		return nil, fmt.Errorf("no such var in scope: ")
	}
	return val, nil
}

type stackFrame struct {
	// if parentFrame is null, this is the root frame.
	parentFrame *stackFrame
	expr        Expr
	scope       *Scope

	// TODO: choice of:
	// - funcName
	// - objName
	// - array ind
}

// TODO: stack frame and stuff
// keep the func name in there
// also keep a query path of some kind in there,
// so we can go back up the stack and install live query
// listeners.

func Interpret(e Expr, rootScope *Scope) (Value, error) {
	i := newInterpreter(rootScope, e)
	return i.interpret()
}
