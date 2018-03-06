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

// Int

type VInt int

var vZero = VInt(0)
var _ Value = &vZero

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

// String

type VString string

var vEmptyStr = VString("")
var _ Value = &vEmptyStr

func (v *VString) Format() pp.Doc {
	// TODO: test escaping
	return pp.Textf(`%#v`, v)
}

func (v *VString) GetType() Type {
	return TString
}

func (v *VString) WriteAsJSON(w *bufio.Writer) error {
	_, err := w.WriteString(v.Format().Render())
	return err
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
		if err != nil {
			switch err.(type) {
			case *endOfIteration:
				break
			default:
				return err
			}
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
