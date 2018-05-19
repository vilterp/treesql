package lang

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// TODO: binary encoding
// take pretty printer and parser off hot path!

func Encode(v Value) ([]byte, error) {
	buf := bytes.Buffer{}

	// TODO get rid of this awkward casting stuff
	encodable, ok := v.(EncodableValue)
	if !ok {
		return nil, fmt.Errorf("not encodable: %T", v)
	}

	if err := encodable.Encode(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func MustEncode(v Value) []byte {
	res, err := Encode(v)
	if err != nil {
		panic(fmt.Sprintf("error encoding: %v", err))
	}
	return res
}

func Decode(typ DecodableType, b []byte) (Value, error) {
	rest, val, err := typ.Decode(b)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		panic(fmt.Errorf("%d extra bytes when decoding %s: %v", len(rest), typ.Format(), rest))
	}

	return val, nil
}

func EncodeInteger(val int32) []byte {
	intBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(intBytes, uint32(val))
	return intBytes
}
