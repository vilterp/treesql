package treesql

import (
	"bytes"
	"fmt"
)

// TODO: rewrite with pretty printer lib

type NodeFormatter interface {
	Format() string
}

func (n *Statement) Format() string {
	if n.CreateTable != nil {
		return n.CreateTable.Format()
	}
	if n.Insert != nil {
		return n.Insert.Format()
	}
	if n.Select != nil {
		return n.Select.Format()
	}
	if n.Update != nil {
		return n.Update.Format()
	}
	panic(fmt.Sprintf("unknown %v", n))
}

func (n *CreateTable) Format() string {
	buf := bytes.NewBufferString("CREATETABLE ")
	buf.WriteString(n.Name)
	buf.WriteString(" (")
	for idx, col := range n.Columns {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(col.Name)
		buf.WriteString(" ")
		buf.WriteString(col.TypeName)
		if col.PrimaryKey {
			buf.WriteString(" PRIMARYKEY")
		}
		if col.References != nil {
			buf.WriteString(" REFERENCESTABLE ")
			buf.WriteString(*col.References)
		}
	}
	buf.WriteString(")")
	return buf.String()
}

func (n *Select) Format() string {
	buf := bytes.NewBufferString("")
	if n.Many {
		buf.WriteString("MANY ")
	} else {
		buf.WriteString("ONE ")
	}
	buf.WriteString(n.Table)
	if n.Where != nil {
		buf.WriteString(" WHERE ")
		buf.WriteString(n.Where.ColumnName)
		buf.WriteString(" = ")
		buf.WriteString(fmt.Sprintf(`%#v`, n.Where.Value))
	}
	buf.WriteString(" { ")
	for idx, selection := range n.Selections {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(selection.Name)
		if selection.SubSelect != nil {
			buf.WriteString(": ")
			sel := selection.SubSelect.Format()
			buf.WriteString(sel)
		}
	}
	buf.WriteString(" }")
	return buf.String()
}

func (n *Update) Format() string {
	return fmt.Sprintf(
		"UPDATE %s SET %s = %#v WHERE %s = %#v",
		n.Table, n.ColumnName, n.Value, n.WhereColumnName, n.EqualsValue,
	)
}

func (n *Insert) Format() string {
	buf := bytes.NewBufferString("INSERT INTO ")
	buf.WriteString(n.Table)
	buf.WriteString(" VALUES ")
	buf.WriteString("(")
	for idx, value := range n.Values {
		if idx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%#v", value))
	}
	buf.WriteString(")")
	return buf.String()
}
