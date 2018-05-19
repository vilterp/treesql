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

func EncodeRecord(record *VRecord, typ *TRecord) ([]byte, error) {
	buf := &bytes.Buffer{}

	for _, key := range typ.sortedKeys {
		val := record.vals[key]
		encodable, ok := val.(EncodableValue)
		if !ok {
			return nil, fmt.Errorf("not encodable: %T", val)
		}
		if err := encodable.Encode(buf); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func DecodeRecord(typ *TRecord, theBytes []byte) (*VRecord, error) {
	values := map[string]Value{}

	curBytes := theBytes
	for _, key := range typ.sortedKeys {
		keyTyp := typ.types[key]
		decodableTyp, ok := keyTyp.(DecodableType)
		if !ok {
			return nil, fmt.Errorf("not decodable: %s", decodableTyp.Format())
		}
		rest, val, err := decodableTyp.Decode(curBytes)
		if err != nil {
			return nil, err
		}
		curBytes = rest
		values[key] = val
	}

	return NewVRecord(values), nil
}

func EncodeInteger(val int32) []byte {
	intBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(intBytes, uint32(val))
	return intBytes
}
