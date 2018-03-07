package treesql

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
)

type Schema struct {
	Tables       map[string]*TableDescriptor
	NextColumnID int
}

// TODO: better name, or refactor. not just a descriptor, since
// it also holds live query info.
type TableDescriptor struct {
	Name          string
	Columns       []*ColumnDescriptor
	PrimaryKey    string
	LiveQueryInfo *LiveQueryInfo
	IsBuiltin     bool
}

func (table *TableDescriptor) colIDForName(name string) (int, error) {
	for _, col := range table.Columns {
		if col.Name == name {
			return col.ID, nil
		}
	}
	return 0, fmt.Errorf("col not found: %s", name)
}

type ColumnName string
type ColumnDescriptor struct {
	ID               int
	Name             string
	Type             ColumnType
	ReferencesColumn *ColumnReference
}

type ColumnReference struct {
	TableName string // we're gonna assume for now that you can only reference the primary key
}

// maybe I should use that iota weirdness
type ColumnType byte

const TypeString ColumnType = 0
const TypeInt ColumnType = 1

var TypeToName = map[ColumnType]string{
	TypeString: "string",
	TypeInt:    "int",
}

var NameToType = map[string]ColumnType{
	"string": TypeString,
	"int":    TypeInt,
}

func (column *ColumnDescriptor) ToRecord(tableName string, db *Database) *Record {
	columnsTable := db.Schema.Tables["__columns__"]
	record := columnsTable.NewRecord()
	record.SetString("id", fmt.Sprintf("%d", column.ID))
	record.SetString("name", column.Name)
	record.SetString("table_name", tableName)
	record.SetString("type", TypeToName[column.Type])
	if column.ReferencesColumn != nil {
		record.SetString("references", column.ReferencesColumn.TableName)
	}
	return record
}

func ColumnFromRecord(record *Record) *ColumnDescriptor {
	idInt, _ := strconv.Atoi(record.GetField("id").StringVal)
	references := record.GetField("references").StringVal
	var columnReference *ColumnReference
	if len(references) > 0 { // should things be nullable? idk
		columnReference = &ColumnReference{
			TableName: references,
		}
	}
	return &ColumnDescriptor{
		ID:               idInt,
		Name:             record.GetField("name").StringVal,
		Type:             NameToType[record.GetField("type").StringVal],
		ReferencesColumn: columnReference,
	}
}

func (table *TableDescriptor) ToRecord(db *Database) *Record {
	record := db.Schema.Tables["__tables__"].NewRecord()
	record.SetString("name", table.Name)
	record.SetString("primary_key", table.PrimaryKey)
	return record
}

func TableFromRecord(record *Record) *TableDescriptor {
	return &TableDescriptor{
		Columns:    make([]*ColumnDescriptor, 0),
		Name:       record.GetField("name").StringVal,
		PrimaryKey: record.GetField("primary_key").StringVal,
	}
}

func (db *Database) EnsureBuiltinSchema() {
	db.BoltDB.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("__tables__"))
		tx.CreateBucketIfNotExists([]byte("__columns__"))
		sequencesBucket, _ := tx.CreateBucketIfNotExists([]byte("__sequences__"))
		// sync next column id
		nextColumnIDBytes := sequencesBucket.Get([]byte("__next_column_id__"))
		if nextColumnIDBytes == nil {
			// write it
			nextColumnIDBytes = make([]byte, 4)
			binary.BigEndian.PutUint32(nextColumnIDBytes, uint32(db.Schema.NextColumnID))
			sequencesBucket.Put([]byte("__next_column_id__"), nextColumnIDBytes)
		} else {
			// read it
			nextColumnID := binary.BigEndian.Uint32(nextColumnIDBytes)
			db.Schema.NextColumnID = int(nextColumnID)
		}
		return nil
	})
}

func (db *Database) LoadUserSchema() {
	tablesTable := db.Schema.Tables["__tables__"]
	columnsTable := db.Schema.Tables["__columns__"]
	db.BoltDB.View(func(tx *bolt.Tx) error {
		tablesDescs := map[string]*TableDescriptor{}
		// Load all table descriptors.
		if err := tx.Bucket([]byte("__tables__")).ForEach(func(_ []byte, tableBytes []byte) error {
			tableRecord := tablesTable.RecordFromBytes(tableBytes)
			tableDesc := &TableDescriptor{
				Name:       tableRecord.GetField("name").StringVal,
				PrimaryKey: tableRecord.GetField("primary_key").StringVal,
				Columns:    make([]*ColumnDescriptor, 0),
			}
			tablesDescs[tableDesc.Name] = tableDesc
			return nil
		}); err != nil {
			return err
		}
		// Load all column descriptors; stick them on table descriptors.
		if err := tx.Bucket([]byte("__columns__")).ForEach(func(key []byte, columnBytes []byte) error {
			columnRecord := columnsTable.RecordFromBytes(columnBytes)
			columnSpec := ColumnFromRecord(columnRecord)
			tableDesc := tablesDescs[columnRecord.GetField("table_name").StringVal]
			tableDesc.Columns = append(tableDesc.Columns, columnSpec)
			return nil
		}); err != nil {
			return err
		}
		// Add them to the in-memory schema.
		for _, tableDesc := range tablesDescs {
			db.addTableDescriptor(tableDesc)
		}
		return nil
	})
}

