package lang

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestWriteAsJSON(t *testing.T) {
	cases := []struct {
		val  Value
		json string
		err  string
	}{
		{
			NewVInt(5),
			"5",
			"",
		},
		{
			NewVString("foo"),
			`"foo"`,
			"",
		},
		{
			&VObject{
				vals: map[string]Value{
					"foo": NewVInt(2),
					"bar": NewVString("baz"),
					"quux": &VIteratorRef{
						ofType:   TInt,
						iterator: NewArrayIterator([]Value{NewVInt(2)}),
					},
				},
			},
			`{"bar": "baz","foo": 2,"quux":[2]}`,
			"",
		},
		{
			&VIteratorRef{
				ofType:   TInt,
				iterator: NewArrayIterator([]Value{NewVInt(2), NewVInt(3), NewVInt(4)}),
			},
			"[2,3,4]",
			"",
		},
		{
			&VBuiltin{},
			"",
			"can't write a builtin to JSON",
		},
		{
			&vLambda{},
			"",
			"can't write a lambda to JSON",
		},
	}

	for idx, testCase := range cases {
		buf := bytes.NewBufferString("")
		w := bufio.NewWriter(buf)
		err := testCase.val.WriteAsJSON(w)
		// TODO: really need to factor this error checking thing out
		if testCase.err == "" {
			if err != nil {
				t.Errorf("case %d: expected nil error; got %s", idx, err.Error())
				continue
			}
		} else {
			if err == nil {
				t.Errorf("case %d: expected error %s, got nil", idx, testCase.err)
				continue
			} else if err.Error() != testCase.err {
				t.Errorf("case %d: expected error %s; got %s", idx, testCase.err, err.Error())
				continue
			} else {
				// Errors are a match
				continue
			}
		}
		w.Flush()
		actual := buf.String()
		equal, err := AreEqualJSON(testCase.json, actual)
		if err != nil {
			t.Errorf("case %d: %v", idx, err)
			break
		}
		if !equal {
			t.Errorf("case %d: EXPECTED:\n\n%s\n\nGOT:\n\n%s", idx, testCase.json, actual)
		}
	}
}

func TestGetType(t *testing.T) {
	// TODO
}

// From https://gist.github.com/turtlemonvh/e4f7404e28387fadb8ad275a99596f67
func AreEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}
