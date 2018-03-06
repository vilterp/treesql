package lang

import (
	pp "github.com/vilterp/treesql/package/pretty_print"
)

type Expr interface {
	Evaluate(*Scope) (Value, error)

	GetType(*Scope) (Type, error)

	Format() pp.Doc
}

// Int

type EIntLit int

var eZero = EIntLit(9)
var _ Expr = &eZero

// TODO: can we avoid an allocation here?
func (e *EIntLit) Evaluate(_ *Scope) (Value, error) {
	theInt := VInt(*e)
	return &theInt, nil
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

func (e *EStringLit) Evaluate(_ *Scope) (Value, error) {
	theStr := VString(*e)
	return &theStr, nil
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

func (e *EVar) Evaluate(scope *Scope) (Value, error) {
	return scope.find(e.name)
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

func (ol *EObjectLit) Evaluate(scope *Scope) (Value, error) {
	vals := map[string]Value{}

	for name, expr := range ol.exprs {
		val, err := expr.Evaluate(scope)
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
		pp.Nest(pp.Concat(kvDocs), 2),
		pp.Text("}"), pp.Newline,
	})
}

func (ol *EObjectLit) GetType(scope *Scope) (Type, error) {
	types := map[string]Type{}

	for name, expr := range ol.exprs {
		val, err := expr.Evaluate(scope)
		if err != nil {
			return nil, err
		}
		types[name] = val.GetType()
	}

	return &TObject{
		Types: types,
	}, nil
}
