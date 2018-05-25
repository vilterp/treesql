package lang

import (
	"fmt"
	"sort"

	pp "github.com/vilterp/treesql/pkg/prettyprint"
)

type Expr interface {
	Evaluate(*interpreter) (Value, error)
	GetType(*TypeScope) (Type, error)
	Inline(*Scope) (Expr, error)
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

func (e *EIntLit) Inline(_ *Scope) (Expr, error) {
	return e, nil
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

func (e *EStringLit) Inline(_ *Scope) (Expr, error) {
	return e, nil
}

// Var

type EVar struct {
	name string
}

var _ Expr = &EVar{}

func NewEVar(name string) *EVar {
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

func (e *EVar) Inline(scope *Scope) (Expr, error) {
	value, err := scope.find(e.name)
	if err != nil {
		return e, nil
	}
	return newEInlinedValue(value), nil
	//return e, nil
}

// Record

type ERecordLit struct {
	exprs map[string]Expr
}

var _ Expr = &ERecordLit{}

func NewERecord(exprs map[string]Expr) *ERecordLit {
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
		pp.Indent(2, pp.Join(kvDocs, pp.CommaNewline)),
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

	return NewTRecord(types), nil
}

func (rl *ERecordLit) Inline(scope *Scope) (Expr, error) {
	inlinedExprs := map[string]Expr{}
	for name, expr := range rl.exprs {
		inlinedExpr, err := expr.Inline(scope)
		if err != nil {
			return nil, err
		}
		inlinedExprs[name] = inlinedExpr
	}
	return NewERecord(inlinedExprs), nil
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

func NewELambda(params paramList, retType Type, body Expr) *ELambda {
	return &ELambda{
		params:  params,
		body:    body,
		retType: retType,
	}
}

func (l *ELambda) Evaluate(interp *interpreter) (Value, error) {
	parentTypeScope := interp.stackTop.scope.GetTypeScope()
	newTypeScope := l.params.createTypeScope(parentTypeScope)
	typ, err := l.body.GetType(newTypeScope)
	if err != nil {
		return nil, err
	}

	inlinedExpr, err := l.Inline(interp.stackTop.scope)
	if err != nil {
		return nil, err
	}
	inlinedLambda := inlinedExpr.(*ELambda)

	return &vLambda{
		def: inlinedLambda,
		// TODO: don't close over the scope if we don't need anything from there
		definedInScope: interp.stackTop.scope,
		typ:            typ,
	}, nil
}

func (l *ELambda) Format() pp.Doc {
	return pp.Seq([]pp.Doc{
		pp.Textf("(%s) => ", l.params.Format()),
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

func (l *ELambda) Inline(scope *Scope) (Expr, error) {
	inlinedExpr, err := l.body.Inline(scope)
	if err != nil {
		return nil, err
	}
	return NewELambda(
		l.params,
		l.retType,
		inlinedExpr,
	), nil
}

// Func Call

type EFuncCall struct {
	funcName string
	args     []Expr

	// when Inline() is called, this is set.
	// BenchmarkSelect indicates that not looking up the function in the scope
	// every time gives an ~8x speedup.
	preBoundFunc Value
}

var _ Expr = &EFuncCall{}

// TODO: remove all these constructors once a parser exists
func NewEFuncCall(name string, args []Expr) *EFuncCall {
	return &EFuncCall{
		funcName: name,
		args:     args,
	}
}

func (fc *EFuncCall) getFuncVal(interp *interpreter) (Value, error) {
	if fc.preBoundFunc != nil {
		return fc.preBoundFunc, nil
	}

	funcVal, err := interp.stackTop.scope.find(fc.funcName)
	if err != nil {
		return nil, err
	}
	fc.preBoundFunc = funcVal

	return funcVal, nil
}

func (fc *EFuncCall) Evaluate(interp *interpreter) (Value, error) {
	funcVal, err := fc.getFuncVal(interp)
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
		if err := bindings.extend(argBindings); err != nil {
			return nil, fmt.Errorf("in argument %d to %s: %s", idx, fc.funcName, err)
		}

	}
	subsType, _, err := tFunc.retType.substitute(bindings)
	if err != nil {
		return nil, err
	}
	return subsType, err
}

func (fc *EFuncCall) Inline(scope *Scope) (Expr, error) {
	funcVal, err := scope.find(fc.funcName)
	if err != nil {
		return fc, nil
	}
	// TODO: not mutate
	fc.preBoundFunc = funcVal

	return fc, nil
}

// Member Access

type EMemberAccess struct {
	record Expr
	member string
}

var _ Expr = &EMemberAccess{}

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
		return nil, fmt.Errorf(
			"member access on a non-record: %s value: %s", ma.Format(), recVal.Format(),
		)
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
			return nil, fmt.Errorf("for expr `%s`: record type `%s` does not have member `%s`", ma.Format(), recTyp.Format(), ma.member)
		}
		return typ, nil
	default:
		return nil, fmt.Errorf("member access on a non-record type: %s %T %s", ma.Format(), recTyp, scope.Format())
	}
}

func (ma *EMemberAccess) Inline(scope *Scope) (Expr, error) {
	inlinedExpr, err := ma.record.Inline(scope)
	if err != nil {
		return nil, err
	}

	switch inlinedExprT := inlinedExpr.(type) {
	case *eInlinedValue:
		value := inlinedExprT.value
		switch inlinedValue := value.(type) {
		case *VRecord:
			return newEInlinedValue(inlinedValue.vals[ma.member]), nil
		default:
			return nil, fmt.Errorf("member access on non-record: %s", ma.member)
		}

	default:
		return NewMemberAccess(inlinedExpr, ma.member), nil
	}
}

// Do block

type DoBinding struct {
	Name string
	Expr Expr
}

type EDoBlock struct {
	doBindings []DoBinding
	lastExpr   Expr
}

var _ Expr = &EDoBlock{}

func NewEDoBlock(bindings []DoBinding, lastExpr Expr) *EDoBlock {
	return &EDoBlock{
		doBindings: bindings,
		lastExpr:   lastExpr,
	}
}

func (db *EDoBlock) Evaluate(interp *interpreter) (Value, error) {
	scope := NewScope(interp.stackTop.scope)
	interp.pushFrame(&stackFrame{
		expr:  db,
		scope: scope,
	})
	for _, binding := range db.doBindings {
		val, err := binding.Expr.Evaluate(interp)
		if err != nil {
			return nil, err
		}
		scope.Add(binding.Name, val)
	}
	res, err := db.lastExpr.Evaluate(interp)
	interp.popFrame()
	return res, err
}

func (db *EDoBlock) Format() pp.Doc {
	docs := make([]pp.Doc, len(db.doBindings)+1)
	for idx, binding := range db.doBindings {
		var doc pp.Doc
		if binding.Name == "" {
			doc = binding.Expr.Format()
		} else {
			doc = pp.Seq([]pp.Doc{
				pp.Text(binding.Name),
				pp.Text(" = "),
				binding.Expr.Format(),
			})
		}
		docs[idx] = doc
	}
	docs[len(db.doBindings)] = db.lastExpr.Format()

	// TODO: maybe add `in` between bindings and expression
	return pp.Seq([]pp.Doc{
		pp.Text("do {"),
		pp.Newline,
		pp.Indent(2, pp.Join(docs, pp.Newline)),
		pp.Newline,
		pp.Text("}"),
	})
}

func (db *EDoBlock) GetType(scope *TypeScope) (Type, error) {
	ts := NewTypeScope(scope)
	for _, binding := range db.doBindings {
		typ, err := binding.Expr.GetType(ts)
		if err != nil {
			return nil, err
		}
		ts.Add(binding.Name, typ)
	}
	return db.lastExpr.GetType(ts)
}

func (db *EDoBlock) Inline(scope *Scope) (Expr, error) {
	//inlinedBindings := make([]DoBinding, len(db.doBindings))
	//for idx, binding := range db.doBindings {
	//	inlinedExpr, err := binding.Expr.Inline(scope)
	//	if err != nil {
	//		return nil, err
	//	}
	//	inlinedBindings[idx] = DoBinding{
	//		Name: binding.Name,
	//		Expr: inlinedExpr,
	//	}
	//}
	//inlinedLastExpr, err := db.lastExpr.Inline(scope)
	//if err != nil {
	//	return nil, err
	//}
	//return NewEDoBlock(inlinedBindings, inlinedLastExpr), nil
	return db, nil
}

type eInlinedValue struct {
	value Value
}

var _ Expr = &eInlinedValue{}

func newEInlinedValue(value Value) *eInlinedValue {
	return &eInlinedValue{
		value: value,
	}
}

func (iv *eInlinedValue) Evaluate(interp *interpreter) (Value, error) {
	return iv.value, nil
}

func (iv *eInlinedValue) Format() pp.Doc {
	return pp.Text("<inlined value>")
}

func (iv *eInlinedValue) GetType(scope *TypeScope) (Type, error) {
	return iv.value.GetType(), nil
}

func (iv *eInlinedValue) Inline(_ *Scope) (Expr, error) {
	return iv, nil
}

type EIndexRef struct {
	table string
	col   string

	index *VIndex
}

var _ Expr = &EIndexRef{}

func NewEIndexRef(table string, col string, index *VIndex) *EIndexRef {
	return &EIndexRef{
		index: index,
		table: table,
		col:   col,
	}
}

func (ir *EIndexRef) Evaluate(interp *interpreter) (Value, error) {
	return ir.index, nil
}

func (ir *EIndexRef) Format() pp.Doc {
	return pp.Textf("<index %s.%s>", ir.table, ir.col)
}

func (ir *EIndexRef) GetType(scope *TypeScope) (Type, error) {
	return ir.index.GetType(), nil
}

func (ir *EIndexRef) Inline(_ *Scope) (Expr, error) {
	return ir, nil
}

// TODO: if
// TODO: case (ayyyy)
