package lang

var BuiltinsScope *Scope
var BuiltinsTypeScope *TypeScope

func init() {
	BuiltinsScope = NewScope(nil)
	BuiltinsScope.AddMap(map[string]Value{
		// Arithmetic.
		"plus": &VBuiltin{
			Name:    "plus",
			Params:  []Param{{"a", TInt}, {"b", TInt}},
			RetType: TInt,
			Impl: func(_ Caller, args []Value) (Value, error) {
				l := int(*mustBeVInt(args[0]))
				r := int(*mustBeVInt(args[1]))
				return NewVInt(l + r), nil
			},
		},
		// Iterator functions.
		"map": &VBuiltin{
			Name: "map",
			Params: []Param{
				{"iter", NewTIterator(NewTVar("A"))},
				{"func", &tFunction{
					params:  []Param{{"x", NewTVar("A")}},
					retType: NewTVar("B"),
				}},
			},
			RetType: NewTIterator(NewTVar("B")),
			Impl: func(c Caller, args []Value) (Value, error) {
				f := mustBeVFunction(args[1])
				return &VIteratorRef{
					iterator: &mapIterator{
						innerIterator: mustBeVIteratorRef(args[0]).iterator,
						f:             f,
					},
					ofType: f.getRetType(),
				}, nil
			},
		},
		"filter": &VBuiltin{
			Name: "filter",
			Params: []Param{
				{"iter", NewTIterator(NewTVar("A"))},
				{"func", &tFunction{
					params:  []Param{{"x", NewTVar("A")}},
					retType: TBool,
				}},
			},
			RetType: NewTIterator(NewTVar("A")),
			Impl: func(interp Caller, args []Value) (Value, error) {
				f := mustBeVFunction(args[1])
				return &VIteratorRef{
					iterator: &filterIterator{
						innerIterator: mustBeVIteratorRef(args[0]).iterator,
						f:             f,
					},
					ofType: f.getRetType(),
				}, nil
			},
		},
		// Index functions.
		"scan": &VBuiltin{
			Name:    "scan",
			Params:  []Param{{"index", NewTIndex(NewTVar("A"))}},
			RetType: NewTIterator(NewTVar("A")),
			Impl: func(interp Caller, args []Value) (Value, error) {
				index := mustBeVIndex(args[0])
				scanIter, err := index.getScanIterator(index.colName)
				if err != nil {
					return nil, err
				}
				return NewVIteratorRef(scanIter, index.innerType), nil
			},
		},
		// Comparison functions.
		"strEq": &VBuiltin{
			Name:    "strEq",
			Params:  []Param{{"a", TString}, {"b", TString}},
			RetType: TBool,
			Impl: func(interp Caller, args []Value) (Value, error) {
				left := mustBeVString(args[0])
				right := mustBeVString(args[1])
				return NewVBool(left == right), nil
			},
		},
		"intEq": &VBuiltin{
			Name:    "intEq",
			Params:  []Param{{"a", TInt}, {"b", TInt}},
			RetType: TBool,
			Impl: func(interp Caller, args []Value) (Value, error) {
				left := mustBeVInt(args[0])
				right := mustBeVInt(args[1])
				return NewVBool(left == right), nil
			},
		},
	})

	BuiltinsTypeScope = BuiltinsScope.toTypeScope()
}

// TODO:
// comparision
// arithmetic
// maybe record subset and update
