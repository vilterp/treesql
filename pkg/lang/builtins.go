package lang

var BuiltinsScope *Scope

func init() {
	// Map live function.
	// TODO: would be nice to parse this instead of writing out the AST
	// mapLive : (Index<K, V>, (V) => S) => Iterator<S>
	// mapLive = (index, f) => do {
	//   iterator = scan(index)
	//   addInsertListener(index, f)
	//   map(iterator, f)
	// }
	mapLiveParams := []Param{
		{"index", NewTIndex(NewTVar("K"), NewTVar("V"))},
		{"f", NewTFunction(
			[]Param{{"record", NewTVar("V")}},
			NewTVar("S"),
		)},
	}
	mapLiveRetType := NewTIterator(NewTVar("S"))

	mapLive := &vLambda{
		typ: NewTFunction(
			mapLiveParams,
			mapLiveRetType,
		),
		def: NewELambda(
			mapLiveParams,
			mapLiveRetType,
			NewEDoBlock(
				[]DoBinding{
					{"iterator", NewEFuncCall("scan", []Expr{NewEVar("index")})},
					{"", NewEFuncCall("addInsertListener", []Expr{NewEVar("index"), NewEVar("f")})},
				},
				NewEFuncCall("map", []Expr{NewEVar("iterator"), NewEVar("f")}),
			),
		),
		definedInScope: BuiltinsScope,
	}

	// Build builtins scope.
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
		// Index functions.
		"get": &VBuiltin{
			Name:    "get",
			RetType: NewTVar("V"),
			Params: []Param{
				{
					Name: "index",
					Typ:  NewTIndex(NewTVar("K"), NewTVar("V")),
				},
				{
					Name: "value",
					Typ:  NewTVar("K"),
				},
			},
			Impl: func(interp Caller, args []Value) (Value, error) {
				index := mustBeVIndex(args[0])
				key := args[1]
				return index.getValue(key)
			},
		},
		"scan": &VBuiltin{
			Name:    "scan",
			Params:  []Param{{"index", NewTIndex(NewTVar("K"), NewTVar("V"))}},
			RetType: NewTIterator(NewTVar("V")),
			Impl: func(interp Caller, args []Value) (Value, error) {
				index := mustBeVIndex(args[0])
				scanIter, err := index.getScanIterator()
				if err != nil {
					return nil, err
				}
				return NewVIteratorRef(scanIter, index.valueType), nil
			},
		},
		"mapLive": mapLive,
		// Index listener functions.
		"addInsertListener": &VBuiltin{
			Name:    "addInsertListener",
			RetType: TUnit,
			Params: []Param{
				{
					Name: "index",
					Typ:  NewTIndex(NewTVar("K"), NewTVar("V")),
				},
				{
					Name: "selection",
					Typ: NewTFunction(
						[]Param{
							{
								Name: "row",
								// TODO: not sure this will always be the same as the index type
								Typ: NewTVar("V"),
							},
						},
						NewTVar("S"),
					),
				},
			},
			Impl: func(interp Caller, args []Value) (Value, error) {
				index := mustBeVIndex(args[0])
				f := mustBeVFunction(args[1])
				index.addInsertListener(f)
				return VUnit, nil
			},
		},
		"addUpdateListener": &VBuiltin{
			Name: "addUpdateListener",
			Params: []Param{
				{
					Name: "index",
					Typ:  NewTIndex(NewTVar("K"), NewTVar("V")),
				},
				{
					Name: "pk",
					Typ:  NewTVar("K"),
				},
				{
					Name: "selection",
					Typ: NewTFunction(
						[]Param{
							{
								Name: "row",
								Typ:  NewTVar("ROW"),
							},
						},
						NewTVar("S"),
					),
				},
			},
			RetType: TUnit,
			Impl: func(interp Caller, args []Value) (Value, error) {
				panic("TODO: implement addUpdateListener")
				return VUnit, nil
			},
		},
	})
}

// TODO:
// comparision
// arithmetic
// maybe record subset and update
