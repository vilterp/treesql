package lang

import (
	"bufio"
	"fmt"
	"sort"

	pp "github.com/vilterp/treesql/pkg/pretty_print"
)

type Value interface {
	Format() pp.Doc
	GetType() Type
	WriteAsJSON(*bufio.Writer, Caller) error
}

// TODO: bool

// Int

type VInt int

var _ Value = NewVInt(0)

func NewVInt(v int) *VInt {
	val := VInt(v)
	return &val
}

func (v *VInt) Format() pp.Doc {
	return pp.Textf("%d", *v)
}

func (v *VInt) GetType() Type {
	return TInt
}

func (v *VInt) WriteAsJSON(w *bufio.Writer, _ Caller) error {
	_, err := w.WriteString(v.Format().String())
	return err
}

func mustBeVInt(v Value) *VInt {
	i, ok := v.(*VInt)
	if !ok {
		panic(fmt.Sprintf("not an int: %s", v.Format()))
	}
	return i
}

// String

type VString string

var _ Value = NewVString("")

func NewVString(s string) *VString {
	val := VString(s)
	return &val
}

func (v *VString) Format() pp.Doc {
	// TODO: test escaping
	return pp.Textf(`%#v`, string(*v))
}

func (v *VString) GetType() Type {
	return TString
}

func (v *VString) WriteAsJSON(w *bufio.Writer, _ Caller) error {
	_, err := w.WriteString(fmt.Sprintf("%#v", *v))
	return err
}

func mustBeVString(v Value) string {
	s, ok := v.(*VString)
	if !ok {
		panic(fmt.Sprintf("not a string: %s", v.Format()))
	}
	return string(*s)
}

// Record

type VRecord struct {
	vals map[string]Value
}

var _ Value = &VRecord{}

func NewVRecord(vals map[string]Value) *VRecord {
	return &VRecord{
		vals: vals,
	}
}

func (v *VRecord) GetType() Type {
	types := map[string]Type{}
	for name, val := range v.vals {
		types[name] = val.GetType()
	}
	return &TRecord{
		Types: types,
	}
}

func (v *VRecord) Format() pp.Doc {
	// Sort keys
	keys := make([]string, len(v.vals))
	idx := 0
	for k := range v.vals {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)

	kvDocs := make([]pp.Doc, len(v.vals))
	for idx, key := range keys {
		kvDocs[idx] = pp.Seq([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			v.vals[key].Format(),
		})
	}

	return pp.Seq([]pp.Doc{
		pp.Text("("), pp.Newline,
		pp.Nest(2, pp.Join(kvDocs, pp.CommaNewline)),
		pp.CommaNewline,
		pp.Text("}"),
	})
}

func (v *VRecord) WriteAsJSON(w *bufio.Writer, c Caller) error {
	w.WriteString("{")
	idx := 0
	for name, val := range v.vals {
		if idx > 0 {
			w.WriteString(",")
		}
		w.WriteString(fmt.Sprintf("%#v:", name))
		val.WriteAsJSON(w, c)
		idx++
	}
	w.WriteString("}")
	return nil
}

// Iterator

// VIteratorRef is a wrapper around an iterator, which
// knows its type.
type VIteratorRef struct {
	iterator Iterator
	ofType   Type
}

var _ Value = &VIteratorRef{}

func NewVIteratorRef(iterator Iterator, ofType Type) *VIteratorRef {
	return &VIteratorRef{
		iterator: iterator,
		ofType:   ofType,
	}
}

func (v *VIteratorRef) GetType() Type {
	return &tIterator{
		innerType: v.ofType,
	}
}

func (v *VIteratorRef) Format() pp.Doc {
	// TODO: some memory address or something to make them distinct?
	return pp.Seq([]pp.Doc{
		pp.Text("<Iterator"),
		v.ofType.Format(),
		pp.Text(">"),
	})
}

func (v *VIteratorRef) WriteAsJSON(w *bufio.Writer, c Caller) error {
	w.WriteString("[")
	idx := 0
	for {
		nextVal, err := v.iterator.Next(c)
		// Check for end of iteration or other error.
		var isEOE bool
		if err != nil {
			switch err.(type) {
			case *endOfIteration:
				isEOE = true
			default:
				return err
			}
		}
		if isEOE {
			break
		}
		// Check type.
		// TODO: maybe define my own equality operator instead of relying on reflect.DeepEqual?
		if matches, _ := nextVal.GetType().matches(v.ofType); !matches {
			return fmt.Errorf(
				"iterator of type %s got next value of wrong type: %s",
				v.ofType.Format(), nextVal.GetType().Format(),
			)
		}
		if idx > 0 {
			w.WriteString(",")
		}
		nextVal.WriteAsJSON(w, c)
		idx++
	}
	w.WriteString("]")
	return nil
}

func mustBeVIteratorRef(v Value) *VIteratorRef {
	ir, ok := v.(*VIteratorRef)
	if !ok {
		panic("not a VIteratorRef")
	}
	return ir
}

// Function

type vFunction interface {
	Value

	GetParamList() ParamList
	GetRetType() Type
}

func mustBeVFunction(v Value) vFunction {
	switch tV := v.(type) {
	case *vLambda:
		return tV
	case *VBuiltin:
		return tV
	default:
		panic("not a vFunction")
	}
}

// Lambda

// aka user-defined function
type vLambda struct {
	def            *ELambda
	definedInScope *Scope
	typ            Type
}

var _ Value = &vLambda{}
var _ vFunction = &vLambda{}

func (vl *vLambda) GetType() Type {
	return vl.typ
}

func (vl *vLambda) Format() pp.Doc {
	return vl.def.Format()
}

func (vl *vLambda) WriteAsJSON(w *bufio.Writer, _ Caller) error {
	return fmt.Errorf("can'out write a lambda to JSON")
}

func (vl *vLambda) GetParamList() ParamList {
	return vl.def.params
}

func (vl *vLambda) GetRetType() Type {
	return vl.def.retType
}

// Builtin

type VBuiltin struct {
	Name    string
	Params  ParamList
	RetType Type

	// TODO: maybe give it a more restricted interface
	Impl func(interp Caller, args []Value) (Value, error)
}

var _ Value = &VBuiltin{}
var _ vFunction = &VBuiltin{}

func (vb *VBuiltin) GetType() Type {
	return &tFunction{
		params:  vb.Params,
		retType: vb.RetType,
	}
}

func (vb *VBuiltin) Format() pp.Doc {
	return pp.Text(fmt.Sprintf(
		`<builtin %s: (%s) => %s>`, vb.Name, vb.Params.Format(), vb.RetType.Format(),
	))
}

func (vb *VBuiltin) WriteAsJSON(w *bufio.Writer, _ Caller) error {
	return fmt.Errorf("can'out write a builtin to JSON")
}

func (vb *VBuiltin) GetParamList() ParamList {
	return vb.Params
}

func (vb *VBuiltin) GetRetType() Type {
	return vb.RetType
}

// TODO: ADT val
