package treesql

import (
	"bytes"
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

func (table *tableDescriptor) NewRecord() *record {
	return &record{
		Table:  table,
		Values: make([]value, len(table.columns)),
	}
}

// TODO: delete once all usages removed
func (table *tableDescriptor) RecordFromBytes(raw []byte) *record {
	record := &record{
		Table:  table,
		Values: make([]value, len(table.columns)),
	}
	buffer := bytes.NewBuffer(raw)
	for valueIdx := 0; valueIdx < len(table.columns); valueIdx++ {
		typeCode, _ := buffer.ReadByte()
		switch ColumnType(typeCode) {
		case TypeString:
			length, _ := readInteger(buffer)
			stringBytes := make([]byte, length)
			buffer.Read(stringBytes)
			record.Values[valueIdx] = value{
				typ:       TypeString,
				stringVal: string(stringBytes),
			}
		case TypeInt:
			val, _ := readInteger(buffer)
			record.Values[valueIdx] = value{
				typ:    TypeInt,
				intVal: val,
			}
		}
	}
	return record
}

func assertType(readType ColumnType, expectedType lang.Type) error {
	if codeForType[expectedType] != readType {
		return fmt.Errorf(
			"deserialization error: expected %s; got %s",
			expectedType.Format(), typeForCode[readType],
		)
	}
	return nil
}

func (table *tableDescriptor) recordFromBytes(raw []byte) (*lang.VRecord, error) {
	// TODO: see if there's a way to reduce memory allocation.
	attrs := map[string]lang.Value{}
	buffer := bytes.NewBuffer(raw)
	for _, col := range table.columns {
		typeCode, _ := buffer.ReadByte()
		if err := assertType(ColumnType(typeCode), col.typ); err != nil {
			return nil, err
		}
		switch col.typ {
		case lang.TString:
			length, _ := readInteger(buffer)
			stringBytes := make([]byte, length)
			buffer.Read(stringBytes)
			attrs[col.name] = lang.NewVString(string(stringBytes))
		case lang.TInt:
			val, _ := readInteger(buffer)
			attrs[col.name] = lang.NewVInt(int(val))
		}
	}
	return lang.NewVRecord(attrs), nil
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

func (record *record) setString(name string, value string) {
	idx := record.fieldIndex(name)
	if idx == -1 {
		log.Fatalln("field not found for table", record.Table.name, ":", name)
	}
	record.Values[idx].typ = TypeString
	record.Values[idx].stringVal = value
}

func (record *record) setInt(name string, value int32) {
	idx := record.fieldIndex(name)
	if idx == -1 {
		log.Fatalln("field not found for table", record.Table.name, ":", name)
	}
	record.Values[idx].typ = TypeString
	record.Values[idx].intVal = value
}

func (record *record) fieldIndex(name string) int {
	idx := -1
	for curIdx, column := range record.Table.columns {
		if column.name == name {
			idx = curIdx
			break
		}
	}
	return idx
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

func (record *record) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	for idx, column := range record.Table.columns {
		code, ok := codeForType[column.typ]
		if !ok {
			return nil, fmt.Errorf(
				"serialization error: cannot serialize type %s", column.typ.Format(),
			)
		}
		buf.WriteByte(byte(code))
		value := record.Values[idx]
		switch column.typ {
		case lang.TInt:
			WriteInteger(buf, value.intVal)
		case lang.TString:
			WriteInteger(buf, int32(len(value.stringVal)))
			buf.WriteString(value.stringVal)
		}
	}
	return buf.Bytes(), nil
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

func (record *record) Clone() *record {
	clone, err := record.ToBytes()
	if err != nil {
		panic(fmt.Sprintf("can't serialize record in clone: %v", err))
	}
	return record.Table.RecordFromBytes(clone)
}

// these are only uints
func readInteger(buffer *bytes.Buffer) (int32, error) {
	bytes := make([]byte, 4)
	buffer.Read(bytes)
	result := binary.BigEndian.Uint32(bytes)
	return int32(result), nil
}

func WriteInteger(buf *bytes.Buffer, val int32) {
	buf.Write(encodeInteger(val))
}

// Encodes an integer in 4 bytes.
func encodeInteger(val int32) []byte {
	intBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(intBytes, uint32(val))
	return intBytes
}
