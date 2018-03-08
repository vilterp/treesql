package lang

import (
	"fmt"
	"sort"

	pp "github.com/vilterp/treesql/package/pretty_print"
)

type Expr interface {
	Evaluate(*interpreter) (Value, error)
	GetType(*Scope) (Type, error)
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

func (e *EIntLit) GetType(_ *Scope) (Type, error) {
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

func (e *EStringLit) GetType(_ *Scope) (Type, error) {
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

func (e *EVar) GetType(scope *Scope) (Type, error) {
	val, err := scope.find(e.name)
	if err != nil {
		return nil, err
	}
	return val.GetType(), nil
}

// Object

type EObjectLit struct {
	exprs map[string]Expr
}

var _ Expr = &EObjectLit{}

func (ol *EObjectLit) Evaluate(interp *interpreter) (Value, error) {
	// TODO: push an object path frame
	vals := map[string]Value{}

	for name, expr := range ol.exprs {
		val, err := expr.Evaluate(interp)
		if err != nil {
			return nil, err
		}
		vals[name] = val
	}

	return &VObject{
		vals: vals,
	}, nil
}

func (ol *EObjectLit) Format() pp.Doc {
	// Sort keys
	keys := make([]string, len(ol.exprs))
	idx := 0
	for k := range ol.exprs {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)

	kvDocs := make([]pp.Doc, len(ol.exprs))
	for idx, key := range keys {
		kvDocs[idx] = pp.Concat([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			ol.exprs[key].Format(),
		})
	}

	return pp.Concat([]pp.Doc{
		pp.Text("("), pp.Newline,
		pp.Nest(2, pp.Join(kvDocs, pp.CommaNewline)),
		pp.Newline,
		pp.Text("}"),
	})
}

func (ol *EObjectLit) GetType(scope *Scope) (Type, error) {
	types := map[string]Type{}

	for name, expr := range ol.exprs {
		typ, err := expr.GetType(scope)
		if err != nil {
			return nil, err
		}
		types[name] = typ
	}

	return &TObject{
		Types: types,
	}, nil
}

// Lambda

type Param struct {
	Name string
	Typ  Type
}

type ELambda struct {
	params  ParamList
	body    Expr
	retType Type
}

var _ Expr = &ELambda{}

func (l *ELambda) Evaluate(interp *interpreter) (Value, error) {
	return &vLambda{
		def: l,
		// TODO: don'out close over the scope if we don'out need anything from there
		definedInScope: interp.stackTop.scope,
	}, nil
}

func (l *ELambda) Format() pp.Doc {
	// TODO: use concat instead of sprintf here to facilitate line breaking
	// when it has that
	return pp.Text(
		fmt.Sprintf(
			"(%s): %s => (%s)",
			l.params.Format().Render(), l.retType.Format().Render(), l.body.Format().Render(),
		),
	)
}

func (l *ELambda) GetType(_ *Scope) (Type, error) {
	return &tFunction{
		params:  l.params,
		retType: l.retType,
	}, nil
}

func NewELambda(params ParamList, body Expr, retType Type) *ELambda {
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

	return pp.Concat([]pp.Doc{
		pp.Text(fc.funcName),
		pp.Text("("),
		pp.Join(argDocs, pp.Text(", ")),
		pp.Text(")"),
	})
}

func (fc *EFuncCall) GetType(scope *Scope) (Type, error) {
	funcVal, err := scope.find(fc.funcName)
	if err != nil {
		return nil, err
	}
	switch tFuncVal := funcVal.(type) {
	case vFunction:
		// Check # args matches.
		if len(fc.args) != len(tFuncVal.GetParamList()) {
			return nil, fmt.Errorf(
				"%s: expected %d args; given %d",
				fc.funcName, len(tFuncVal.GetParamList()), len(fc.args),
			)
		}
		// Check arg types match.
		params := tFuncVal.GetParamList()
		bindings := make(TypeVarBindings)
		for idx, argExpr := range fc.args {
			param := params[idx]
			argType, err := argExpr.GetType(scope)
			if err != nil {
				return nil, err
			}
			matches, argBindings := param.Typ.Matches(argType)
			if !matches {
				// TODO: add this check back in
				return nil, fmt.Errorf(
					"call to %s, param %d: have %s; want %s",
					fc.funcName, idx, argType.Format().Render(), param.Typ.Format().Render(),
				)
			}
			bindings.extend(argBindings)
		}
		// Check that body matches declared type.
		// this should really be a TypeScope, not a scope.
		// will need to construct a new scope here.
		return tFuncVal.GetRetType().substitute(bindings), nil
	default:
		return nil, fmt.Errorf("not a function: %s", fc.funcName)
	}
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
	objVal, err := ma.record.Evaluate(interp)
	if err != nil {
		return nil, err
	}
	switch tRecordVal := objVal.(type) {
	case *VObject:
		val, ok := tRecordVal.vals[ma.member]
		if !ok {
			return nil, fmt.Errorf("nonexistent member: %s", ma.member)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("member access on a non-object: %s", ma.Format().Render())
	}
}

func (ma *EMemberAccess) Format() pp.Doc {
	return pp.Concat([]pp.Doc{ma.record.Format(), pp.Text("."), pp.Text(ma.member)})
}

func (ma *EMemberAccess) GetType(scope *Scope) (Type, error) {
	objTyp, err := ma.record.GetType(scope)
	if err != nil {
		return nil, err
	}
	switch tTyp := objTyp.(type) {
	case *TObject:
		typ, ok := tTyp.Types[ma.member]
		if !ok {
			return nil, fmt.Errorf("nonexistent member: %s", ma.member)
		}
		return typ, nil
	default:
		return nil, fmt.Errorf("member access on a non-object: %s", ma.Format().Render())
	}
}

// TODO: Let binding
// TODO: if
// TODO: case (ayyyy)
