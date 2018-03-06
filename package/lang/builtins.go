package lang

var BuiltinsScope *Scope

func init() {
	BuiltinsScope = NewScope(nil)
	BuiltinsScope.Add("plus", &VBuiltin{
		Name:    "plus",
		RetType: TInt,
		Params:  []Param{{"a", TInt}, {"b", TInt}},
		Impl: func(_ *interpreter, args []Value) (Value, error) {
			l := MustBeVInt(args[0])
			r := MustBeVInt(args[1])
			return NewVInt(l + r), nil
		},
	})
}

// TODO:
// arithmetic
// object member access
// maybe object subset and object update
