package treesql

import (
	"encoding/binary"
	"fmt"
	"log"
	"strconv"

	"github.com/vilterp/treesql/pkg/lang"
)

type record struct {
	Table  *tableDescriptor
	Values []value
}

type value struct {
	// tagged union plz?
	typ       ColumnType
	stringVal string
	intVal    int32
}

func (v *value) Format() string {
	switch v.typ {
	case TypeInt:
		return fmt.Sprintf("%d", v.intVal)
	case TypeString:
		return fmt.Sprintf("%#v", v.stringVal)
	default:
		return ""
	}
}

func (record *record) GetField(name string) *value {
	idx := -1
	for curIdx, column := range record.Table.columns {
		if column.name == name {
			idx = curIdx
			break
		}
	}
	if idx == -1 {
		log.Fatalln("field not found for table", record.Table.name, ":", name)
	}
	return &record.Values[idx]
}

// maybe I should use that iota weirdness
type ColumnType byte

const TypeString ColumnType = 0
const TypeInt ColumnType = 1

var codeForType = map[lang.Type]ColumnType{
	lang.TInt:    TypeInt,
	lang.TString: TypeString,
}

var typeForCode = map[ColumnType]lang.Type{}

func init() {
	for typ, code := range codeForType {
		typeForCode[code] = typ
	}
}

func (record *record) MarshalJSON() ([]byte, error) {
	out := "{"
	for idx, column := range record.Table.columns {
		if idx > 0 {
			out += ","
		}
		out += fmt.Sprintf("%s:%s", strconv.Quote(column.name), strconv.Quote(record.GetField(column.name).stringVal))
	}
	out += "}"
	return []byte(out), nil
}

// Encodes an integer in 4 bytes.
func encodeInteger(val int32) []byte {
	intBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(intBytes, uint32(val))
	return intBytes
}
