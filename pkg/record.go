package treesql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"

	"github.com/vilterp/treesql/package/lang"
)

type Record struct {
	Table  *TableDescriptor
	Values []Value
}

type Value struct {
	// tagged union plz?
	Type      ColumnType
	StringVal string
	IntVal    int32
}

func (v *Value) Format() string {
	switch v.Type {
	case TypeInt:
		return fmt.Sprintf("%d", v.IntVal)
	case TypeString:
		return fmt.Sprintf("%#v", v.StringVal)
	default:
		return ""
	}
}

func (table *TableDescriptor) NewRecord() *Record {
	return &Record{
		Table:  table,
		Values: make([]Value, len(table.Columns)),
	}
}

// TODO: delete once all usages removed
func (table *TableDescriptor) RecordFromBytes(raw []byte) *Record {
	record := &Record{
		Table:  table,
		Values: make([]Value, len(table.Columns)),
	}
	buffer := bytes.NewBuffer(raw)
	for valueIdx := 0; valueIdx < len(table.Columns); valueIdx++ {
		typeCode, _ := buffer.ReadByte()
		switch ColumnType(typeCode) {
		case TypeString:
			length, _ := readInteger(buffer)
			stringBytes := make([]byte, length)
			buffer.Read(stringBytes)
			record.Values[valueIdx] = Value{
				Type:      TypeString,
				StringVal: string(stringBytes),
			}
		case TypeInt:
			val, _ := readInteger(buffer)
			record.Values[valueIdx] = Value{
				Type:   TypeInt,
				IntVal: val,
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

func (table *TableDescriptor) recordFromBytes(raw []byte) (*lang.VRecord, error) {
	// TODO: see if there's a way to reduce memory allocation.
	attrs := map[string]lang.Value{}
	buffer := bytes.NewBuffer(raw)
	for _, col := range table.Columns {
		typeCode, _ := buffer.ReadByte()
		if err := assertType(ColumnType(typeCode), col.Type); err != nil {
			return nil, err
		}
		switch col.Type {
		case lang.TString:
			length, _ := readInteger(buffer)
			stringBytes := make([]byte, length)
			buffer.Read(stringBytes)
			attrs[col.Name] = lang.NewVString(string(stringBytes))
		case lang.TInt:
			val, _ := readInteger(buffer)
			attrs[col.Name] = lang.NewVInt(int(val))
		}
	}
	return lang.NewVRecord(attrs), nil
}

func (record *Record) GetField(name string) *Value {
	idx := -1
	for curIdx, column := range record.Table.Columns {
		if column.Name == name {
			idx = curIdx
			break
		}
	}
	if idx == -1 {
		log.Fatalln("field not found for table", record.Table.Name, ":", name)
	}
	return &record.Values[idx]
}

func (record *Record) SetString(name string, value string) {
	idx := record.fieldIndex(name)
	if idx == -1 {
		log.Fatalln("field not found for table", record.Table.Name, ":", name)
	}
	record.Values[idx].Type = TypeString
	record.Values[idx].StringVal = value
}

func (record *Record) SetInt(name string, value int32) {
	idx := record.fieldIndex(name)
	if idx == -1 {
		log.Fatalln("field not found for table", record.Table.Name, ":", name)
	}
	record.Values[idx].Type = TypeString
	record.Values[idx].IntVal = value
}

func (record *Record) fieldIndex(name string) int {
	idx := -1
	for curIdx, column := range record.Table.Columns {
		if column.Name == name {
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

func (record *Record) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	for idx, column := range record.Table.Columns {
		code, ok := codeForType[column.Type]
		if !ok {
			return nil, fmt.Errorf(
				"serialization error: cannot serialize type %s", column.Type.Format(),
			)
		}
		buf.WriteByte(byte(code))
		value := record.Values[idx]
		switch column.Type {
		case lang.TInt:
			WriteInteger(buf, value.IntVal)
		case lang.TString:
			WriteInteger(buf, int32(len(value.StringVal)))
			buf.WriteString(value.StringVal)
		}
	}
	return buf.Bytes(), nil
}

func (record *Record) MarshalJSON() ([]byte, error) {
	out := "{"
	for idx, column := range record.Table.Columns {
		if idx > 0 {
			out += ","
		}
		out += fmt.Sprintf("%s:%s", strconv.Quote(column.Name), strconv.Quote(record.GetField(column.Name).StringVal))
	}
	out += "}"
	return []byte(out), nil
}

func (record *Record) Clone() *Record {
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
