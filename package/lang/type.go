package lang

import (
	pp "github.com/vilterp/treesql/package/pretty_print"
)

type Type interface {
	Format() pp.Doc

	typ()
}

// Int

type tInt struct{}

var TInt = &tInt{}
var _ Type = TInt

func (tInt) Format() pp.Doc {
	return pp.Text("Int")
}

func (tInt) typ() {}

// String

type tString struct{}

var TString = &tString{}
var _ Type = TString

func (tString) Format() pp.Doc {
	return pp.Text("String")
}

func (tString) typ() {}

// Object

type TObject struct {
	Types map[string]Type
}

var _ Type = &TObject{}

func (to TObject) Format() pp.Doc {
	// Sort keys
	keys := make([]string, len(to.Types))
	idx := 0
	for k := range to.Types {
		keys[idx] = k
		idx++
	}

	kvDocs := make([]pp.Doc, len(to.Types))
	for idx, key := range keys {
		kvDocs[idx] = pp.Concat([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			to.Types[key].Format(),
		})
	}

	return pp.Concat([]pp.Doc{
		pp.Text("("), pp.Newline,
		pp.Nest(pp.Concat(kvDocs), 2),
		pp.Text("}"), pp.Newline,
	})
}

func (TObject) typ() {}

// Iterator

type tIterator struct {
	innerType Type
}

var _ Type = &tIterator{}

func (ti tIterator) Format() pp.Doc {
	return pp.Textf("Iterator<%s>", ti.innerType.Format().Render())
}

func (tIterator) typ() {}
