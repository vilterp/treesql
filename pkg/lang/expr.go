package lang

import (
	"fmt"
	"sort"

	pp "github.com/vilterp/treesql/pkg/pretty_print"
)

type Expr interface {
	Evaluate(*interpreter) (Value, error)
	GetType(*TypeScope) (Type, error)
	Format() pp.Doc
}

// Int

type EIntLit int

var _ Expr = NewIntLit(0)

func NewIntLit(i int) *EIntLit {
	val := EIntLit(i)
	return &val
}

// TODO: can we avoid an allocation here?
func (e *EIntLit) Evaluate(_ *interpreter) (Value, error) {
	return NewVInt(int(*e)), nil
}

func (e *EIntLit) Format() pp.Doc {
	return pp.Textf("%d", *e)
}

func (e *EIntLit) GetType(*TypeScope) (Type, error) {
	return TInt, nil
}

// String

type EStringLit string

var eEmptyStr = EStringLit("")
var _ Expr = &eEmptyStr

func NewStringLit(s string) *EStringLit {
	val := EStringLit(s)
	return &val
}

func (e *EStringLit) Evaluate(_ *interpreter) (Value, error) {
	return NewVString(string(*e)), nil
}

func (e *EStringLit) Format() pp.Doc {
	return pp.Textf("%#v", *e)
}

func (e *EStringLit) GetType(*TypeScope) (Type, error) {
	return TString, nil
}

// Var

type EVar struct {
	name string
}

var _ Expr = &EVar{}

func NewVar(name string) *EVar {
	return &EVar{name: name}
}

func (e *EVar) Evaluate(interp *interpreter) (Value, error) {
	return interp.stackTop.scope.find(e.name)
}

func (e *EVar) Format() pp.Doc {
	return pp.Text(e.name)
}

func (e *EVar) GetType(scope *TypeScope) (Type, error) {
	typ, err := scope.find(e.name)
	if err != nil {
		return nil, err
	}
	return typ, nil
}

// Record

type ERecordLit struct {
	exprs map[string]Expr
}

var _ Expr = &ERecordLit{}

func NewRecordLit(exprs map[string]Expr) *ERecordLit {
	return &ERecordLit{
		exprs: exprs,
	}
}

func (rl *ERecordLit) Evaluate(interp *interpreter) (Value, error) {
	// TODO: push an record path frame
	vals := map[string]Value{}

	for name, expr := range rl.exprs {
		val, err := expr.Evaluate(interp)
		if err != nil {
			return nil, err
		}
		vals[name] = val
	}

	return &VRecord{
		vals: vals,
	}, nil
}

func (rl *ERecordLit) Format() pp.Doc {
	// Empty record
	if len(rl.exprs) == 0 {
		return pp.Text("{}")
	}

	// Sort keys
	keys := make([]string, len(rl.exprs))
	idx := 0
	for k := range rl.exprs {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)

	kvDocs := make([]pp.Doc, len(rl.exprs))
	for idx, key := range keys {
		kvDocs[idx] = pp.Seq([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			rl.exprs[key].Format(),
		})
	}

	return pp.Seq([]pp.Doc{
		pp.Text("{"), pp.Newline,
		pp.Nest(2, pp.Join(kvDocs, pp.CommaNewline)),
		pp.Newline,
		pp.Text("}"),
	})
}

func (rl *ERecordLit) GetType(scope *TypeScope) (Type, error) {
	types := map[string]Type{}

	for name, expr := range rl.exprs {
		typ, err := expr.GetType(scope)
		if err != nil {
			return nil, err
		}
		types[name] = typ
	}

	return &TRecord{
		types: types,
	}, nil
}

// Lambda

type Param struct {
	Name string
	Typ  Type
}

type ELambda struct {
	params  paramList
	body    Expr
	retType Type
}

var _ Expr = &ELambda{}

func (l *ELambda) Evaluate(interp *interpreter) (Value, error) {
	parentTypeScope := interp.stackTop.scope.toTypeScope()
	newTypeScope := l.params.createTypeScope(parentTypeScope)
	typ, err := l.body.GetType(newTypeScope)
	if err != nil {
		return nil, err
	}
	return &vLambda{
		def: l,
		// TODO: don't close over the scope if we don't need anything from there
		definedInScope: interp.stackTop.scope,
		typ:            typ,
	}, nil
}

func (l *ELambda) Format() pp.Doc {
	return pp.Seq([]pp.Doc{
		pp.Textf("(%s): %s => ", l.params.Format(), l.retType.Format()),
		l.body.Format(),
	})
}

