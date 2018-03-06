package lang

import (
	"bufio"

	"fmt"

	pp "github.com/vilterp/treesql/package/pretty_print"
)

type Value interface {
	Format() pp.Doc
	GetType() Type
	WriteAsJSON(w *bufio.Writer) error
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

func (v *VInt) WriteAsJSON(w *bufio.Writer) error {
	_, err := w.WriteString(v.Format().Render())
	return err
}

func MustBeVInt(v Value) int {
	i, ok := v.(*VInt)
	if !ok {
		panic(fmt.Sprintf("not an int: %s", v.Format().Render()))
	}
	return int(*i)
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

func (v *VString) WriteAsJSON(w *bufio.Writer) error {
	_, err := w.WriteString(v.Format().Render())
	return err
}

func MustBeVString(v Value) string {
	s, ok := v.(*VString)
	if !ok {
		panic(fmt.Sprintf("not a string: %s", v.Format().Render()))
	}
	return string(*s)
}

// Object

type VObject struct {
	vals map[string]Value
}

var _ Value = &VObject{}

func (v *VObject) GetType() Type {
	types := map[string]Type{}
	for name, val := range v.vals {
		types[name] = val.GetType()
	}
	return &TObject{
		Types: types,
	}
}

func (v *VObject) Format() pp.Doc {
	// Sort keys
	keys := make([]string, len(v.vals))
	idx := 0
	for k := range v.vals {
		keys[idx] = k
		idx++
	}

	kvDocs := make([]pp.Doc, len(v.vals))
	for idx, key := range keys {
		kvDocs[idx] = pp.Concat([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			v.vals[key].Format(),
		})
	}

	return pp.Concat([]pp.Doc{
		pp.Text("("), pp.Newline,
		pp.Nest(pp.Concat(kvDocs), 2),
		pp.Text("}"), pp.Newline,
	})
}

func (v *VObject) WriteAsJSON(w *bufio.Writer) error {
	w.WriteString("{")
	idx := 0
	for name, val := range v.vals {
		if idx > 0 {
			w.WriteString(",")
		}
		w.WriteString(fmt.Sprintf("%#v:", name))
		val.WriteAsJSON(w)
		idx++
	}
	w.WriteString("}")
	return nil
}

// Iterator

type VIteratorRef struct {
	iterator Iterator
	ofType   Type
}

var _ Value = &VIteratorRef{}

func (v *VIteratorRef) GetType() Type {
	return &tIterator{
		innerType: v.ofType,
	}
}

func (v *VIteratorRef) Format() pp.Doc {
	// TODO: some memory address or something to make them distinct?
	return pp.Concat([]pp.Doc{pp.Text("<iterator>")})
}

func (v *VIteratorRef) WriteAsJSON(w *bufio.Writer) error {
	w.WriteString("[")
	idx := 0
	for {
		nextVal, err := v.iterator.Next()
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
		// TODO: not sure this actually checks value equality, since
		// these are both pointers.
		if nextVal.GetType() != v.ofType {
			return fmt.Errorf(
				"iterator of type %s got next value %s",
				v.ofType.Format().Render(), nextVal.Format().Render(),
			)
		}
		if idx > 0 {
			w.WriteString(",")
		}
		nextVal.WriteAsJSON(w)
		idx++
	}
	w.WriteString("]")
	return nil
}

// Function

type vFunction interface {
	Value

	GetParamList() ParamList
	GetRetType() Type
}

type ParamList []Param

func (pl ParamList) Format() pp.Doc {
	paramDocs := make([]pp.Doc, len(pl))
	for idx, param := range pl {
		paramDocs[idx] = pp.Concat([]pp.Doc{
			pp.Text(param.Name),
			pp.Text(" "),
			param.Typ.Format(),
		})
	}
	return pp.Join(paramDocs, pp.Text(", "))
}

// Lambda

// aka user-defined function
type vLambda struct {
	def            *ELambda
	definedInScope *Scope
}

var _ Value = &vLambda{}
var _ vFunction = &vLambda{}

func (vl *vLambda) GetType() Type {
	// TODO: this is a bit awkward
	t, err := vl.def.GetType(nil)
	if err != nil {
		panic("panic in lambda get type")
	}
	return t
}

func (vl *vLambda) Format() pp.Doc {
	return vl.def.Format()
}

func (vl *vLambda) WriteAsJSON(w *bufio.Writer) error {
	return fmt.Errorf("can't write a lambda to JSON")
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
	Impl func(interp *interpreter, args []Value) (Value, error)
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
		`<builtin %s(%s): %s>`,
		vb.Name, vb.Params.Format().Render(), vb.RetType.Format().Render(),
	))
}

func (vb *VBuiltin) WriteAsJSON(w *bufio.Writer) error {
	return fmt.Errorf("can't write a builtin to JSON")
}

func (vb *VBuiltin) GetParamList() ParamList {
	return vb.Params
}

func (vb *VBuiltin) GetRetType() Type {
	return vb.RetType
}

// TODO: ADT val
