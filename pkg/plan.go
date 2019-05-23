package treesql

import (
	"fmt"

	"github.com/vilterp/treesql/pkg/parse"
)

func Plan(ast parse.Statement, schema *Schema) (*Statement, error) {
	if ast.Select != nil {
		sel, err := transformSelect(ast.Select, schema)
		if err != nil {
			return nil, err
		}
		return &Statement{
			Select: sel,
		}, nil
	}
	if ast.CreateTable != nil {
		return &Statement{
			CreateTable: transformCreateTable(ast.CreateTable),
		}, nil
	}
	if ast.Update != nil {
		return &Statement{
			Update: transformUpdate(ast.Update),
		}, nil
	}
	if ast.Insert != nil {
		return &Statement{
			Insert: transformInsert(ast.Insert),
		}, nil
	}
	panic(fmt.Sprintf("unknown statement type: %#v", ast))
}

func transformSelect(sel *parse.Select, schema *Schema) (*Select, error) {
	sels, err := expandSelections(sel.Selections, schema)
	if err != nil {
		return nil, err
	}
	return &Select{
		Many:       sel.Many,
		Table:      sel.Table,
		Live:       sel.Live,
		Where:      transformWhere(sel.Where),
		Selections: sels,
	}, nil
}

func expandSelections(soss []*parse.SelectionOrStar, schema *Schema) ([]*Selection, error) {
	hasStar, selections, err := validateStar(soss)
	if err != nil {
		return nil, err
	}
	if hasStar {
		return expandStar(selections, schema), nil
	}
	XXX
}

// validateStar returns
// - true and selections if it has a star
// - false if it doesn't have a star
// - an error if there's a star and named columns
func validateStar(soss []*parse.SelectionOrStar) (bool, []*Selection, error) {
	hasStar := false
	var sels []*Selection
	for _, sos := range soss {
		if sos.Star {
			hasStar = true
		} else if sos.Selection != nil {
			if sos.Selection.SubSelect != nil {
				sels = append(sels, sos.Selection)
				continue
			} else {
				if hasStar {
					return true, nil, fmt.Errorf("if there's a *, only subselects are allowed; no named columns")
				}
			}
		}
	}
	return hasStar, sels, nil
}

func transformSelection(s *parse.Selection, schema *Schema) (*Selection, error) {
	var ss *Select
	var err error
	if s.SubSelect != nil {
		ss, err = transformSelect(s.SubSelect, schema)
		if err != nil {
			return nil, err
		}
	}
	return &Selection{
		Name:      s.Name,
		SubSelect: ss,
	}, nil
}

func expandStar(sels []*Selection, schema *Schema) []*Selection {
	XXXX
}

func transformWhere(w *parse.Where) *Where {
	if w == nil {
		return nil
	}
	return &Where{
		ColumnName: w.ColumnName,
		Value:      w.Value,
	}
}

func transformInsert(ins *parse.Insert) *Insert {
	return &Insert{
		Table:  ins.Table,
		Values: ins.Values,
	}
}

func transformUpdate(upd *parse.Update) *Update {
	return &Update{
		Table:           upd.Table,
		Value:           upd.Value,
		ColumnName:      upd.ColumnName,
		EqualsValue:     upd.EqualsValue,
		WhereColumnName: upd.WhereColumnName,
	}
}

func transformCreateTable(ct *parse.CreateTable) *CreateTable {
	var cols []*CreateTableColumn
	for _, col := range ct.Columns {
		cols = append(cols, &CreateTableColumn{
			Name:       col.Name,
			PrimaryKey: col.PrimaryKey,
			References: col.References,
			TypeName:   col.TypeName,
		})
	}
	return &CreateTable{
		Name:    ct.Name,
		Columns: cols,
	}
}
