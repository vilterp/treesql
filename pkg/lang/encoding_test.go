package lang

import (
	"reflect"
	"testing"
)

func TestEncoding(t *testing.T) {
	cases := []struct {
		value  Value
		output []byte
	}{
		{
			NewVInt(42),
			[]byte("42"),
		},
		{
			NewVString("foo"),
			[]byte(`"foo"`),
		},
		{
			NewVRecord(map[string]Value{
				"foo": NewVInt(42),
			}),
			[]byte(`{
  foo: 42,
}`),
		},
	}

	for idx, testCase := range cases {
		out, err := Encode(testCase.value)

		if err != nil {
			t.Fatalf("case %d: err: %s", idx, err)
		}

		// Test that we produce the right output.
		if !reflect.DeepEqual(out, testCase.output) {
			t.Fatalf(
				`case %d: expected %v ("%v") got %v ("%v")`,
				idx, testCase.output, string(testCase.output), out, string(out),
			)
		}

		// Test that it round trips.
		decoded, err := Decode(out)
		if err != nil {
			t.Fatalf("case %d: error while decoding: %v", idx, err)
		}

		// TODO: equality for vaules... ugh
		if decoded.Format().String() != testCase.value.Format().String() {
			t.Fatalf(
				"case %d: didn't round trip: started with %v; decoded to %v",
				idx, testCase.value.Format(), decoded.Format(),
			)
		}
	}
}
