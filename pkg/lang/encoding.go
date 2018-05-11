package lang

import (
	"encoding/binary"
	"fmt"
)

// TODO: binary encoding
// take pretty printer and parser off hot path!

func Encode(v Value) ([]byte, error) {
	// TODO get rid of this awkward casting stuff
	encodable, ok := v.(EncodableValue)
	if !ok {
		return nil, fmt.Errorf("not encodable: %T", v)
	}

	return encodable.Encode(), nil
}

func MustEncode(v Value) []byte {
	res, err := Encode(v)
	if err != nil {
		panic(fmt.Sprintf("error encoding: %v", err))
	}
	return res
}

func Decode(b []byte) (Value, error) {
	expr, err := Parse(string(b))
	if err != nil {
		return nil, err
	}
	interp := NewInterpreter(NewScope(nil), expr)
	return interp.Interpret()
}

func EncodeInteger(val int32) []byte {
	intBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(intBytes, uint32(val))
	return intBytes
}