func (l *ELambda) GetType(s *TypeScope) (Type, error) {
	innerScope := l.params.createTypeScope(s)

	innerTyp, err := l.body.GetType(innerScope)
	if err != nil {
		return nil, err
	}
	if matches, _ := innerTyp.matches(l.retType); !matches {
		return nil, fmt.Errorf(
			"lambda declared as returning %s; body is of type %s",
			l.retType.Format(), innerTyp.Format(),
		)
	}
	return &tFunction{
		params:  l.params,
		retType: l.retType,
	}, nil
}

func NewELambda(params paramList, body Expr, retType Type) *ELambda {
	return &ELambda{
		params:  params,
		body:    body,
		retType: retType,
	}
}

// Func Call

type EFuncCall struct {
	funcName string
	args     []Expr
}

// TODO: remove all these constructors once a parser exists
func NewFuncCall(name string, args []Expr) *EFuncCall {
	return &EFuncCall{
		funcName: name,
		args:     args,
	}
}

func (fc *EFuncCall) Evaluate(interp *interpreter) (Value, error) {
	// Get function value.
	funcVal, err := interp.stackTop.scope.find(fc.funcName)
	if err != nil {
		return nil, err
	}

	// Get argument values.
	argVals := make([]Value, len(fc.args))
	for idx, argExpr := range fc.args {
		argVal, err := argExpr.Evaluate(interp)
		if err != nil {
			return nil, err
		}
		argVals[idx] = argVal
	}

	switch tFuncVal := funcVal.(type) {
	case *vLambda:
		return interp.Call(tFuncVal, argVals)
	case *VBuiltin:
		return interp.Call(tFuncVal, argVals)
	default:
		return nil, fmt.Errorf("not a function: %s", fc.funcName)
	}
}

func (fc *EFuncCall) Format() pp.Doc {
	argDocs := make([]pp.Doc, len(fc.args))
	for idx, arg := range fc.args {
		argDocs[idx] = arg.Format()
	}

	return pp.Seq([]pp.Doc{
		pp.Text(fc.funcName),
		pp.Text("("),
		pp.Join(argDocs, pp.Text(", ")),
		pp.Text(")"),
	})
}

func (fc *EFuncCall) GetType(scope *TypeScope) (Type, error) {
	maybeFunc, err := scope.find(fc.funcName)
	if err != nil {
		return nil, err
	}

	tFunc, ok := maybeFunc.(*tFunction)
	if !ok {
		return nil, fmt.Errorf(
			"expected %s to be a function; it's %v", fc.funcName, tFunc,
		)
	}
	if len(fc.args) != len(tFunc.params) {
		return nil, fmt.Errorf(
			"%s: expected %d args; given %d",
			fc.funcName, len(tFunc.params), len(fc.args),
		)
	}
	// Check arg types match.
	params := tFunc.params
	bindings := make(typeVarBindings)
	for idx, argExpr := range fc.args {
		param := params[idx]
		argType, err := argExpr.GetType(scope)
		if err != nil {
			return nil, err
		}
		matches, argBindings := param.Typ.matches(argType)
		if !matches {
			return nil, fmt.Errorf(
				"call to %s, param %d: have %s; want %s",
				fc.funcName, idx, argType.Format(), param.Typ.Format(),
			)
		}
		bindings.extend(argBindings)
	}
	subsType, _, err := tFunc.retType.substitute(bindings)
	return subsType, err
}

// Member Access

type EMemberAccess struct {
	record Expr
	member string
}

var _ Expr = &EMemberAccess{}

// TODO: idk how I feel about all these constructors
// other packages wouldn't need to construct AST nodes if there
// was a parser for this language.
func NewMemberAccess(record Expr, member string) *EMemberAccess {
	return &EMemberAccess{
		record: record,
		member: member,
	}
}

func (ma *EMemberAccess) Evaluate(interp *interpreter) (Value, error) {
	recVal, err := ma.record.Evaluate(interp)
	if err != nil {
		return nil, err
	}
	switch tRecordVal := recVal.(type) {
	case *VRecord:
		val, ok := tRecordVal.vals[ma.member]
		if !ok {
			return nil, fmt.Errorf("nonexistent member: %s", ma.member)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("member access on a non-record: %s", ma.Format())
	}
}

func (ma *EMemberAccess) Format() pp.Doc {
	return pp.Seq([]pp.Doc{ma.record.Format(), pp.Text("."), pp.Text(ma.member)})
}

func (ma *EMemberAccess) GetType(scope *TypeScope) (Type, error) {
	recTyp, err := ma.record.GetType(scope)
	if err != nil {
		return nil, err
	}
	switch tTyp := recTyp.(type) {
	case *TRecord:
		typ, ok := tTyp.types[ma.member]
		if !ok {
			return nil, fmt.Errorf("nonexistent member: %s", ma.member)
		}
		return typ, nil
	default:
		return nil, fmt.Errorf("member access on a non-record: %s %T", ma.Format(), recTyp)
	}
}

// TODO: Let binding
// TODO: if
// TODO: case (ayyyy)