// TODO: move to schema? idk
// buildTableDescriptor converts a CREATE TABLE AST node into a TableDescriptor.
// It also assigns column ids.
func (db *Database) buildTableDescriptor(create *CreateTable) *TableDescriptor {
	// find primary key
	var primaryKey string
	for _, column := range create.Columns {
		if column.PrimaryKey {
			primaryKey = column.Name
			break
		}
	}
	// Create table descriptor
	tableDesc := &TableDescriptor{
		Name:       create.Name,
		PrimaryKey: primaryKey,
		Columns:    make([]*ColumnDescriptor, len(create.Columns)),
	}
	// Create column descriptors
	for idx, parsedColumn := range create.Columns {
		// extract reference
		var reference *ColumnReference
		if parsedColumn.References != nil {
			reference = &ColumnReference{
				TableName: *parsedColumn.References,
			}
		}
		// build column spec
		columnSpec := &ColumnDescriptor{
			ID:               db.Schema.NextColumnID,
			Name:             parsedColumn.Name,
			ReferencesColumn: reference,
			Type:             NameToType[parsedColumn.TypeName],
		}
		// TODO: synchronize access to this
		db.Schema.NextColumnID++
		tableDesc.Columns[idx] = columnSpec
	}

	return tableDesc
}

// addTableDescriptor initializes the table's LiveQueryInfo
// and adds it to the schema.
func (db *Database) addTableDescriptor(table *TableDescriptor) {
	table.LiveQueryInfo = table.NewLiveQueryInfo() // def something weird about this
	go table.HandleEvents()
	// TODO: synchronize access to this
	db.Schema.Tables[table.Name] = table
}

func EmptySchema() *Schema {
	return &Schema{
		Tables: map[string]*TableDescriptor{},
	}
}

func (db *Database) AddBuiltinSchema() {
	// these never go in the on-disk __tables__ and __columns__ Bolt buckets
	// doing ids like this is kind of precarious...

	// TODO: could use string CREATE TABLEs here
	// parse 'em; call buildTableDescriptor to assign ids
	db.addTableDescriptor(&TableDescriptor{
		Name:       "__tables__",
		PrimaryKey: "name",
		Columns: []*ColumnDescriptor{
			{
				ID:   0,
				Name: "name",
				Type: TypeString,
			},
			{
				ID:   1,
				Name: "primary_key",
				Type: TypeString,
			},
		},
		IsBuiltin: true,
	})
	db.addTableDescriptor(&TableDescriptor{
		Name:       "__columns__",
		PrimaryKey: "id",
		Columns: []*ColumnDescriptor{
			{
				ID:   2,
				Name: "id",
				Type: TypeString, // TODO: switch to int when they work
			},
			{
				ID:   3,
				Name: "name",
				Type: TypeString,
			},
			{
				ID:   4,
				Name: "table_name",
				Type: TypeString,
				ReferencesColumn: &ColumnReference{
					TableName: "__tables__",
				},
			},
			{
				ID:   5,
				Name: "type",
				Type: TypeString,
			},
			{
				ID:   6,
				Name: "references", // TODO: this is a keyword. rename to "references_table"
				Type: TypeString,
			},
		},
		IsBuiltin: true,
	})
	db.addTableDescriptor(&TableDescriptor{
		Name:       "__record_listeners__",
		PrimaryKey: "id",
		Columns: []*ColumnDescriptor{
			{
				ID:   7,
				Name: "id",
				Type: TypeString,
			},
			{
				ID:   8,
				Name: "connection_id",
				Type: TypeString,
			},
			{
				ID:   9,
				Name: "channel_id",
				Type: TypeString,
			},
			{
				ID:   10,
				Name: "table_name",
				Type: TypeString,
				ReferencesColumn: &ColumnReference{
					TableName: "__tables__",
				},
			},
			{
				ID:   11,
				Name: "pk_value",
				Type: TypeString,
			},
			{
				ID:   12,
				Name: "query_path",
				Type: TypeString,
			},
		},
		IsBuiltin: true,
	})
	db.Schema.NextColumnID = 13 // ugh magic numbers.
}

// TODO: __connections__, __channels__, __whole_table_listeners__, __filtered_table_listeners__
