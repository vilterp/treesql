package lang

import (
	"bufio"
	"fmt"
	"sort"

	pp "github.com/vilterp/treesql/pkg/prettyprint"
)

type Value interface {
	Format() pp.Doc
	GetType() Type

	// TODO: implementations of this are swallowing errors all over the place.
	// also, what would we even do if we found an error? stop the stream mid-JSON?
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

// Bool

type VBool bool

var _ Value = NewVBool(false)

func NewVBool(b bool) *VBool {
	val := VBool(b)
	return &val
}

func (v *VBool) Format() pp.Doc {
	if *v {
		return pp.Text("true")
	}
	return pp.Text("false")
}

func (v *VBool) GetType() Type {
	return TBool
}

func (v *VBool) WriteAsJSON(w *bufio.Writer, _ Caller) error {
	_, err := w.WriteString(v.Format().String())
	return err
}

func mustBeVBool(v Value) *VBool {
	b, ok := v.(*VBool)
	if !ok {
		panic(fmt.Sprintf("not a bool: %s", v.Format()))
	}
	return b
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
		types: types,
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
		pp.Text("{"), pp.Newline,
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
		if err := val.WriteAsJSON(w, c); err != nil {
			return err
		}
		idx++
	}
	w.WriteString("}")
	return nil
}

// Array

type VArray struct {
	innerType Type
	values    []Value
}

var _ Value = &VArray{}

func (v *VArray) GetType() Type {
	panic("unimplemented")
}

func (v *VArray) Format() pp.Doc {
	return pp.Seq([]pp.Doc{
		pp.Text("Array<"),
		v.innerType.Format(),
		pp.Text(">"),
	})
}

func (v *VArray) WriteAsJSON(w *bufio.Writer, c Caller) error {
	w.WriteString("[")
	for idx, val := range v.values {
		if idx > 0 {
			w.WriteString(",")
		}
		if err := val.WriteAsJSON(w, c); err != nil {
			return err
		}
	}
	w.WriteString("]")
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
	return NewTIterator(v.ofType)
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
		if err := nextVal.WriteAsJSON(w, c); err != nil {
			return err
		}
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

	GetParamList() paramList
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

func (vl *vLambda) GetParamList() paramList {
	return vl.def.params
}

func (vl *vLambda) GetRetType() Type {
	return vl.def.retType
}

// Builtin

type VBuiltin struct {
	Name    string
	Params  paramList
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

func (vb *VBuiltin) GetParamList() paramList {
	return vb.Params
}

func (vb *VBuiltin) GetRetType() Type {
	return vb.RetType
}

// TODO: ADT val
