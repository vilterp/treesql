package treesql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
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

func (record *Record) ToBytes() []byte {
	buf := new(bytes.Buffer)
	for idx, column := range record.Table.Columns {
		buf.Write([]byte{byte(column.Type)})
		value := record.Values[idx]
		switch column.Type {
		case TypeInt:
			WriteInteger(buf, value.IntVal)
		case TypeString:
			WriteInteger(buf, int32(len(value.StringVal)))
			buf.WriteString(value.StringVal)
		}
	}
	return buf.Bytes()
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
	return record.Table.RecordFromBytes(record.ToBytes())
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
