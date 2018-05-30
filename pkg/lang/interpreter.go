package lang

import (
	"fmt"
)

type interpreter struct {
	stackTop *stackFrame
}

type Caller interface {
	Call(VFunction, []Value) (Value, error)
}

// TODO: callLookuper interface
// to pass in places?

func NewInterpreter(rootScope *Scope, expr Expr) *interpreter {
	return &interpreter{
		stackTop: &stackFrame{
			expr:  expr,
			scope: rootScope,
		},
	}
}

func (i *interpreter) Interpret() (Value, error) {
	return i.stackTop.expr.Evaluate(i)
}

func (i *interpreter) pushFrame(frame *stackFrame) {
	frame.parentFrame = i.stackTop
	i.stackTop = frame
}

func (i *interpreter) popFrame() *stackFrame {
	if i.stackTop == nil {
		panic("can't pop frame; at bottom")
	}
	top := i.stackTop
	i.stackTop = top.parentFrame
	return top
}

func (i *interpreter) Call(vFunc VFunction, argVals []Value) (Value, error) {
	// Make new scope.
	newScope := i.stackTop.scope.NewChildScope()
	params := vFunc.getParamList()
	if len(params) != len(argVals) {
		panic("wrong number of args; should have been caught by type checker")
	}
	for idx, argVal := range argVals {
		param := params[idx]
		newScope.Add(param.Name, argVal)
	}
	// Make and push new stack frame.
	newFrame := &stackFrame{
		scope: newScope,
		vFunc: vFunc,
	}
	i.pushFrame(newFrame)
	// Call the lambda or builtin.
	var val Value
	var err error
	switch tVFunc := vFunc.(type) {
	case *vLambda:
		newFrame.expr = tVFunc.def.body
		val, err = i.Interpret()
		return val, err
	case *VBuiltin:
		val, err = tVFunc.Impl(i, argVals)
		if err != nil {
			return nil, err
		}
		if matches, _ := tVFunc.RetType.matches(val.GetType()); !matches {
			return nil, fmt.Errorf(
				"builtin %s supposed to return %s; returned %s",
				tVFunc.Name, tVFunc.RetType.Format(), val.GetType().Format(),
			)
		}
	}
	// Pop and return.
	i.popFrame()
	return val, err
}

type stackFrame struct {
	// if parentFrame is null, this is the root frame.
	parentFrame *stackFrame
	expr        Expr
	scope       *Scope

	// if it's a function stack frame
	vFunc VFunction
}

// TODO: stack frame and stuff
// keep the func Name in there
// also keep a query path of some kind in there,
// so we can go back up the stack and install live query
// listeners.
