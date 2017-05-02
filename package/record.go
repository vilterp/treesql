package treesql

import (
	"bytes"
	"encoding/binary"
	"log"
)

type Record struct {
	Table  *Table
	Values []Value
}

type Value struct {
	// tagged union plz?
	Type      ColumnType
	StringVal string
	IntVal    int
}

func (table *Table) NewRecord() *Record {
	return &Record{
		Table:  table,
		Values: make([]Value, len(table.Columns)),
	}
}

func (table *Table) RecordFromBytes(raw []byte) *Record {
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

func (record *Record) SetInt(name string, value int) {
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
	buffer := NewByteBuffer()
	for idx, column := range record.Table.Columns {
		buffer.Write([]byte{byte(column.Type)})
		value := record.Values[idx]
		switch column.Type {
		case TypeInt:
			writeInteger(buffer, value.IntVal)
		case TypeString:
			writeInteger(buffer, len(value.StringVal))
			buffer.WriteString(value.StringVal)
		}
	}
	result := buffer.GetBytes()
	return result
}

// these are only uints
func readInteger(buffer *bytes.Buffer) (int, error) {
	bytes := make([]byte, 4)
	buffer.Read(bytes)
	result := binary.BigEndian.Uint32(bytes)
	if result > 100000 {
		log.Println("wut")
		panic(int(result))
	}
	return int(result), nil
}

func writeInteger(buffer *ByteBuffer, val int) {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, uint32(val))
	buffer.Write(bytes)
}
