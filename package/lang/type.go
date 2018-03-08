package lang

import (
	"sort"

	"fmt"

	pp "github.com/vilterp/treesql/package/pretty_print"
)

type Type interface {
	Format() pp.Doc
	typ()
}

func ParseType(name string) (Type, error) {
	switch name {
	case "String":
		return TString, nil
	case "Int":
		return TInt, nil
	default:
		return nil, fmt.Errorf("can't parse type %s", name)
	}
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
	sort.Strings(keys)

	kvDocs := make([]pp.Doc, len(to.Types))
	for idx, key := range keys {
		kvDocs[idx] = pp.Concat([]pp.Doc{
			pp.Text(key),
			pp.Text(": "),
			to.Types[key].Format(),
		})
	}

	return pp.Concat([]pp.Doc{
		pp.Text("{"), pp.Newline,
		pp.Nest(2, pp.Join(kvDocs, pp.CommaNewline)),
		pp.Newline,
		pp.Text("}"),
	})
}

func (TObject) typ() {}

// Iterator

type tIterator struct {
	innerType Type
}

var _ Type = &tIterator{}

func (ti tIterator) Format() pp.Doc {
	return pp.Concat([]pp.Doc{
		pp.Text("Iterator<"),
		ti.innerType.Format(),
		pp.Text(">"),
	})
}

func (tIterator) typ() {}

// Function

type tFunction struct {
	params  ParamList
	retType Type
}

var _ Type = &tFunction{}

func (tf *tFunction) Format() pp.Doc {
	return pp.Concat([]pp.Doc{
		pp.Text("("),
		tf.params.Format(),
		pp.Text(") => "),
		tf.retType.Format(),
	})
}

func (tFunction) typ() {}

// TODO: type vars
// .isConcrete or something

// TODO: ADTs
